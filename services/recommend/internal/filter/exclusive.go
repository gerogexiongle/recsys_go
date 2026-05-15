package filter

import (
	"recsys_go/pkg/recsyskit"
	"recsys_go/services/recommend/internal/centerconfig"
)

// ApplyToExclusivePool runs rule/feature filters on each exclusive lane (C++ Filter on exclusive_item_info).
func ApplyToExclusivePool(rctx recsyskit.RequestContext, rules []centerconfig.RuleFilterStrategy, feats []centerconfig.FeatureFilterStrategy, pool recsyskit.ExclusivePool) recsyskit.ExclusivePool {
	if len(pool) == 0 {
		return pool
	}
	out := make(recsyskit.ExclusivePool, len(pool))
	for recallType, items := range pool {
		items = ApplyRuleFilters(rctx, rules, items)
		items = ApplyFeatureFilters(rctx, feats, items)
		if len(items) > 0 {
			out[recallType] = items
		}
	}
	return out
}
