package recsyskit

import "context"

// RecallStage returns candidate items for downstream stages.
type RecallStage interface {
	Name() string
	Recall(ctx context.Context, rctx RequestContext) ([]ItemInfo, error)
}

// FilterStage removes or annotates items before ranking.
type FilterStage interface {
	Name() string
	Filter(ctx context.Context, rctx RequestContext, items []ItemInfo) ([]ItemInfo, error)
}

// ShowControlStage reorders or caps the final list after ranking.
type ShowControlStage interface {
	Name() string
	Apply(ctx context.Context, rctx RequestContext, items []ItemInfo) ([]ItemInfo, error)
}

// Pipeline wires generic stages; vertical-specific logic lives in plugins/adapters.
type Pipeline struct {
	Recall      []RecallStage
	Filter      []FilterStage
	Rank        RankClient
	ShowControl ShowControlStage
}

// Run executes recall → filter → rank (multi-group) → show in order.
// For a minimal first version, callers may pass a single "Main" group.
func (p *Pipeline) Run(ctx context.Context, rctx RequestContext, main []ItemInfo) ([]ItemInfo, error) {
	items := append([]ItemInfo(nil), main...)
	for _, s := range p.Recall {
		more, err := s.Recall(ctx, rctx)
		if err != nil {
			return nil, err
		}
		items = append(items, more...)
	}
	for _, s := range p.Filter {
		var err error
		items, err = s.Filter(ctx, rctx, items)
		if err != nil {
			return nil, err
		}
	}
	if p.Rank != nil && len(items) > 0 {
		groups := []ItemGroup{{Name: "Main", ItemIDs: ids(items), RetCount: int32(len(items))}}
		resp, err := p.Rank.MultiRank(ctx, &MultiRankRequest{Ctx: rctx, Groups: groups, PreRankTrunc: 0, RankTrunc: 0, RankProfile: ""})
		if err != nil {
			return nil, err
		}
		if resp != nil && len(resp.Groups) > 0 {
			items = reorderByRank(items, resp.Groups[0].Items)
		}
	}
	if p.ShowControl != nil {
		var err error
		items, err = p.ShowControl.Apply(ctx, rctx, items)
		if err != nil {
			return nil, err
		}
	}
	return items, nil
}

func ids(items []ItemInfo) []ItemID {
	out := make([]ItemID, len(items))
	for i := range items {
		out[i] = items[i].ID
	}
	return out
}

func reorderByRank(items []ItemInfo, scores []ItemScores) []ItemInfo {
	if len(scores) == 0 {
		return items
	}
	index := make(map[ItemID]ItemInfo, len(items))
	for _, it := range items {
		index[it.ID] = it
	}
	out := make([]ItemInfo, 0, len(scores))
	seen := make(map[ItemID]struct{}, len(items))
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
