package recall

import (
	"context"

	"recsys_go/pkg/featurestore"
	"recsys_go/pkg/recsyskit"
)

// laneFromRedisList loads recsysgo:recall:lane:{RecallType} or cf:user for CF.
func laneFromRedisList(ctx context.Context, fetch featurestore.RecallFetcher, userID int64, recallType string, n int) ([]recsyskit.ItemInfo, bool) {
	if fetch == nil {
		return nil, false
	}
	var raw []byte
	var missing bool
	var err error
	if recallType == "CollaborativeFiltering" {
		raw, missing, err = fetch.RecallCFUserJSON(ctx, userID)
	} else {
		raw, missing, err = fetch.RecallLaneJSON(ctx, recallType)
	}
	if err != nil || missing || len(raw) == 0 {
		return nil, false
	}
	ids := featurestore.ParseRecallList(raw)
	if len(ids) == 0 {
		return nil, false
	}
	if n > 0 && len(ids) > n {
		ids = ids[:n]
	}
	out := make([]recsyskit.ItemInfo, len(ids))
	for i, id := range ids {
		out[i] = recsyskit.ItemInfo{ID: recsyskit.ItemID(id), RecallType: recallType}
	}
	return out, true
}
