package centerconfig

import (
	"context"
	"testing"

	"recsys_go/pkg/recsyskit"
)

func TestForcedInsertFromExclusivePool(t *testing.T) {
	exclusive := recsyskit.ExclusivePool{
		"LiveRedirect": {{ID: 910001, RecallType: "LiveRedirect"}},
	}
	main := []recsyskit.ItemInfo{
		{ID: 910010, RecallType: "CollaborativeFiltering", Score: 2},
		{ID: 910008, RecallType: "CrossTag7d", Score: 1},
	}
	out := ApplyShowStrategiesWithExclusive(context.Background(), nil, main, exclusive, []ShowStrategy{{
		ShowControlType: "ForcedInsert",
		ForcedInsert: []ForcedInsertRule{{
			RecallType: "LiveRedirect", ForcedInsertCount: 1, ExtractMethod: "TopNOrder",
		}},
	}})
	if len(out) == 0 || out[0].ID != 910001 {
		t.Fatalf("expected 910001 first from exclusive, got %+v", out)
	}
}
