package centerconfig

import (
	"context"
	"strings"

	"recsys_go/pkg/recsyskit"
)

// ApplyRuleFilters runs Config_Filter-style rule strategies in order (OSS subset; unknown types are no-op).
func ApplyRuleFilters(rctx recsyskit.RequestContext, rules []RuleFilterStrategy, items []recsyskit.ItemInfo) []recsyskit.ItemInfo {
	out := items
	for _, rule := range rules {
		switch rule.FilterType {
		case "LiveExposure":
			out = filterLiveExposure(out, rctx, rule.ExposureLimit)
		default:
			// NoRoom, SuppressMap, UserDislikeMap, HighQuality, PlatformOGC, ... — domain-specific, skipped in OSS demo.
		}
	}
	return out
}

func filterLiveExposure(items []recsyskit.ItemInfo, rctx recsyskit.RequestContext, limit int) []recsyskit.ItemInfo {
	if limit <= 0 || rctx.Exposure == nil || len(items) == 0 {
		return items
	}
	var out []recsyskit.ItemInfo
	for _, it := range items {
		if rctx.Exposure[it.ID] > limit {
			continue
		}
		out = append(out, it)
	}
	return out
}

// ApplyFeatureFilters runs feature-side filters (OSS subset).
func ApplyFeatureFilters(_ recsyskit.RequestContext, rules []FeatureFilterStrategy, items []recsyskit.ItemInfo) []recsyskit.ItemInfo {
	out := items
	for _, rule := range rules {
		switch rule.FilterType {
		case "FeatureLess":
			out = filterFeatureLess(out)
		case "LabelTypeWhiteList":
			out = filterLabelWhiteList(out, rule.WhiteListLabel)
		default:
		}
	}
	return out
}

func filterFeatureLess(items []recsyskit.ItemInfo) []recsyskit.ItemInfo {
	if len(items) == 0 {
		return items
	}
	var out []recsyskit.ItemInfo
	for _, it := range items {
		if !it.HasPortrait {
			continue
		}
		out = append(out, it)
	}
	return out
}

func filterLabelWhiteList(items []recsyskit.ItemInfo, label string) []recsyskit.ItemInfo {
	if label == "" || len(items) == 0 {
		return items
	}
	var out []recsyskit.ItemInfo
	for _, it := range items {
		if it.Extra == nil {
			continue
		}
		if it.Extra["label"] == label {
			out = append(out, it)
		}
	}
	return out
}

// CapKeepItemNum truncates to pre-rank pool size (Config_Filter.KeepItemNum).
func CapKeepItemNum(keep int, items []recsyskit.ItemInfo) []recsyskit.ItemInfo {
	if keep <= 0 || len(items) <= keep {
		return items
	}
	return items[:keep]
}

// ApplyShowStrategies runs Config_ShowControl without exclusive pool (unit tests).
func ApplyShowStrategies(items []recsyskit.ItemInfo, strategies []ShowStrategy) []recsyskit.ItemInfo {
	return ApplyShowStrategiesWithExclusive(context.Background(), nil, items, nil, strategies)
}

func applyScoreControl(items []recsyskit.ItemInfo, st ShowStrategy) []recsyskit.ItemInfo {
	if len(items) == 0 || st.Method != "RecallType" || st.ScoreControlFactor <= 0 {
		return items
	}
	set := parseRecallTypeSet(st.RecallTypeList)
	if len(set) == 0 {
		return items
	}
	out := append([]recsyskit.ItemInfo(nil), items...)
	for i := range out {
		if set[out[i].RecallType] {
			out[i].Score *= st.ScoreControlFactor
		}
	}
	return out
}

func parseRecallTypeSet(s string) map[string]bool {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	m := make(map[string]bool, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			m[p] = true
		}
	}
	return m
}

func applyHomogenCap(items []recsyskit.ItemInfo, topN int) []recsyskit.ItemInfo {
	if topN <= 0 || len(items) <= topN {
		return items
	}
	return items[:topN]
}

func applyMMRRearrange(items []recsyskit.ItemInfo, st ShowStrategy) []recsyskit.ItemInfo {
	if len(items) < 2 {
		return items
	}
	lambda := st.MMRConstant
	if lambda <= 0 || lambda > 1 {
		lambda = 0.5
	}
	maxOut := st.MMRDimension
	if maxOut <= 0 {
		maxOut = len(items)
	}
	rel := make([]float64, len(items))
	maxS := 0.0
	for i := range items {
		if items[i].Score > maxS {
			maxS = items[i].Score
		}
	}
	if maxS <= 0 {
		maxS = 1
	}
	for i := range items {
		rel[i] = items[i].Score / maxS
	}
	used := make([]bool, len(items))
	var picked []int
	for len(picked) < maxOut && len(picked) < len(items) {
		bestI := -1
		bestScore := -1.0
		for i := range items {
			if used[i] {
				continue
			}
			div := 0.0
			for _, j := range picked {
				if items[i].RecallType == items[j].RecallType {
					div = 1
					break
				}
			}
			mm := lambda*rel[i] - (1-lambda)*div
			if mm > bestScore {
				bestScore = mm
				bestI = i
			}
		}
		if bestI < 0 {
			break
		}
		used[bestI] = true
		picked = append(picked, bestI)
	}
	out := make([]recsyskit.ItemInfo, 0, len(picked))
	for _, i := range picked {
		out = append(out, items[i])
	}
	return out
}

