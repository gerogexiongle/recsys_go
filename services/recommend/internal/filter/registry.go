package filter

import (
	"recsys_go/pkg/recsyskit"
	"recsys_go/services/recommend/internal/centerconfig"
)

// ApplyRuleFilters delegates to centerconfig rule strategies (LiveExposure, ...).
func ApplyRuleFilters(rctx recsyskit.RequestContext, rules []centerconfig.RuleFilterStrategy, items []recsyskit.ItemInfo) []recsyskit.ItemInfo {
	return centerconfig.ApplyRuleFilters(rctx, rules, items)
}

// ApplyFeatureFilters runs Config_Filter feature strategies (FeatureLess = no portrait).
func ApplyFeatureFilters(_ recsyskit.RequestContext, rules []centerconfig.FeatureFilterStrategy, items []recsyskit.ItemInfo) []recsyskit.ItemInfo {
	out := items
	for _, rule := range rules {
		switch rule.FilterType {
		case "FeatureLess":
			out = ApplyFeatureLess(out)
		case "LabelTypeWhiteList":
			out = centerconfig.ApplyFeatureFilters(recsyskit.RequestContext{}, []centerconfig.FeatureFilterStrategy{rule}, out)
		default:
		}
	}
	return out
}
