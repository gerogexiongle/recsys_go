package recsyskit

import "context"

// RankClient calls the ranking service using generic item vocabulary.
type RankClient interface {
	MultiRank(ctx context.Context, req *MultiRankRequest) (*MultiRankResponse, error)
}

// MultiRankRequest is the generic counterpart of legacy multi-lane rank requests.
type MultiRankRequest struct {
	Ctx    RequestContext
	Groups []ItemGroup
	// PreRankTrunc / RankTrunc mirror C++ coarse FM head and fine-rank head (per-request AB); 0 = use rank service yaml.
	PreRankTrunc int32
	RankTrunc    int32
	// RankProfile selects rank RankProfiles[name] (精排 / 不同模型 AB).
	RankProfile string
}

// MultiRankResponse carries ranked groups back to the center service.
type MultiRankResponse struct {
	UUID   string
	UserID int64
	Exp    ExpInfo
	Groups []RankedItemGroup
}
