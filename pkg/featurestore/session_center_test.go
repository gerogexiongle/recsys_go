package featurestore

import (
	"context"
	"testing"

	"recsys_go/pkg/recsyskit"
)

type stubStrategyFetcher struct {
	NoOpFetcher
	expJSON     []byte
	expMissing  bool
	featureLess map[int64][]byte
	labels      map[int64][]byte
}

func (s stubStrategyFetcher) UserExposureJSON(context.Context, int64) ([]byte, bool, error) {
	return s.expJSON, s.expMissing, nil
}

func (s stubStrategyFetcher) ItemsFeatureLessJSON(_ context.Context, ids []int64) (map[int64][]byte, error) {
	out := make(map[int64][]byte)
	for _, id := range ids {
		if b, ok := s.featureLess[id]; ok {
			out[id] = b
		}
	}
	return out, nil
}

func (s stubStrategyFetcher) ItemsLabelJSON(_ context.Context, ids []int64) (map[int64][]byte, error) {
	out := make(map[int64][]byte)
	for _, id := range ids {
		if b, ok := s.labels[id]; ok {
			out[id] = b
		}
	}
	return out, nil
}

func TestLoadCenterSession_strategyKeys(t *testing.T) {
	fetch := stubStrategyFetcher{
		expJSON:    []byte(`{"910005":15}`),
		expMissing: false,
		featureLess: map[int64][]byte{
			910009: []byte("1"),
		},
	}
	cs, err := LoadCenterSession(context.Background(), fetch, 900001, []int64{910005, 910009, 910010})
	if err != nil {
		t.Fatal(err)
	}
	if cs.Exposure[910005] != 15 {
		t.Fatalf("exposure %+v", cs.Exposure)
	}
	items := []recsyskit.ItemInfo{
		{ID: 910009}, {ID: 910010},
	}
	enriched := cs.EnrichItems(items)
	if enriched[0].Extra["feature_less"] != "1" {
		t.Fatalf("910009 feature_less %+v", enriched[0].Extra)
	}
	if enriched[1].Extra != nil && enriched[1].Extra["feature_less"] == "1" {
		t.Fatal("910010 must not be feature-less when key missing")
	}
}
