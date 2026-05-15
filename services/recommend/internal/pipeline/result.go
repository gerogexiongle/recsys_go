package pipeline

import (
	"recsys_go/pkg/recsyskit"
	"recsys_go/pkg/recsyskit/transporthttp"
)

// Result is the center pipeline output plus final items for algorithm Kafka log.
type Result struct {
	Resp  *transporthttp.RecommendResponseJSON
	Items []recsyskit.ItemInfo
}
