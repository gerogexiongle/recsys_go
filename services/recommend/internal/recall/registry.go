package recall

import (
	"context"
	"fmt"

	"recsys_go/pkg/featurestore"
	"recsys_go/pkg/recsyskit"
)

// Registry dispatches RecallType to lane implementations (Redis-first, stub fallback).
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
	if rule.RecallType == "LiveRedirect" {
		if items, ok := liveRedirectFromUser(rctx.UserFeat, n); ok {
			return items, nil
		}
	}
	if reg.recall != nil {
		if featurestore.IsCrossTagRecallType(rule.RecallType) {
			if items, ok := crossTagLane(ctx, reg.recall, rctx.UserID, rule); ok {
				return items, nil
			}
		}
		if items, ok := laneFromRedisList(ctx, reg.recall, rctx.UserID, rule.RecallType, n); ok {
			return items, nil
		}
	}
	return reg.laneStub(rule.RecallType, n, rctx.UserID, rctx.UserGroup)
}

func (reg *Registry) laneStub(recallType string, n int, userID int64, userGroup string) ([]recsyskit.ItemInfo, error) {
	switch recallType {
	case "LiveRedirect":
		return liveRedirectLane(n), nil
	case "CollaborativeFiltering":
		return collaborativeLane(n, userID), nil
	case "LiveTag":
		return fmDemoItemLane(recallType, n, 910006), nil
	case "HotMap", "HotSearch":
		return hotLaneForGroup(recallType, n, userGroup), nil
	case "NewUser_Hot", "NewUser_HighRetention":
		if userGroup == UserGroupNewUser {
			return newUserLane(recallType, n), nil
		}
		return hotLaneForGroup(recallType, n, userGroup), nil
	case "CrossTag7d", "CrossTag14d", "CrossTag30d", "ExtendTag", "RandMap":
		return genericHotLane(recallType, n, userID), nil
	default:
		return nil, fmt.Errorf("recall: unknown RecallType %q", recallType)
	}
}

func liveRedirectLane(n int) []recsyskit.ItemInfo {
	base := []int64{910001, 910002, 910003}
	var out []recsyskit.ItemInfo
	for i := 0; i < len(base) && i < n; i++ {
		out = append(out, recsyskit.ItemInfo{ID: recsyskit.ItemID(base[i]), RecallType: "LiveRedirect"})
	}
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

func hotLaneForGroup(recallType string, n int, userGroup string) []recsyskit.ItemInfo {
	if userGroup == UserGroupNewUser {
		return newUserLane(recallType, n)
	}
	return genericHotLane(recallType, n, 0)
}

func newUserLane(recallType string, n int) []recsyskit.ItemInfo {
	var base []int64
	switch recallType {
	case "NewUser_Hot":
		base = []int64{910002, 910003, 910004}
	case "NewUser_HighRetention":
		base = []int64{910001, 910010}
	case "HotMap":
		base = []int64{910002, 910003, 910004, 910010}
	default:
		base = []int64{910002, 910003}
	}
	var out []recsyskit.ItemInfo
	for i := 0; i < len(base) && i < n; i++ {
		out = append(out, recsyskit.ItemInfo{ID: recsyskit.ItemID(base[i]), RecallType: recallType})
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
