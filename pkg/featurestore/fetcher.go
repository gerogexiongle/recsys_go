package featurestore

import "context"

// Fetcher loads profile JSON for FM / rank (recsysgo:feat:user|item).
// Missing key => nil bytes => no portrait features for that entity.
type Fetcher interface {
	UserJSON(ctx context.Context, uin int64) ([]byte, error)
	ItemJSON(ctx context.Context, itemID int64) ([]byte, error)
}

// BatchFetcher optional extension for center merge→filter (one MGET per request).
type BatchFetcher interface {
	Fetcher
	ItemsJSON(ctx context.Context, itemIDs []int64) (map[int64][]byte, error)
}

// StrategyFetcher loads filter-side Redis keys (separate from profile feat keys).
type StrategyFetcher interface {
	UserExposureJSON(ctx context.Context, uin int64) ([]byte, bool, error) // bytes, keyMissing, err
	ItemsFeatureLessJSON(ctx context.Context, itemIDs []int64) (map[int64][]byte, error)
	ItemsLabelJSON(ctx context.Context, itemIDs []int64) (map[int64][]byte, error)
}

// NoOpFetcher always returns empty payloads (offline / CI).
type NoOpFetcher struct{}

func (NoOpFetcher) UserJSON(context.Context, int64) ([]byte, error) { return nil, nil }
func (NoOpFetcher) ItemJSON(context.Context, int64) ([]byte, error) { return nil, nil }
func (NoOpFetcher) ItemsJSON(context.Context, []int64) (map[int64][]byte, error) {
	return map[int64][]byte{}, nil
}
func (NoOpFetcher) UserExposureJSON(context.Context, int64) ([]byte, bool, error) {
	return nil, true, nil
}
func (NoOpFetcher) ItemsFeatureLessJSON(context.Context, []int64) (map[int64][]byte, error) {
	return map[int64][]byte{}, nil
}
func (NoOpFetcher) ItemsLabelJSON(context.Context, []int64) (map[int64][]byte, error) {
	return map[int64][]byte{}, nil
}

// NoOp is a Fetcher that never returns data (rank falls back to placeholders).
var NoOp Fetcher = NoOpFetcher{}
