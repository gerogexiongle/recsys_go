package featurestore

import "context"

// Fetcher loads profile JSON for FM / rank (recsysgo:feat:user|item).
type Fetcher interface {
	UserJSON(ctx context.Context, uin int64) ([]byte, error)
	ItemJSON(ctx context.Context, itemID int64) ([]byte, error)
}

type BatchFetcher interface {
	Fetcher
	ItemsJSON(ctx context.Context, itemIDs []int64) (map[int64][]byte, error)
}

// StrategyFetcher loads merged filter blobs (one GET per strategy namespace).
type StrategyFetcher interface {
	FilterExposureJSON(ctx context.Context) ([]byte, bool, error)
	FilterFeatureLessJSON(ctx context.Context) ([]byte, bool, error)
	FilterLabelJSON(ctx context.Context) ([]byte, bool, error)
}

// RecallFetcher loads recall lists: lane = global; CF / tag-interest = per user; tag invert per tag id.
type RecallFetcher interface {
	RecallLaneJSON(ctx context.Context, lane string) ([]byte, bool, error)
	RecallCFUserJSON(ctx context.Context, uin int64) ([]byte, bool, error)
	UserTagInterestJSON(ctx context.Context, window string, uin int64) ([]byte, bool, error)
	TagInvertJSON(ctx context.Context, tagID int) ([]byte, bool, error)
}

type NoOpFetcher struct{}

func (NoOpFetcher) UserJSON(context.Context, int64) ([]byte, error) { return nil, nil }
func (NoOpFetcher) ItemJSON(context.Context, int64) ([]byte, error) { return nil, nil }
func (NoOpFetcher) ItemsJSON(context.Context, []int64) (map[int64][]byte, error) {
	return map[int64][]byte{}, nil
}
func (NoOpFetcher) FilterExposureJSON(context.Context) ([]byte, bool, error) {
	return nil, true, nil
}
func (NoOpFetcher) FilterFeatureLessJSON(context.Context) ([]byte, bool, error) {
	return nil, true, nil
}
func (NoOpFetcher) FilterLabelJSON(context.Context) ([]byte, bool, error) {
	return nil, true, nil
}
func (NoOpFetcher) RecallLaneJSON(context.Context, string) ([]byte, bool, error) {
	return nil, true, nil
}
func (NoOpFetcher) RecallCFUserJSON(context.Context, int64) ([]byte, bool, error) {
	return nil, true, nil
}
func (NoOpFetcher) UserTagInterestJSON(context.Context, string, int64) ([]byte, bool, error) {
	return nil, true, nil
}
func (NoOpFetcher) TagInvertJSON(context.Context, int) ([]byte, bool, error) {
	return nil, true, nil
}

var NoOp Fetcher = NoOpFetcher{}
