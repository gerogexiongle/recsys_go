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

	"recsys_go/pkg/featurestore"
	"recsys_go/pkg/recsyskit"
	"recsys_go/pkg/recsyskit/transporthttp"
	"recsys_go/services/recommend/internal/recall"
)

func TestFunnelFiltersHighExposureItem(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller")
	}
	fpath := filepath.Join(filepath.Dir(file), "..", "..", "etc", "recommend-funnel.json")
	lib, err := recsyskit.LoadFunnelLibrary(fpath)
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			ItemGroups []struct {
				ItemIDs []int64 `json:"item_ids"`
			} `json:"item_groups"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if len(body.ItemGroups) == 0 {
			http.Error(w, "bad", 400)
			return
		}
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
			"exp":    map[string]any{"pre_rank_exp_id": 0, "rank_exp_id": 0, "re_rank_exp_id": 0},
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
	rec := NewRecommendFunnel(client, featurestore.NoOp, lib, recall.NewRegistry(nil))
	resp, err := rec.Handle(context.Background(), &transporthttp.RecommendRequestJSON{
		UUID:     "t",
		UserID:   1,
		ExpIDs:   []int32{0},
		RetCount: 200,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range resp.ItemIDs {
		if id == 910005 {
			t.Fatalf("910005 should be filtered by exposure_backoff")
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
		t.Fatalf("expected LiveRedirect head 910001 in ids %+v", resp.ItemIDs)
	}
}
