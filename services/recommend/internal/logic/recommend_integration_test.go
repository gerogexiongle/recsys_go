package logic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"recsys_go/pkg/featurestore"
	"recsys_go/pkg/recsyskit/transporthttp"
)

func TestRecommendUsesRankOrder(t *testing.T) {
	rankBody := `{"uuid":"x","user_id":1,"exp":{"pre_rank_exp_id":0,"rank_exp_id":0,"re_rank_exp_id":0},"ranked_groups":[{"name":"Main","item_scores":[{"item_id":10003,"pre_rank_score":3,"rank_score":3,"re_rank_score":3},{"item_id":10002,"pre_rank_score":2,"rank_score":2,"re_rank_score":2},{"item_id":10001,"pre_rank_score":1,"rank_score":1,"re_rank_score":1}]}]}`
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/rank/multi" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(rankBody))
	}))
	defer ts.Close()

	client := transporthttp.NewRankHTTPClient(ts.URL, time.Second)
	rec := NewRecommend(client, featurestore.NoOp)

	resp, err := rec.Handle(context.Background(), &transporthttp.RecommendRequestJSON{
		UUID:     "x",
		UserID:   1,
		RetCount: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.ItemIDs) != 2 || resp.ItemIDs[0] != 10003 || resp.ItemIDs[1] != 10002 {
		t.Fatalf("got %+v", resp.ItemIDs)
	}
}
