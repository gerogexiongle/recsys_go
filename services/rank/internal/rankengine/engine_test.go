package rankengine

import (
	"context"
	"testing"

	"recsys_go/services/rank/internal/config"
	"recsys_go/pkg/featurestore"
)

func TestEnginePipelineToyFM(t *testing.T) {
	cfg := config.RankEngineConfig{
		Mode: "pipeline",
		FM: config.FMConfig{
			Factor:    2,
			ModelPath: "testdata/fm_toy.txt",
			TransPath: "",
		},
	}
	eng, err := NewEngine(cfg, featurestore.NoOp)
	if err != nil {
		t.Fatal(err)
	}
	out := eng.RankGroup(context.Background(), 1, nil, []int64{10001, 10002, 10003}, 3)
	if len(out) != 3 {
		t.Fatalf("len %d", len(out))
	}
	for _, it := range out {
		if it.PreRank <= 0 || it.Rank <= 0 || it.ReRank <= 0 {
			t.Fatalf("non-positive scores %+v", it)
		}
	}
}
