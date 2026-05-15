package pipeline

import (
	"context"

	"recsys_go/pkg/featurestore"
	"recsys_go/pkg/recsyskit"
	"recsys_go/pkg/recsyskit/transporthttp"
	"recsys_go/services/recommend/internal/centerconfig"
	"recsys_go/services/recommend/internal/filter"
	"recsys_go/services/recommend/internal/merge"
	"recsys_go/services/recommend/internal/recall"
	"recsys_go/services/recommend/internal/show"
)

// Center runs C++-order: recall (exclusive separate) → filter → rank main → show (ForcedInsert / Homogen).
type Center struct {
	Features featurestore.Fetcher
	Center   *centerconfig.CenterBundle
	Recall   *recall.Registry
	Rank     recsyskit.RankClient
}

func (p *Center) Run(ctx context.Context, req *transporthttp.RecommendRequestJSON, rctx recsyskit.RequestContext) (*transporthttp.RecommendResponseJSON, error) {
	if p.Center == nil || p.Center.Recall == nil || p.Recall == nil {
		return nil, nil
	}
	userFeat, _ := p.Features.UserJSON(ctx, rctx.UserID)
	rctx.UserFeat = userFeat
	rctx.UserGroup = recall.ResolveUserGroup(rctx.UserGroup, userFeat)

	prof := p.Center.Recall.ResolveRecall(rctx.ExpIDs, rctx.UserGroup)
	if prof == nil {
		return nil, nil
	}
	exclusiveRules, mainRules := prof.ResolvedRecallLists(rctx.ExpIDs)
	exclusive := p.runExclusiveLanes(ctx, rctx, exclusiveRules)
	mainBatches, err := p.runRecallLanes(ctx, rctx, mainRules)
	if err != nil {
		return nil, err
	}
	mainMerged := merge.MainOnly(mainBatches, prof.AllMergeNum)

	rctx = p.loadExposure(ctx, rctx)
	if p.Center.Filter != nil {
		fg := p.Center.Filter.ResolveFilter(rctx.ExpIDs, rctx.UserGroup)
		if fg != nil {
			rules, feats := fg.ResolvedRuleAndFeature(rctx.ExpIDs)
			mainMerged = filter.ApplyRuleFilters(rctx, rules, mainMerged)
			mainMerged = filter.ApplyFeatureFilters(rctx, feats, mainMerged)
			exclusive = filter.ApplyToExclusivePool(rctx, rules, feats, exclusive)
			mainMerged = centerconfig.CapKeepItemNum(fg.KeepItemNum, mainMerged)
		}
	}
	if len(mainMerged) == 0 && len(exclusive) == 0 {
		return &transporthttp.RecommendResponseJSON{UserID: req.UserID}, nil
	}
	ret := req.RetCount
	if ret <= 0 && prof.FinalRetCount > 0 {
		ret = int32(prof.FinalRetCount)
	}
	return p.rankAndShow(ctx, req, rctx, mainMerged, exclusive, ret)
}

func (p *Center) runExclusiveLanes(ctx context.Context, rctx recsyskit.RequestContext, rules []recsyskit.RecallMergeRule) recsyskit.ExclusivePool {
	pool := make(recsyskit.ExclusivePool)
	for _, rule := range rules {
		raw, err := p.Recall.Lane(ctx, rctx, rule)
		if err != nil || len(raw) == 0 {
			continue
		}
		if rule.UseTopKIndex > 0 && len(raw) > rule.UseTopKIndex {
			raw = raw[:rule.UseTopKIndex]
		}
		marked, err := featurestore.MarkItemPortraits(ctx, p.Features, raw)
		if err != nil {
			continue
		}
		marked = featurestore.DropWithoutPortrait(marked)
		mergeMax := rule.MergeMaxNum
		if mergeMax <= 0 {
			mergeMax = rule.RecallNum
		}
		batch := recsyskit.ApplySampleFoldAndCap(marked, rule.SampleFold, mergeMax)
		if len(batch) > 0 {
			pool[rule.RecallType] = batch
		}
	}
	return pool
}

func (p *Center) runRecallLanes(ctx context.Context, rctx recsyskit.RequestContext, rules []recsyskit.RecallMergeRule) ([][]recsyskit.ItemInfo, error) {
	var batches [][]recsyskit.ItemInfo
	for _, rule := range rules {
		raw, err := p.Recall.Lane(ctx, rctx, rule)
		if err != nil {
			return nil, err
		}
		if rule.UseTopKIndex > 0 && len(raw) > rule.UseTopKIndex {
			raw = raw[:rule.UseTopKIndex]
		}
		marked, err := featurestore.MarkItemPortraits(ctx, p.Features, raw)
		if err != nil {
			return nil, err
		}
		marked = featurestore.DropWithoutPortrait(marked)
		mergeMax := rule.MergeMaxNum
		if mergeMax <= 0 {
			mergeMax = rule.RecallNum
		}
		batch := recsyskit.ApplySampleFoldAndCap(marked, rule.SampleFold, mergeMax)
		batches = append(batches, batch)
	}
	return batches, nil
}

func (p *Center) loadExposure(ctx context.Context, rctx recsyskit.RequestContext) recsyskit.RequestContext {
	if p.Features == nil || p.Features == featurestore.NoOp {
		return rctx
	}
	st, ok := p.Features.(featurestore.StrategyFetcher)
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

func (p *Center) rankAndShow(ctx context.Context, req *transporthttp.RecommendRequestJSON, rctx recsyskit.RequestContext, main []recsyskit.ItemInfo, exclusive recsyskit.ExclusivePool, ret int32) (*transporthttp.RecommendResponseJSON, error) {
	out := main
	if len(main) > 0 {
		if ret <= 0 {
			ret = int32(len(main))
		}
		ids := make([]recsyskit.ItemID, len(main))
		for i := range main {
			ids[i] = main[i].ID
		}
		resp, err := p.Rank.MultiRank(ctx, &recsyskit.MultiRankRequest{
			Ctx: rctx,
			Groups: []recsyskit.ItemGroup{{
				Name: "Main", ItemIDs: ids, RetCount: ret,
			}},
		})
		if err != nil {
			return nil, err
		}
		if resp != nil && len(resp.Groups) > 0 && len(resp.Groups[0].Items) > 0 {
			out = reorderByRank(main, resp.Groups[0].Items)
		}
	}
	if p.Center.Show != nil {
		if sg := p.Center.Show.ResolveShow(rctx.ExpIDs, rctx.UserGroup); sg != nil {
			out = show.ApplyStrategies(ctx, p.Features, out, exclusive, sg.ResolvedStrategyList(rctx.ExpIDs))
		}
	}
	return buildResponse(req, out, ret), nil
}

func reorderByRank(items []recsyskit.ItemInfo, scores []recsyskit.ItemScores) []recsyskit.ItemInfo {
	index := make(map[recsyskit.ItemID]recsyskit.ItemInfo, len(items))
	for _, it := range items {
		index[it.ID] = it
	}
	var out []recsyskit.ItemInfo
	seen := make(map[recsyskit.ItemID]struct{})
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

func buildResponse(req *transporthttp.RecommendRequestJSON, items []recsyskit.ItemInfo, capCount int32) *transporthttp.RecommendResponseJSON {
	out := &transporthttp.RecommendResponseJSON{UserID: req.UserID}
	limit := len(items)
	if capCount > 0 && int(capCount) < limit {
		limit = int(capCount)
	}
	for i := 0; i < limit; i++ {
		it := items[i]
		out.ItemIDs = append(out.ItemIDs, int64(it.ID))
		out.Recall = append(out.Recall, transporthttp.ItemRecallJSON{
			ItemID: int64(it.ID), RecallType: it.RecallType,
		})
	}
	return out
}
