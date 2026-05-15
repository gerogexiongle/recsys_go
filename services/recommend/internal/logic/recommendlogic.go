package logic

import (
	"context"

	"recsys_go/pkg/featurestore"
	"recsys_go/pkg/kafkapush"
	"recsys_go/pkg/recsyskit"
	"recsys_go/pkg/recsyskit/transporthttp"
	"recsys_go/services/recommend/internal/centerconfig"
	"recsys_go/services/recommend/internal/pipeline"
	"recsys_go/services/recommend/internal/recall"
)

// Recommend wires recall → merge → filter → rank → show (center or funnel mode).
type Recommend struct {
	Pipeline *recsyskit.Pipeline
	Features featurestore.Fetcher
	Funnel   *recsyskit.FunnelLibrary
	Center   *centerconfig.CenterBundle
	Recall   *recall.Registry
	AlgoKafka *kafkapush.Pool
	centerPL *pipeline.Center
}

func NewRecommend(rank recsyskit.RankClient, feat featurestore.Fetcher) *Recommend {
	return &Recommend{
		Pipeline: &recsyskit.Pipeline{Rank: rank},
		Features: feat,
	}
}

func NewRecommendFunnel(rank recsyskit.RankClient, feat featurestore.Fetcher, funnel *recsyskit.FunnelLibrary, reg *recall.Registry) *Recommend {
	return &Recommend{
		Pipeline: &recsyskit.Pipeline{Rank: rank},
		Features: feat,
		Funnel:   funnel,
		Recall:   reg,
	}
}

func NewRecommendCenter(rank recsyskit.RankClient, feat featurestore.Fetcher, center *centerconfig.CenterBundle, reg *recall.Registry) *Recommend {
	r := &Recommend{
		Pipeline: &recsyskit.Pipeline{Rank: rank},
		Features: feat,
		Center:   center,
		Recall:   reg,
	}
	r.centerPL = &pipeline.Center{Features: feat, Center: center, Recall: reg, Rank: rank}
	return r
}

func (l *Recommend) Handle(ctx context.Context, req *transporthttp.RecommendRequestJSON) (*transporthttp.RecommendResponseJSON, error) {
	rctx := recsyskit.RequestContext{
		UUID:            req.UUID,
		UserID:          req.UserID,
		Section:         req.Section,
		ExpIDs:          append([]int32(nil), req.ExpIDs...),
		DisablePersonal: req.DisablePersonal,
		DeviceID:        req.DeviceID,
		TerminalModel:   req.TerminalModel,
		OSType:          req.OS,
		UserGroup:       req.UserGroup,
	}

	if l.centerPL != nil {
		result, err := l.centerPL.Run(ctx, req, rctx)
		if err != nil {
			return nil, err
		}
		if result != nil {
			l.pushAlgoLog(req, rctx, result.Items)
			return result.Resp, nil
		}
	}
	if l.Funnel != nil && l.Recall != nil {
		resp, items, err := l.handleFunnel(ctx, req, rctx)
		if err != nil {
			return nil, err
		}
		l.pushAlgoLog(req, rctx, items)
		return resp, nil
	}
	stub := []recsyskit.ItemInfo{
		{ID: 10001, RecallType: "stub_hot"},
		{ID: 10002, RecallType: "stub_hot"},
		{ID: 10003, RecallType: "stub_hot"},
	}
	items, err := l.Pipeline.Run(ctx, rctx, stub)
	if err != nil {
		return nil, err
	}
	resp := buildRecommendResponse(req, items, req.RetCount)
	l.pushAlgoLog(req, rctx, items)
	return resp, nil
}

func (l *Recommend) handleFunnel(ctx context.Context, req *transporthttp.RecommendRequestJSON, rctx recsyskit.RequestContext) (*transporthttp.RecommendResponseJSON, []recsyskit.ItemInfo, error) {
	prof := l.Funnel.ResolveFunnel(rctx.ExpIDs, rctx.UserGroup)
	if prof == nil {
		resp, items, err := l.stubRankResponse(ctx, req, rctx)
		return resp, items, err
	}
	exclusive, main := prof.ResolvedRecallLists(rctx.ExpIDs)
	exclusiveBatches, err := l.runRules(ctx, rctx, exclusive)
	if err != nil {
		return nil, nil, err
	}
	mainBatches, err := l.runRules(ctx, rctx, main)
	if err != nil {
		return nil, nil, err
	}
	merged := recsyskit.MergeRecallLanes(exclusiveBatches, mainBatches, prof.AllMergeNum)
	merged, _ = l.markAndDropNoPortrait(ctx, merged)
	rctx = l.loadExposure(ctx, rctx)
	merged = recsyskit.ApplyFilterPolicies(rctx, prof.ResolvedFilterPolicies(rctx.ExpIDs), merged)
	if len(merged) == 0 {
		return &transporthttp.RecommendResponseJSON{UserID: req.UserID}, nil, nil
	}
	ret := effectiveRetCount(req, prof.FinalRetCount)
	return l.rankAndShowFunnel(ctx, req, rctx, merged, prof.ResolvedShowControl(rctx.ExpIDs), ret)
}

func (l *Recommend) stubRankResponse(ctx context.Context, req *transporthttp.RecommendRequestJSON, rctx recsyskit.RequestContext) (*transporthttp.RecommendResponseJSON, []recsyskit.ItemInfo, error) {
	stub := []recsyskit.ItemInfo{
		{ID: 10001, RecallType: "stub_hot"},
		{ID: 10002, RecallType: "stub_hot"},
		{ID: 10003, RecallType: "stub_hot"},
	}
	items, err := l.Pipeline.Run(ctx, rctx, stub)
	if err != nil {
		return nil, nil, err
	}
	return buildRecommendResponse(req, items, req.RetCount), items, nil
}

func effectiveRetCount(req *transporthttp.RecommendRequestJSON, finalRet int) int32 {
	ret := req.RetCount
	if ret <= 0 && finalRet > 0 {
		ret = int32(finalRet)
	}
	return ret
}

func (l *Recommend) rankAndShowFunnel(ctx context.Context, req *transporthttp.RecommendRequestJSON, rctx recsyskit.RequestContext, merged []recsyskit.ItemInfo, show recsyskit.ShowControlCfg, ret int32) (*transporthttp.RecommendResponseJSON, []recsyskit.ItemInfo, error) {
	if ret <= 0 {
		ret = int32(len(merged))
	}
	mreq := &recsyskit.MultiRankRequest{
		Ctx:          rctx,
		PreRankTrunc: 0,
		RankTrunc:    0,
		RankProfile:  "",
		Groups: []recsyskit.ItemGroup{{
			Name:     "Main",
			ItemIDs:  recsyskitIDs(merged),
			RetCount: ret,
		}},
	}
	resp, err := l.Pipeline.Rank.MultiRank(ctx, mreq)
	if err != nil {
		return nil, nil, err
	}
	items := merged
	if resp != nil && len(resp.Groups) > 0 && len(resp.Groups[0].Items) > 0 {
		items = reorderByRankGeneric(merged, resp.Groups[0].Items)
	}
	items = recsyskit.ApplyShowControl(show, items)
	return buildRecommendResponse(req, items, ret), items, nil
}

func (l *Recommend) runRules(ctx context.Context, rctx recsyskit.RequestContext, rules []recsyskit.RecallMergeRule) ([][]recsyskit.ItemInfo, error) {
	var batches [][]recsyskit.ItemInfo
	for _, rule := range rules {
		raw, err := l.Recall.Lane(ctx, rctx, rule)
		if err != nil {
			return nil, err
		}
		if rule.UseTopKIndex > 0 && len(raw) > rule.UseTopKIndex {
			raw = raw[:rule.UseTopKIndex]
		}
		raw, err = l.markAndDropNoPortrait(ctx, raw)
		if err != nil {
			return nil, err
		}
		mergeMax := rule.MergeMaxNum
		if mergeMax <= 0 {
			mergeMax = rule.RecallNum
		}
		batch := recsyskit.ApplySampleFoldAndCap(raw, rule.SampleFold, mergeMax)
		batches = append(batches, batch)
	}
	return batches, nil
}

func (l *Recommend) loadExposure(ctx context.Context, rctx recsyskit.RequestContext) recsyskit.RequestContext {
	if l.Features == nil || l.Features == featurestore.NoOp {
		return rctx
	}
	st, ok := l.Features.(featurestore.StrategyFetcher)
	if !ok {
		return rctx
	}
	raw, miss, err := st.FilterExposureJSON(ctx)
	if err != nil || miss {
		return rctx
	}
	if m := featurestore.ParseExposureJSON(raw, miss); len(m) > 0 {
		rctx.Exposure = make(map[recsyskit.ItemID]int, len(m))
		for id, c := range m {
			rctx.Exposure[recsyskit.ItemID(id)] = c
		}
	}
	return rctx
}

func (l *Recommend) markAndDropNoPortrait(ctx context.Context, items []recsyskit.ItemInfo) ([]recsyskit.ItemInfo, error) {
	marked, err := featurestore.MarkItemPortraits(ctx, l.Features, items)
	if err != nil {
		return items, err
	}
	return featurestore.DropWithoutPortrait(marked), nil
}

func recsyskitIDs(items []recsyskit.ItemInfo) []recsyskit.ItemID {
	out := make([]recsyskit.ItemID, len(items))
	for i := range items {
		out[i] = items[i].ID
	}
	return out
}

func reorderByRankGeneric(items []recsyskit.ItemInfo, scores []recsyskit.ItemScores) []recsyskit.ItemInfo {
	index := make(map[recsyskit.ItemID]recsyskit.ItemInfo, len(items))
	for _, it := range items {
		index[it.ID] = it
	}
	var out []recsyskit.ItemInfo
	seen := make(map[recsyskit.ItemID]struct{}, len(items))
	for _, sc := range scores {
		if it, ok := index[sc.ItemID]; ok {
			it.Score = float64(sc.RankScore)
			out = append(out, it)
			seen[it.ID] = struct{}{}
		}
	}
	for _, it := range items {
		if _, ok := seen[it.ID]; !ok {
			out = append(out, it)
		}
	}
	return out
}

func buildRecommendResponse(req *transporthttp.RecommendRequestJSON, items []recsyskit.ItemInfo, capCount int32) *transporthttp.RecommendResponseJSON {
	out := &transporthttp.RecommendResponseJSON{UserID: req.UserID}
	limit := len(items)
	if capCount > 0 && int(capCount) < limit {
		limit = int(capCount)
	}
	for i := 0; i < limit; i++ {
		it := items[i]
		out.ItemIDs = append(out.ItemIDs, int64(it.ID))
		out.Recall = append(out.Recall, transporthttp.ItemRecallJSON{
			ItemID:     int64(it.ID),
			RecallType: it.RecallType,
		})
	}
	return out
}
