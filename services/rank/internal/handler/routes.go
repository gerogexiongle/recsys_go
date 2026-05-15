package handler

import (
	"encoding/json"
	"net/http"

	"github.com/zeromicro/go-zero/rest"

	"recsys_go/pkg/recsyskit/transporthttp"
	"recsys_go/services/rank/internal/logic"
	"recsys_go/services/rank/internal/svc"
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
	l := logic.NewMultiRank(svcCtx)
	server.AddRoute(rest.Route{
		Method: http.MethodPost,
		Path:   "/v1/rank/multi",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			var req transporthttp.MultiRankRequestJSON
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			resp, err := l.Handle(r.Context(), &req)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
		},
	})
}
