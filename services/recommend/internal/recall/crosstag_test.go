package recall

import (
	"context"
	"testing"

	"recsys_go/pkg/featurestore"
	"recsys_go/pkg/recsyskit"
)

type stubTagRecall struct {
	featurestore.NoOpFetcher
	interest map[int64][]byte
	invert   map[int][]byte
}

func (s stubTagRecall) UserTagInterestJSON(_ context.Context, window string, uin int64) ([]byte, bool, error) {
	if window != "7d" {
		return nil, true, nil
	}
	b, ok := s.interest[uin]
	return b, !ok, nil
}

func (s stubTagRecall) TagInvertJSON(_ context.Context, tagID int) ([]byte, bool, error) {
	b, ok := s.invert[tagID]
	return b, !ok, nil
}

func TestCrossTagLane_fromInvert(t *testing.T) {
	fetch := stubTagRecall{
		interest: map[int64][]byte{
			900001: []byte(`[{"tag":3,"weight":0.7},{"tag":4,"weight":0.3}]`),
		},
		invert: map[int][]byte{
			3: []byte(`[910005,910006]`),
			4: []byte(`[910007,910008]`),
		},
	}
	reg := NewRegistry(fetch)
	items, err := reg.Lane(context.Background(), recsyskit.RequestContext{UserID: 900001}, recsyskit.RecallMergeRule{
		RecallType:   "CrossTag7d",
		RecallNum:    4,
		UseTopKIndex: 2,
		SampleFold:   1,
	})
	if err != nil {
		t.Fatal(err)
	}
	seen := map[int64]bool{}
	for _, it := range items {
		if it.RecallType != "CrossTag7d" {
			t.Fatalf("type %s", it.RecallType)
		}
		seen[int64(it.ID)] = true
	}
	if !seen[910005] || !seen[910007] {
		t.Fatalf("expected tag invert items, got %+v", items)
	}
}
