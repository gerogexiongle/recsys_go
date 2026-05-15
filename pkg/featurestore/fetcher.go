package featurestore

import "context"

// Fetcher loads raw JSON feature blobs (user / item dimensions).
// Aligns with C++ UserFeature_0 HGET at request start + batch map feature before filter/rank.
type Fetcher interface {
	UserJSON(ctx context.Context, uin int64) ([]byte, error)
	ItemJSON(ctx context.Context, itemID int64) ([]byte, error)
}

// BatchFetcher optional extension for center merge→filter (one MGET per request).
type BatchFetcher interface {
	Fetcher
	ItemsJSON(ctx context.Context, itemIDs []int64) (map[int64][]byte, error)
}

// NoOpFetcher always returns empty payloads (offline / CI).
type NoOpFetcher struct{}

func (NoOpFetcher) UserJSON(context.Context, int64) ([]byte, error) { return nil, nil }
func (NoOpFetcher) ItemJSON(context.Context, int64) ([]byte, error) { return nil, nil }
func (NoOpFetcher) ItemsJSON(context.Context, []int64) (map[int64][]byte, error) {
	return map[int64][]byte{}, nil
}

// NoOp is a Fetcher that never returns data (rank falls back to placeholders).
var NoOp Fetcher = NoOpFetcher{}
