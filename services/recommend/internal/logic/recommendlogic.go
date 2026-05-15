package logic

import (
	"context"

	"recsys_go/pkg/featurestore"
	"recsys_go/pkg/recsyskit"
	"recsys_go/pkg/recsyskit/transporthttp"
	"recsys_go/services/recommend/internal/centerconfig"
	"recsys_go/services/recommend/internal/recall"
)

// Recommend wires optional Config_Recall-style funnel (multi-recall → merge → filter → rank → show). Sorting trunc/model AB lives in rank RankExpConf.json.
type Recommend struct {
	Pipeline *recsyskit.Pipeline
	Features featurestore.Fetcher
	Funnel   *recsyskit.FunnelLibrary
	Center   *centerconfig.CenterBundle
	Recall   *recall.Registry
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
	return &Recommend{
		Pipeline: &recsyskit.Pipeline{Rank: rank},
		Features: feat,
		Center:   center,
		Recall:   reg,
	}
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
		UserGroup: req.UserGroup,
	}

	if l.Center != nil && l.Recall != nil {
		return l.handleCenter(ctx, req, rctx)
	}
	if l.Funnel != nil && l.Recall != nil {
		return l.handleFunnel(ctx, req, rctx)
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
	return buildRecommendResponse(req, items, req.RetCount), nil
}

func (l *Recommend) handleFunnel(ctx context.Context, req *transporthttp.RecommendRequestJSON, rctx recsyskit.RequestContext) (*transporthttp.RecommendResponseJSON, error) {
	prof := l.Funnel.ResolveFunnel(rctx.ExpIDs, rctx.UserGroup)
	if prof == nil {
		return l.stubRankResponse(ctx, req, rctx)
	}
	exclusive, main := prof.ResolvedRecallLists(rctx.ExpIDs)
	exclusiveBatches, err := l.runRules(ctx, rctx, exclusive)
	if err != nil {
		return nil, err
	}
	mainBatches, err := l.runRules(ctx, rctx, main)
	if err != nil {
		return nil, err
	}
	merged := recsyskit.MergeRecallLanes(exclusiveBatches, mainBatches, prof.AllMergeNum)
	merged, rctx = l.enrichFromRedis(ctx, rctx, merged)
	merged = recsyskit.ApplyFilterPolicies(rctx, prof.ResolvedFilterPolicies(rctx.ExpIDs), merged)
	if len(merged) == 0 {
		return &transporthttp.RecommendResponseJSON{UserID: req.UserID}, nil
	}
	ret := effectiveRetCount(req, prof.FinalRetCount)
	return l.rankAndShowFunnel(ctx, req, rctx, merged, prof.ResolvedShowControl(rctx.ExpIDs), ret)
}

func (l *Recommend) handleCenter(ctx context.Context, req *transporthttp.RecommendRequestJSON, rctx recsyskit.RequestContext) (*transporthttp.RecommendResponseJSON, error) {
	if l.Center.Recall == nil {
		return l.stubRankResponse(ctx, req, rctx)
	}
	prof := l.Center.Recall.ResolveRecall(rctx.ExpIDs, rctx.UserGroup)
	if prof == nil {
		return l.stubRankResponse(ctx, req, rctx)
	}
	exclusive, main := prof.ResolvedRecallLists(rctx.ExpIDs)
	exclusiveBatches, err := l.runRules(ctx, rctx, exclusive)
	if err != nil {
		return nil, err
	}
	mainBatches, err := l.runRules(ctx, rctx, main)
	if err != nil {
		return nil, err
	}
	merged := recsyskit.MergeRecallLanes(exclusiveBatches, mainBatches, prof.AllMergeNum)
	merged, rctx = l.enrichFromRedis(ctx, rctx, merged)
	if l.Center.Filter != nil {
		fg := l.Center.Filter.ResolveFilter(rctx.ExpIDs, rctx.UserGroup)
		if fg != nil {
			rules, feats := fg.ResolvedRuleAndFeature(rctx.ExpIDs)
			merged = centerconfig.ApplyRuleFilters(rctx, rules, merged)
			merged = centerconfig.ApplyFeatureFilters(rctx, feats, merged)
			merged = centerconfig.CapKeepItemNum(fg.KeepItemNum, merged)
		}
	}
	if len(merged) == 0 {
		return &transporthttp.RecommendResponseJSON{UserID: req.UserID}, nil
	}
	ret := effectiveRetCount(req, prof.FinalRetCount)
	return l.rankAndShowCenter(ctx, req, rctx, merged, ret)
}

func (l *Recommend) stubRankResponse(ctx context.Context, req *transporthttp.RecommendRequestJSON, rctx recsyskit.RequestContext) (*transporthttp.RecommendResponseJSON, error) {
	stub := []recsyskit.ItemInfo{
		{ID: 10001, RecallType: "stub_hot"},
		{ID: 10002, RecallType: "stub_hot"},
		{ID: 10003, RecallType: "stub_hot"},
	}
	items, err := l.Pipeline.Run(ctx, rctx, stub)
	if err != nil {
		return nil, err
	}
	return buildRecommendResponse(req, items, req.RetCount), nil
}

func effectiveRetCount(req *transporthttp.RecommendRequestJSON, finalRet int) int32 {
	ret := req.RetCount
	if ret <= 0 && finalRet > 0 {
		ret = int32(finalRet)
	}
	return ret
}

func (l *Recommend) rankAndShowFunnel(ctx context.Context, req *transporthttp.RecommendRequestJSON, rctx recsyskit.RequestContext, merged []recsyskit.ItemInfo, show recsyskit.ShowControlCfg, ret int32) (*transporthttp.RecommendResponseJSON, error) {
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
		return nil, err
	}
	items := merged
	if resp != nil && len(resp.Groups) > 0 && len(resp.Groups[0].Items) > 0 {
		items = reorderByRankGeneric(merged, resp.Groups[0].Items)
	}
	items = recsyskit.ApplyShowControl(show, items)
	return buildRecommendResponse(req, items, ret), nil
}

func (l *Recommend) rankAndShowCenter(ctx context.Context, req *transporthttp.RecommendRequestJSON, rctx recsyskit.RequestContext, merged []recsyskit.ItemInfo, ret int32) (*transporthttp.RecommendResponseJSON, error) {
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
		return nil, err
	}
	items := merged
	if resp != nil && len(resp.Groups) > 0 && len(resp.Groups[0].Items) > 0 {
		items = reorderByRankGeneric(merged, resp.Groups[0].Items)
	}
	if l.Center.Show != nil {
		sg := l.Center.Show.ResolveShow(rctx.ExpIDs, rctx.UserGroup)
		if sg != nil {
			items = centerconfig.ApplyShowStrategies(items, sg.ResolvedStrategyList(rctx.ExpIDs))
		}
	}
	return buildRecommendResponse(req, items, ret), nil
}

// enrichFromRedis loads profile feat + per-strategy Redis keys (C++ separate proto fields).
func (l *Recommend) enrichFromRedis(ctx context.Context, rctx recsyskit.RequestContext, items []recsyskit.ItemInfo) ([]recsyskit.ItemInfo, recsyskit.RequestContext) {
	if l.Features == nil || l.Features == featurestore.NoOp {
		if rctx.Exposure == nil {
			rctx.Exposure = demoItemExposure()
		}
		return items, rctx
	}
	ids := make([]int64, len(items))
	for i := range items {
		ids[i] = int64(items[i].ID)
	}
	cs, err := featurestore.LoadCenterSession(ctx, l.Features, rctx.UserID, ids)
	if err != nil {
		return items, rctx
	}
	if len(cs.Exposure) > 0 {
		rctx.Exposure = cs.Exposure
	}
	return cs.EnrichItems(items), rctx
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
		mergeMax := rule.MergeMaxNum
		if mergeMax <= 0 {
			mergeMax = rule.RecallNum
		}
		batch := recsyskit.ApplySampleFoldAndCap(raw, rule.SampleFold, mergeMax)
		batches = append(batches, batch)
	}
	return batches, nil
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

// demoItemExposure item-level fallback when Redis disabled (recsysgo:filter:exposure).
func demoItemExposure() map[recsyskit.ItemID]int {
	return map[recsyskit.ItemID]int{910005: 15}
}
