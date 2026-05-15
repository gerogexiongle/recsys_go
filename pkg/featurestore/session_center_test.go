package featurestore

import (
	"context"
	"testing"

	"recsys_go/pkg/recsyskit"
)

type stubStrategyFetcher struct {
	NoOpFetcher
	expJSON []byte
	flJSON  []byte
	lbJSON  []byte
}

func (s stubStrategyFetcher) FilterExposureJSON(context.Context) ([]byte, bool, error) {
	return s.expJSON, len(s.expJSON) == 0, nil
}

func (s stubStrategyFetcher) FilterLabelJSON(context.Context) ([]byte, bool, error) {
	return s.lbJSON, len(s.lbJSON) == 0, nil
}

func TestLoadCenterSession_mergedFilterKeys(t *testing.T) {
	fetch := stubStrategyFetcher{
		expJSON: []byte(`{"910005":15}`),
		lbJSON:  []byte(`{"910001":"Adventure"}`),
	}
	cs, err := LoadCenterSession(context.Background(), fetch, 900001, []int64{910001})
	if err != nil {
		t.Fatal(err)
	}
	if cs.Exposure[910005] != 15 {
		t.Fatalf("exposure %+v", cs.Exposure)
	}
	items := []recsyskit.ItemInfo{{ID: 910001}}
	enriched := cs.EnrichItems(items)
	if enriched[0].Extra["label"] != "Adventure" {
		t.Fatalf("label %+v", enriched[0].Extra)
	}
}
