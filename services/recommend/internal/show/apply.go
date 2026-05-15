package show

import (
	"context"

	"recsys_go/pkg/featurestore"
	"recsys_go/pkg/recsyskit"
	"recsys_go/services/recommend/internal/centerconfig"
)

// ApplyStrategies runs Config_ShowControl (exclusive pool used by ForcedInsert, Homogen on ranked main).
func ApplyStrategies(ctx context.Context, feat featurestore.Fetcher, items []recsyskit.ItemInfo, exclusive recsyskit.ExclusivePool, strategies []centerconfig.ShowStrategy) []recsyskit.ItemInfo {
	return centerconfig.ApplyShowStrategiesWithExclusive(ctx, feat, items, exclusive, strategies)
}
