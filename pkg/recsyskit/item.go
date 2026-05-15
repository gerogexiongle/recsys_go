// Package recsyskit holds generic recommendation primitives.
// Business-specific names (e.g. "map") are normalized to "item" here.
package recsyskit

// ItemID is a generic content identifier in the recommendation domain.
type ItemID int64

// ItemInfo carries per-candidate state through recall → filter → rank → show.
type ItemInfo struct {
	ID         ItemID
	RecallType string
	Score      float64
	Extra      map[string]string
}

// ItemGroup is a named bucket of candidates (e.g. main lane vs exclusive recall lane).
type ItemGroup struct {
	Name     string
	ItemIDs  []ItemID
	RetCount int32
}

// ItemScores holds model scores for one item.
type ItemScores struct {
	ItemID       ItemID
	PreRankScore float32
	RankScore    float32
	ReRankScore  float32
}

// RankedItemGroup is the ranked output for one group.
type RankedItemGroup struct {
	Name  string
	Items []ItemScores
}

// ExpInfo mirrors experiment ids selected by the ranker.
type ExpInfo struct {
	PreRankExpID int32
	RankExpID    int32
	ReRankExpID  int32
}
