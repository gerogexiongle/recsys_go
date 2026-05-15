package recall

import (
	"context"
	"fmt"

	"recsys_go/pkg/featurestore"
	"recsys_go/pkg/recsyskit"
)

// Registry executes RecallMergeRule lanes (Redis lists when RecallFetcher set, else stub).
type Registry struct {
	recall featurestore.RecallFetcher
}

func NewRegistry(rec featurestore.RecallFetcher) *Registry {
	return &Registry{recall: rec}
}

func (reg *Registry) Lane(ctx context.Context, rctx recsyskit.RequestContext, rule recsyskit.RecallMergeRule) ([]recsyskit.ItemInfo, error) {
	n := rule.RecallNum
	if n <= 0 {
		return nil, nil
	}
	if reg.recall != nil {
		if items, ok := reg.laneFromRedis(ctx, rctx.UserID, rule.RecallType, n); ok {
			return items, nil
		}
	}
	return reg.laneStub(rule.RecallType, n, rctx.UserID)
}

func (reg *Registry) laneFromRedis(ctx context.Context, userID int64, recallType string, n int) ([]recsyskit.ItemInfo, bool) {
	var raw []byte
	var missing bool
	var err error
	if recallType == "CollaborativeFiltering" {
		raw, missing, err = reg.recall.RecallCFUserJSON(ctx, userID)
	} else {
		raw, missing, err = reg.recall.RecallLaneJSON(ctx, recallType)
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

func (reg *Registry) laneStub(recallType string, n int, userID int64) ([]recsyskit.ItemInfo, error) {
	switch recallType {
	case "LiveRedirect":
		return liveRedirectLane(n, userID), nil
	case "CollaborativeFiltering":
		return collaborativeLane(n, userID), nil
	case "LiveTag":
		return fmDemoItemLane(recallType, n, 910006), nil
	case "HotMap", "CrossTag7d", "CrossTag14d", "CrossTag30d", "ExtendTag", "HotSearch", "RandMap":
		return genericHotLane(recallType, n, userID), nil
	default:
		return nil, fmt.Errorf("recall: unknown RecallType %q", recallType)
	}
}

func liveRedirectLane(n int, userID int64) []recsyskit.ItemInfo {
	base := []int64{910001, 910002, 910003}
	var out []recsyskit.ItemInfo
	for i := 0; i < len(base) && i < n; i++ {
		out = append(out, recsyskit.ItemInfo{ID: recsyskit.ItemID(base[i]), RecallType: "LiveRedirect"})
	}
	_ = userID
	return out
}

func collaborativeLane(n int, userID int64) []recsyskit.ItemInfo {
	var out []recsyskit.ItemInfo
	for i := 0; i < n; i++ {
		id := int64(910004 + (i % 7))
		out = append(out, recsyskit.ItemInfo{ID: recsyskit.ItemID(id), RecallType: "CollaborativeFiltering"})
	}
	_ = userID
	return out
}

func fmDemoItemLane(recallType string, n int, startID int64) []recsyskit.ItemInfo {
	var out []recsyskit.ItemInfo
	for i := 0; i < n; i++ {
		id := startID + int64(i)
		if id > 910010 {
			break
		}
		out = append(out, recsyskit.ItemInfo{ID: recsyskit.ItemID(id), RecallType: recallType})
	}
	return out
}

func genericHotLane(recallType string, n int, userID int64) []recsyskit.ItemInfo {
	var out []recsyskit.ItemInfo
	for i := 0; i < n; i++ {
		id := int64(920001 + i*11 + int(userID%7))
		out = append(out, recsyskit.ItemInfo{ID: recsyskit.ItemID(id), RecallType: recallType})
	}
	return out
}
