package logic

import (
	"recsys_go/pkg/algolog"
	"recsys_go/pkg/recsyskit"
	"recsys_go/pkg/recsyskit/transporthttp"
)

func (l *Recommend) pushAlgoLog(req *transporthttp.RecommendRequestJSON, rctx recsyskit.RequestContext, items []recsyskit.ItemInfo) {
	if l.AlgoKafka == nil || !l.AlgoKafka.Enabled() {
		return
	}
	in := algolog.Input{
		UUID:            req.UUID,
		UserID:          req.UserID,
		Section:         req.Section,
		ExpIDs:          rctx.ExpIDs,
		DisablePersonal: req.DisablePersonal,
		DeviceID:        req.DeviceID,
		TerminalModel:   req.TerminalModel,
		OSType:          req.OS,
		Items:           items,
		APIType:         l.AlgoKafka.APIType(),
		DataType:        l.AlgoKafka.DataType(),
	}
	rec := algolog.BuildRecord(in)
	l.AlgoKafka.Push(rec.Serialize())
}
