package recsyskit

import (
	"encoding/json"
	"os"
)

// FunnelLibrary mirrors online_map_center Config_Recall.MapRecommend (subset for OSS funnel).
type FunnelLibrary struct {
	MapRecommend []FunnelVariant `json:"MapRecommend"`
}

// FunnelVariant is one AB bucket keyed by exp_id (first match in request ExpIDs wins, else exp_id 0).
type FunnelVariant struct {
	ExpID          int32             `json:"exp_id"`
	UserGroupList []FunnelUserGroup `json:"UserGroupList"`
}

// FunnelUserGroup is one user bucket (def_group, T0_NewUser, ...).
type FunnelUserGroup struct {
	UserGroup           string            `json:"UserGroup"`
	AllMergeNum         int               `json:"AllMergeNum"`
	FilterPolicies      []FilterPolicy    `json:"FilterPolicies,omitempty"`
	ExclusiveRecallList []RecallMergeRule `json:"ExclusiveRecallList,omitempty"`
	RecallAndMergeList  []RecallMergeRule `json:"RecallAndMergeList"`
	// FinalRetCount default when request ret_count is 0.
	FinalRetCount int `json:"FinalRetCount,omitempty"`
	ShowControl   ShowControlCfg `json:"ShowControl,omitempty"`

	// --- Per-layer AB (filter / recall / 展控 only; 粗精排序见 rank RankExpConf.json). ---
	FilterAb []FilterAbVariant `json:"FilterAb,omitempty"`
	RecallAb []RecallAbVariant `json:"RecallAb,omitempty"`
	ShowAb   []ShowAbVariant   `json:"ShowAb,omitempty"`
}

// FilterAbVariant is filter-layer AB (独立实验号，可与召回/排序不同).
type FilterAbVariant struct {
	ExpID          int32          `json:"exp_id"`
	FilterPolicies []FilterPolicy `json:"FilterPolicies"`
}

// RecallAbVariant is recall-layer AB (多路召回配置整包替换).
type RecallAbVariant struct {
	ExpID               int32             `json:"exp_id"`
	ExclusiveRecallList []RecallMergeRule `json:"ExclusiveRecallList,omitempty"`
	RecallAndMergeList  []RecallMergeRule `json:"RecallAndMergeList"`
}

// ShowAbVariant is 展控层 AB.
type ShowAbVariant struct {
	ExpID        int32          `json:"exp_id"`
	ShowControl  ShowControlCfg `json:"ShowControl,omitempty"`
}

// RecallMergeRule mirrors RecallAndMergeList / ExclusiveRecallList rows.
type RecallMergeRule struct {
	RecallType  string `json:"RecallType"`
	RecallNum   int    `json:"RecallNum"`
	MergeMaxNum int    `json:"MergeMaxNum,omitempty"`
	SampleFold  int    `json:"SampleFold,omitempty"`
	// UseTopKIndex: if >0, keep only the first K raw candidates from the lane (before SampleFold / merge cap).
	UseTopKIndex int `json:"UseTopKIndex,omitempty"`
}

// FilterPolicy is a named gate after merge (过滤 / 退避).
type FilterPolicy struct {
	Type         string `json:"Type"` // exposure_backoff
	BackoffAfter int    `json:"BackoffAfter,omitempty"`
}

// ShowControlCfg is a minimal 展控 layer (cap / future diversity).
type ShowControlCfg struct {
	MaxItems int `json:"MaxItems,omitempty"`
}

// LoadFunnelLibrary loads JSON funnel config from path.
func LoadFunnelLibrary(path string) (*FunnelLibrary, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var lib FunnelLibrary
	if err := json.Unmarshal(b, &lib); err != nil {
		return nil, err
	}
	return &lib, nil
}

// ResolveFunnel picks AB variant then user group. userGroup empty matches first list entry.
func (lib *FunnelLibrary) ResolveFunnel(expIDs []int32, userGroup string) *FunnelUserGroup {
	if lib == nil || len(lib.MapRecommend) == 0 {
		return nil
	}
	var v *FunnelVariant
	for _, exp := range expIDs {
		for i := range lib.MapRecommend {
			if lib.MapRecommend[i].ExpID == exp {
				v = &lib.MapRecommend[i]
				break
			}
		}
		if v != nil {
			break
		}
	}
	if v == nil {
		for i := range lib.MapRecommend {
			if lib.MapRecommend[i].ExpID == 0 {
				v = &lib.MapRecommend[i]
				break
			}
		}
	}
	if v == nil {
		v = &lib.MapRecommend[0]
	}
	if len(v.UserGroupList) == 0 {
		return nil
	}
	if userGroup != "" {
		for i := range v.UserGroupList {
			if v.UserGroupList[i].UserGroup == userGroup {
				return &v.UserGroupList[i]
			}
		}
	}
	return &v.UserGroupList[0]
}

// PickABVariantIndex picks the first variant whose exp_id appears in expIDs, else exp_id 0, else index 0.
func PickABVariantIndex(expIDs []int32, n int, expOf func(int) int32) int {
	if n <= 0 {
		return 0
	}
	for _, e := range expIDs {
		for i := 0; i < n; i++ {
			if expOf(i) == e {
				return i
			}
		}
	}
	for i := 0; i < n; i++ {
		if expOf(i) == 0 {
			return i
		}
	}
	return 0
}

// ResolvedFilterPolicies returns FilterAb hit by expIDs, or legacy FilterPolicies.
func (g *FunnelUserGroup) ResolvedFilterPolicies(expIDs []int32) []FilterPolicy {
	if g == nil {
		return nil
	}
	if len(g.FilterAb) == 0 {
		return g.FilterPolicies
	}
	i := PickABVariantIndex(expIDs, len(g.FilterAb), func(j int) int32 { return g.FilterAb[j].ExpID })
	return g.FilterAb[i].FilterPolicies
}

// ResolvedRecallLists returns exclusive + main lists from RecallAb or legacy fields.
func (g *FunnelUserGroup) ResolvedRecallLists(expIDs []int32) (exclusive, main []RecallMergeRule) {
	if g == nil {
		return nil, nil
	}
	if len(g.RecallAb) == 0 {
		return g.ExclusiveRecallList, g.RecallAndMergeList
	}
	i := PickABVariantIndex(expIDs, len(g.RecallAb), func(j int) int32 { return g.RecallAb[j].ExpID })
	return g.RecallAb[i].ExclusiveRecallList, g.RecallAb[i].RecallAndMergeList
}

// ResolvedShowControl returns 展控 from ShowAb or legacy ShowControl.
func (g *FunnelUserGroup) ResolvedShowControl(expIDs []int32) ShowControlCfg {
	if g == nil {
		return ShowControlCfg{}
	}
	if len(g.ShowAb) == 0 {
		return g.ShowControl
	}
	i := PickABVariantIndex(expIDs, len(g.ShowAb), func(j int) int32 { return g.ShowAb[j].ExpID })
	return g.ShowAb[i].ShowControl
}
