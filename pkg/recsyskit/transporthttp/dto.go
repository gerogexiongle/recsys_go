// Package transporthttp exposes JSON DTOs and HTTP RankClient for services without protobuf codegen.
package transporthttp

// MultiRankRequestJSON is the wire shape for /v1/rank/multi (generic item naming).
type MultiRankRequestJSON struct {
	UUID            string              `json:"uuid"`
	UserID          int64               `json:"user_id"`
	Section         int32               `json:"section"`
	ExpIDs          []int32             `json:"exp_ids"`
	DisablePersonal int32               `json:"disable_personal"`
	DeviceID        string              `json:"device_id"`
	TerminalModel   string              `json:"terminal_model"`
	OS              string              `json:"os_type"`
	ItemGroups      []ItemGroupJSON     `json:"item_groups"`
	PreRankTrunc    int32               `json:"pre_rank_trunc,omitempty"`
	RankTrunc       int32               `json:"rank_trunc,omitempty"`
	RankProfile     string              `json:"rank_profile,omitempty"`
}

// ItemGroupJSON is a named candidate list for ranking.
type ItemGroupJSON struct {
	Name     string  `json:"name"`
	ItemIDs  []int64 `json:"item_ids"`
	RetCount int32   `json:"ret_count"`
}

// MultiRankResponseJSON mirrors rank output with item naming.
type MultiRankResponseJSON struct {
	UUID        string              `json:"uuid"`
	UserID      int64               `json:"user_id"`
	Exp         ExpInfoJSON         `json:"exp"`
	RankedGroups []RankedGroupJSON  `json:"ranked_groups"`
}

// ExpInfoJSON carries resolved experiment ids.
type ExpInfoJSON struct {
	PreRankExpID int32 `json:"pre_rank_exp_id"`
	RankExpID    int32 `json:"rank_exp_id"`
	ReRankExpID  int32 `json:"re_rank_exp_id"`
}

// RankedGroupJSON is one ranked lane.
type RankedGroupJSON struct {
	Name       string           `json:"name"`
	ItemScores []ItemScoresJSON `json:"item_scores"`
}

// ItemScoresJSON holds per-item model scores.
type ItemScoresJSON struct {
	ItemID       int64   `json:"item_id"`
	PreRankScore float32 `json:"pre_rank_score"`
	RankScore    float32 `json:"rank_score"`
	ReRankScore  float32 `json:"re_rank_score"`
}

// RecommendRequestJSON is the generic recommend entry (item-based).
type RecommendRequestJSON struct {
	UUID            string  `json:"uuid"`
	UserID          int64   `json:"user_id"`
	Section         int32   `json:"section"`
	ExpIDs          []int32 `json:"exp_ids"`
	RetCount        int32   `json:"ret_count"`
	DisablePersonal int32   `json:"disable_personal"`
	DeviceID        string  `json:"device_id"`
	TerminalModel   string  `json:"terminal_model"`
	OS              string  `json:"os_type"`
	UserGroup       string  `json:"user_group,omitempty"`
}

// RecommendResponseJSON returns ranked item ids and optional debug fields.
type RecommendResponseJSON struct {
	UserID   int64              `json:"user_id"`
	ItemIDs  []int64            `json:"item_ids"`
	Recall   []ItemRecallJSON   `json:"recall,omitempty"`
}

// ItemRecallJSON explains why an item entered the funnel.
type ItemRecallJSON struct {
	ItemID     int64  `json:"item_id"`
	RecallType string `json:"recall_type"`
}
