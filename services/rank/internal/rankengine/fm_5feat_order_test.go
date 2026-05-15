package rankengine

import (
	"context"
	"testing"

	"recsys_go/services/rank/internal/config"
	"recsys_go/pkg/featurestore"
)

func TestPipelineFM5FeatCTROrdering(t *testing.T) {
	cfg := config.RankEngineConfig{
		Mode: "pipeline",
		FM: config.FMConfig{
			Factor:    4,
			ModelPath: "testdata/fm_5feat.txt",
			TransPath: "",
		},
	}
	eng, err := NewEngine(cfg, featurestore.NoOp)
	if err != nil {
		t.Fatal(err)
	}
	u := int64(900001)
	ids := []int64{910001, 910002, 910003}
	out := eng.RankGroup(context.Background(), u, nil, ids, 2)
	if len(out) != 2 {
		t.Fatalf("len %d", len(out))
	}
	if out[0].ItemID != 910003 || out[1].ItemID != 910002 {
		t.Fatalf("want [910003,910002] got %+v", out)
	}
}

// stubSemanticFetcher returns fixed Redis-shaped JSON (age/gender/income + ctr/revenue).
type stubSemanticFetcher struct {
	user []byte
	item map[int64][]byte
}

func (s stubSemanticFetcher) UserJSON(_ context.Context, _ int64) ([]byte, error) {
	return s.user, nil
}

func (s stubSemanticFetcher) ItemJSON(_ context.Context, itemID int64) ([]byte, error) {
	if s.item == nil {
		return nil, nil
	}
	return s.item[itemID], nil
}

func TestPipelineFM5FeatSemanticJSONOrder(t *testing.T) {
	user := []byte(`{"age":40,"gender":1,"income_wan":6}`)
	cfg := config.RankEngineConfig{
		Mode: "pipeline",
		FM: config.FMConfig{
			Factor:    4,
			ModelPath: "testdata/fm_5feat.txt",
			TransPath: "",
		},
	}
	fetch := stubSemanticFetcher{
		user: user,
		item: map[int64][]byte{
			910001: []byte(`{"ctr_7d":0.02,"revenue_7d":20000}`),
			910002: []byte(`{"ctr_7d":0.06,"revenue_7d":21000}`),
			910003: []byte(`{"ctr_7d":0.14,"revenue_7d":20000}`),
		},
	}
	eng, err := NewEngine(cfg, fetch)
	if err != nil {
		t.Fatal(err)
	}
	out := eng.RankGroup(context.Background(), 900001, nil, []int64{910001, 910002, 910003}, 2)
	if len(out) != 2 {
		t.Fatalf("len %d", len(out))
	}
	if out[0].ItemID != 910003 || out[1].ItemID != 910002 {
		t.Fatalf("want [910003,910002] got %+v", out)
	}
}
