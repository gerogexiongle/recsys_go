package featurestore

import "fmt"

// Key layout mirrors C++ online_map_center: profile HASH fields vs side-channel proto types.
//
// Profile (FM / rank only) — missing key => no user/item portrait; rank uses placeholder sparse.
const (
	DefaultUserFeatKeyPattern = "recsysgo:feat:user:%d"
	DefaultItemFeatKeyPattern = "recsysgo:feat:item:%d"
)

// Filter strategy keys — each FilterType reads its own namespace (adjust data without touching feat JSON).
// Missing key semantics (align C++ GetProtoDataEmpty):
//   - exposure user key missing => empty exposure map => LiveExposure does not filter
//   - featureless item key missing => item treated as having features => FeatureLess keeps item
//   - label item key missing => LabelTypeWhiteList cannot match label
const (
	DefaultUserExposureKeyPattern     = "recsysgo:filter:exposure:user:%d"
	DefaultItemFeatureLessKeyPattern  = "recsysgo:filter:featureless:item:%d"
	DefaultItemLabelKeyPattern        = "recsysgo:filter:label:item:%d"
)

// KeyPatterns builds Redis STRING keys for profile JSON.
type KeyPatterns struct {
	UserFeat string
	ItemFeat string
}

// StrategyKeyPatterns builds Redis STRING keys for non-profile strategies.
type StrategyKeyPatterns struct {
	UserExposure    string
	ItemFeatureLess string
	ItemLabel       string
}

func DefaultKeyPatterns() KeyPatterns {
	return KeyPatterns{UserFeat: DefaultUserFeatKeyPattern, ItemFeat: DefaultItemFeatKeyPattern}
}

func DefaultStrategyKeyPatterns() StrategyKeyPatterns {
	return StrategyKeyPatterns{
		UserExposure:    DefaultUserExposureKeyPattern,
		ItemFeatureLess: DefaultItemFeatureLessKeyPattern,
		ItemLabel:       DefaultItemLabelKeyPattern,
	}
}

func (p KeyPatterns) UserKey(uin int64) string  { return fmt.Sprintf(p.UserFeat, uin) }
func (p KeyPatterns) ItemKey(itemID int64) string { return fmt.Sprintf(p.ItemFeat, itemID) }

func (p StrategyKeyPatterns) UserExposureKey(uin int64) string {
	return fmt.Sprintf(p.UserExposure, uin)
}

func (p StrategyKeyPatterns) ItemFeatureLessKey(itemID int64) string {
	return fmt.Sprintf(p.ItemFeatureLess, itemID)
}

func (p StrategyKeyPatterns) ItemLabelKey(itemID int64) string {
	return fmt.Sprintf(p.ItemLabel, itemID)
}

// FutureKeyKinds documents C++-style keys for later adapters (invert / material).
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
