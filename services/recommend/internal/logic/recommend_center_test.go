package logic

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"recsys_go/pkg/recsyskit/transporthttp"
	"recsys_go/services/recommend/internal/centerconfig"
	"recsys_go/services/recommend/internal/recall"
)

func TestCenterLiveExposureFilters910005(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller")
	}
	etc := filepath.Join(filepath.Dir(file), "..", "..", "etc")
	recallLib, err := centerconfig.LoadRecallLibrary(filepath.Join(etc, "recommend-recall.json"))
	if err != nil {
		t.Fatal(err)
	}
	filterLib, err := centerconfig.LoadFilterLibrary(filepath.Join(etc, "recommend-filter.json"))
	if err != nil {
		t.Fatal(err)
	}
	showLib, err := centerconfig.LoadShowLibrary(filepath.Join(etc, "recommend-showcontrol.json"))
	if err != nil {
		t.Fatal(err)
	}
	center := &centerconfig.CenterBundle{Recall: recallLib, Filter: filterLib, Show: showLib}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			ItemGroups []struct {
				ItemIDs []int64 `json:"item_ids"`
			} `json:"item_groups"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		ids := body.ItemGroups[0].ItemIDs
		var scores []map[string]any
		for i := len(ids) - 1; i >= 0; i-- {
			s := float32(len(ids) - i)
			scores = append(scores, map[string]any{
				"item_id": ids[i], "pre_rank_score": s, "rank_score": s, "re_rank_score": s,
			})
		}
		out := map[string]any{
			"uuid": "t", "user_id": 1,
			"exp":           map[string]any{"pre_rank_exp_id": 0, "rank_exp_id": 0, "re_rank_exp_id": 0},
			"ranked_groups": []any{map[string]any{"name": "Main", "item_scores": scores}},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
	}))
	defer ts.Close()

	client, err := transporthttp.NewRankHTTPClientSingle(ts.URL, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	rec := NewRecommendCenter(client, demoTestFetcher{}, center, recall.NewRegistry(nil))
	resp, err := rec.Handle(context.Background(), &transporthttp.RecommendRequestJSON{
		UUID:     "t",
		UserID:   1,
		ExpIDs:   []int32{0},
		RetCount: 50,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range resp.ItemIDs {
		if id == 910005 {
			t.Fatalf("910005 should be filtered by LiveExposure")
		}
	}
	found := false
	for _, id := range resp.ItemIDs {
		if id == 910001 {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected LiveRedirect 910001 in %+v", resp.ItemIDs)
	}
}
