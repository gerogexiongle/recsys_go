package centerconfig

import (
	"testing"

	"recsys_go/pkg/recsyskit"
)

func TestLiveExposureFilter(t *testing.T) {
	items := []recsyskit.ItemInfo{
		{ID: 910001},
		{ID: 910005},
		{ID: 910004},
	}
	rctx := recsyskit.RequestContext{
		Exposure: map[recsyskit.ItemID]int{910005: 15},
	}
	out := ApplyRuleFilters(rctx, []RuleFilterStrategy{{FilterType: "LiveExposure", ExposureLimit: 3}}, items)
	for _, it := range out {
		if it.ID == 910005 {
			t.Fatalf("910005 should be filtered")
		}
	}
}

func TestScoreControlBoost(t *testing.T) {
	items := []recsyskit.ItemInfo{
		{ID: 1, RecallType: "LiveRedirect", Score: 1.0},
		{ID: 2, RecallType: "HotMap", Score: 1.0},
	}
	out := ApplyShowStrategies(items, []ShowStrategy{{
		ShowControlType:    "ScoreControl",
		Method:             "RecallType",
		RecallTypeList:     "LiveRedirect",
		ScoreControlFactor: 2.0,
	}})
	if out[0].Score != 2.0 {
		t.Fatalf("expected boost on LiveRedirect, got %v", out[0].Score)
	}
	if out[1].Score != 1.0 {
		t.Fatalf("expected unchanged HotMap score, got %v", out[1].Score)
	}
}
