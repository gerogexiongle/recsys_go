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

func (s stubStrategyFetcher) FilterFeatureLessJSON(context.Context) ([]byte, bool, error) {
	return s.flJSON, len(s.flJSON) == 0, nil
}

func (s stubStrategyFetcher) FilterLabelJSON(context.Context) ([]byte, bool, error) {
	return s.lbJSON, len(s.lbJSON) == 0, nil
}

func TestLoadCenterSession_mergedFilterKeys(t *testing.T) {
	fetch := stubStrategyFetcher{
		expJSON: []byte(`{"910005":15}`),
		flJSON:  []byte(`[910009]`),
	}
	cs, err := LoadCenterSession(context.Background(), fetch, 900001, []int64{910005, 910009})
	if err != nil {
		t.Fatal(err)
	}
	if cs.Exposure[910005] != 15 {
		t.Fatalf("exposure %+v", cs.Exposure)
	}
	items := []recsyskit.ItemInfo{{ID: 910009}, {ID: 910010}}
	enriched := cs.EnrichItems(items)
	if enriched[0].Extra["feature_less"] != "1" {
		t.Fatalf("910009 %+v", enriched[0].Extra)
	}
	if enriched[1].Extra != nil && enriched[1].Extra["feature_less"] == "1" {
		t.Fatal("910010 should not be feature-less")
	}
}
