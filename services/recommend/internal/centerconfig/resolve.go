package centerconfig

import "recsys_go/pkg/recsyskit"

func resolveMapRecommend[T any](lib []mapRecommendVariant[T], expIDs []int32, userGroup string, userGroupOf func(T) string) *T {
	if len(lib) == 0 {
		return nil
	}
	var v *mapRecommendVariant[T]
	for _, exp := range expIDs {
		for i := range lib {
			if lib[i].ExpID == exp {
				v = &lib[i]
				break
			}
		}
		if v != nil {
			break
		}
	}
	if v == nil {
		for i := range lib {
			if lib[i].ExpID == 0 {
				v = &lib[i]
				break
			}
		}
	}
	if v == nil {
		v = &lib[0]
	}
	if len(v.UserGroupList) == 0 {
		return nil
	}
	if userGroup != "" {
		for i := range v.UserGroupList {
			if userGroupOf(v.UserGroupList[i]) == userGroup {
				return &v.UserGroupList[i]
			}
		}
	}
	return &v.UserGroupList[0]
}

type mapRecommendVariant[T any] struct {
	ExpID          int32 `json:"exp_id"`
	UserGroupList []T   `json:"UserGroupList"`
}

// --- recall ---

// RecallUserGroup is one MapRecommend.UserGroupList row (Config_Recall subset).
type RecallUserGroup struct {
	UserGroup           string                      `json:"UserGroup"`
	AllMergeNum         int                         `json:"AllMergeNum"`
	ExclusiveRecallList []recsyskit.RecallMergeRule `json:"ExclusiveRecallList,omitempty"`
	RecallAndMergeList  []recsyskit.RecallMergeRule `json:"RecallAndMergeList"`
	FinalRetCount       int                         `json:"FinalRetCount,omitempty"`
	RecallAb            []recsyskit.RecallAbVariant `json:"RecallAb,omitempty"`
}

// RecallLibrary mirrors Config_Recall.MapRecommend (subset).
type RecallLibrary struct {
	MapRecommend []mapRecommendVariant[RecallUserGroup] `json:"MapRecommend"`
}

// ResolveRecall picks AB bucket then user group (same rules as funnel).
func (lib *RecallLibrary) ResolveRecall(expIDs []int32, userGroup string) *RecallUserGroup {
	if lib == nil {
		return nil
	}
	return resolveMapRecommend(lib.MapRecommend, expIDs, userGroup, func(g RecallUserGroup) string { return g.UserGroup })
}

// ResolvedRecallLists returns exclusive + main lanes (RecallAb or legacy).
func (g *RecallUserGroup) ResolvedRecallLists(expIDs []int32) (exclusive, main []recsyskit.RecallMergeRule) {
	if g == nil {
		return nil, nil
	}
	if len(g.RecallAb) == 0 {
		return g.ExclusiveRecallList, g.RecallAndMergeList
	}
	i := recsyskit.PickABVariantIndex(expIDs, len(g.RecallAb), func(j int) int32 { return g.RecallAb[j].ExpID })
	return g.RecallAb[i].ExclusiveRecallList, g.RecallAb[i].RecallAndMergeList
}

// --- filter (Config_Filter-style) ---

type RuleFilterStrategy struct {
	FilterType     string `json:"FilterType"`
	ValidityTime   int    `json:"ValidityTime,omitempty"`
	ExposureLimit  int    `json:"ExposureLimit,omitempty"`
	ForcedRatio    int    `json:"ForcedRatio,omitempty"`
}

type FeatureFilterStrategy struct {
	FilterType     string `json:"FilterType"`
	WhiteListLabel string `json:"WhiteListLabel,omitempty"`
}

type filterAbVariant struct {
	ExpID                     int32                   `json:"exp_id"`
	RuleFilterStrategyList    []RuleFilterStrategy    `json:"RuleFilterStrategyList,omitempty"`
	FeatureFilterStrategyList []FeatureFilterStrategy `json:"FeatureFilterStrategyList,omitempty"`
}

type filterUserGroup struct {
	UserGroup                 string                   `json:"UserGroup"`
	KeepItemNum               int                      `json:"KeepItemNum,omitempty"`
	RuleFilterStrategyList    []RuleFilterStrategy     `json:"RuleFilterStrategyList,omitempty"`
	FeatureFilterStrategyList []FeatureFilterStrategy  `json:"FeatureFilterStrategyList,omitempty"`
	FilterAb                  []filterAbVariant        `json:"FilterAb,omitempty"`
}

// FilterLibrary mirrors Config_Filter.MapRecommend (subset).
type FilterLibrary struct {
	MapRecommend []mapRecommendVariant[filterUserGroup] `json:"MapRecommend"`
}

// ResolveFilter picks variant + user group.
func (lib *FilterLibrary) ResolveFilter(expIDs []int32, userGroup string) *filterUserGroup {
	if lib == nil {
		return nil
	}
	return resolveMapRecommend(lib.MapRecommend, expIDs, userGroup, func(g filterUserGroup) string { return g.UserGroup })
}

// ResolvedRuleAndFeature returns strategy lists after optional FilterAb.
func (g *filterUserGroup) ResolvedRuleAndFeature(expIDs []int32) (rules []RuleFilterStrategy, feats []FeatureFilterStrategy) {
	if g == nil {
		return nil, nil
	}
	if len(g.FilterAb) == 0 {
		return g.RuleFilterStrategyList, g.FeatureFilterStrategyList
	}
	i := recsyskit.PickABVariantIndex(expIDs, len(g.FilterAb), func(j int) int32 { return g.FilterAb[j].ExpID })
	return g.FilterAb[i].RuleFilterStrategyList, g.FilterAb[i].FeatureFilterStrategyList
}

// --- show (Config_ShowControl-style) ---

type ForcedInsertRule struct {
	RecallType         string `json:"RecallType"`
	ForcedInsertCount  int    `json:"ForcedInsertCount"`
	ExtractMethod      string `json:"ExtractMethod,omitempty"`
}

type ShowStrategy struct {
	ShowControlType     string             `json:"ShowControlType"`
	Method              string             `json:"Method,omitempty"`
	TopNShowControl     int                `json:"TopNShowControl,omitempty"`
	RecallTypeList      string             `json:"RecallTypeList,omitempty"`
	ScoreControlFactor  float64            `json:"ScoreControlFactor,omitempty"`
	PageNum             int                `json:"PageNum,omitempty"`
	PageSize            int                `json:"PageSize,omitempty"`
	MMRConstant         float64            `json:"MMRConstant,omitempty"`
	MMRDimension        int                `json:"MMRDimension,omitempty"`
	ForcedInsert        []ForcedInsertRule `json:"ForcedInsert,omitempty"`
	Comment             string             `json:"Comment,omitempty"`
}

type showAbVariant struct {
	ExpID        int32          `json:"exp_id"`
	StrategyList []ShowStrategy `json:"StrategyList,omitempty"`
}

type showUserGroup struct {
	UserGroup    string           `json:"UserGroup"`
	StrategyList []ShowStrategy   `json:"StrategyList,omitempty"`
	ShowAb       []showAbVariant  `json:"ShowAb,omitempty"`
}

// ShowLibrary mirrors Config_ShowControl.MapRecommend (subset).
type ShowLibrary struct {
	MapRecommend []mapRecommendVariant[showUserGroup] `json:"MapRecommend"`
}

// ResolveShow picks variant + user group.
func (lib *ShowLibrary) ResolveShow(expIDs []int32, userGroup string) *showUserGroup {
	if lib == nil {
		return nil
	}
	return resolveMapRecommend(lib.MapRecommend, expIDs, userGroup, func(g showUserGroup) string { return g.UserGroup })
}

// ResolvedStrategyList returns StrategyList after optional ShowAb.
func (g *showUserGroup) ResolvedStrategyList(expIDs []int32) []ShowStrategy {
	if g == nil {
		return nil
	}
	if len(g.ShowAb) == 0 {
		return g.StrategyList
	}
	i := recsyskit.PickABVariantIndex(expIDs, len(g.ShowAb), func(j int) int32 { return g.ShowAb[j].ExpID })
	return g.ShowAb[i].StrategyList
}
