package transporthttp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"recsys_go/pkg/recsyskit"
)

func TestRankHTTPClient_AllowsEmptyResponseWhenNoItemGroups(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"uuid":"","user_id":0,"exp":{"pre_rank_exp_id":0,"rank_exp_id":0,"re_rank_exp_id":0}}`))
	}))
	defer ts.Close()

	c, err := NewRankHTTPClientSingle(ts.URL, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.MultiRank(context.Background(), &recsyskit.MultiRankRequest{
		Ctx:    recsyskit.RequestContext{},
		Groups: nil,
	})
	if err != nil {
		t.Fatal(err)
	}
}
