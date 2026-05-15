package transporthttp

import (
	"encoding/json"
	"testing"
)

func TestMultiRankResponseJSONUnmarshal(t *testing.T) {
	const sample = `{"uuid":"u","user_id":1,"exp":{"pre_rank_exp_id":0,"rank_exp_id":0,"re_rank_exp_id":0},"ranked_groups":[{"name":"Main","item_scores":[{"item_id":10003,"pre_rank_score":3,"rank_score":3,"re_rank_score":3},{"item_id":10002,"pre_rank_score":2,"rank_score":2,"re_rank_score":2},{"item_id":10001,"pre_rank_score":1,"rank_score":1,"re_rank_score":1}]}]}`
	var wire MultiRankResponseJSON
	if err := json.Unmarshal([]byte(sample), &wire); err != nil {
		t.Fatal(err)
	}
	if len(wire.RankedGroups) != 1 {
		t.Fatalf("ranked_groups: got %+v", wire)
	}
	if len(wire.RankedGroups[0].ItemScores) != 3 {
		t.Fatalf("item_scores: got %+v", wire.RankedGroups[0])
	}
}
