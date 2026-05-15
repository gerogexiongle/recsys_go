package featurestore

import "fmt"

// Profile (FM / rank / show when derivable) — one STRING JSON per entity.
const (
	DefaultUserFeatKeyPattern = "recsysgo:feat:user:%d"
	DefaultItemFeatKeyPattern = "recsysgo:feat:item:%d"
)

// Filter — one merged key per strategy namespace (item-level only in OSS demo).
// Missing key => strategy inactive (C++ GetProtoDataEmpty).
const (
	KeyFilterExposure    = "recsysgo:filter:exposure"    // JSON map item_id -> count
	KeyFilterFeatureLess = "recsysgo:filter:featureless" // JSON array of item ids
	KeyFilterLabel       = "recsysgo:filter:label"       // JSON map item_id -> label
)

// Recall — non-personalized lane = single list; CF = per-user list (C++ invert ZSET vs user CF).
const (
	KeyRecallLanePrefix   = "recsysgo:recall:lane:"       // + RecallType e.g. LiveRedirect
	DefaultRecallCFUserKey = "recsysgo:recall:cf:user:%d" // JSON [item_ids...]
)

type KeyPatterns struct {
	UserFeat string
	ItemFeat string
}

type StrategyKeyPatterns struct {
	FilterExposure    string
	FilterFeatureLess string
	FilterLabel       string
	RecallLanePrefix  string
	RecallCFUser      string
}

func DefaultKeyPatterns() KeyPatterns {
	return KeyPatterns{UserFeat: DefaultUserFeatKeyPattern, ItemFeat: DefaultItemFeatKeyPattern}
}

func DefaultStrategyKeyPatterns() StrategyKeyPatterns {
	return StrategyKeyPatterns{
		FilterExposure:    KeyFilterExposure,
		FilterFeatureLess: KeyFilterFeatureLess,
		FilterLabel:       KeyFilterLabel,
		RecallLanePrefix:  KeyRecallLanePrefix,
		RecallCFUser:      DefaultRecallCFUserKey,
	}
}

func (p KeyPatterns) UserKey(uin int64) string   { return fmt.Sprintf(p.UserFeat, uin) }
func (p KeyPatterns) ItemKey(itemID int64) string { return fmt.Sprintf(p.ItemFeat, itemID) }

func (p StrategyKeyPatterns) RecallLaneKey(lane string) string {
	return p.RecallLanePrefix + lane
}

func (p StrategyKeyPatterns) RecallCFUserKey(uin int64) string {
	return fmt.Sprintf(p.RecallCFUser, uin)
}

type FutureKeyKinds struct {
	InvertTagRecall  string
	MaterialSuppress string
}

func CppFutureKeys() FutureKeyKinds {
	return FutureKeyKinds{
		InvertTagRecall:  "online_tag_recall_%s_5pcts_%d",
		MaterialSuppress: "online_suppress_map",
	}
}
