package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/zeromicro/go-zero/rest"

	"recsys_go/pkg/recsyskit/transporthttp"
	"recsys_go/services/recommend/internal/logic"
	"recsys_go/services/recommend/internal/svc"
)

// RegisterHandlers wires HTTP routes on the go-zero rest server.
func RegisterHandlers(server *rest.Server, svcCtx *svc.ServiceContext) {
	server.AddRoute(rest.Route{
		Method: http.MethodGet,
		Path:   "/health",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		},
	})
	// /v1/ready is for smoke tests and ops; do not expose on public ingress without auth.
	server.AddRoute(rest.Route{
		Method: http.MethodGet,
		Path:   "/v1/ready",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintf(w, `{"rank_client":%t,"rank_base_url":%q,"funnel":%t,"center":%t}`, svcCtx.Rank != nil, svcCtx.Config.RankService.BaseURL, svcCtx.Funnel != nil, svcCtx.Center != nil)
		},
	})
	server.AddRoute(rest.Route{
		Method: http.MethodPost,
		Path:   "/v1/recommend",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			var rec *logic.Recommend
			if svcCtx.Center != nil && svcCtx.Recall != nil {
				rec = logic.NewRecommendCenter(svcCtx.Rank, svcCtx.Features, svcCtx.Center, svcCtx.Recall)
			} else if svcCtx.Funnel != nil && svcCtx.Recall != nil {
				rec = logic.NewRecommendFunnel(svcCtx.Rank, svcCtx.Features, svcCtx.Funnel, svcCtx.Recall)
			} else {
				rec = logic.NewRecommend(svcCtx.Rank, svcCtx.Features)
			}
			rec.AlgoKafka = svcCtx.AlgoKafka
			var req transporthttp.RecommendRequestJSON
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			resp, err := rec.Handle(r.Context(), &req)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
		},
	})
}
