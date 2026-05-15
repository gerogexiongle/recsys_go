package recall

import (
	"context"
	"fmt"

	"recsys_go/pkg/recsyskit"
)

// Registry executes RecallMergeRule lanes (mock data aligned with FM 5-feature Redis demo ids).
type Registry struct{}

func NewRegistry() *Registry { return &Registry{} }

// Lane executes one recall rule and returns up to RecallNum synthetic items before merge cap / sample.
func (Registry) Lane(ctx context.Context, rctx recsyskit.RequestContext, rule recsyskit.RecallMergeRule) ([]recsyskit.ItemInfo, error) {
	_ = ctx
	n := rule.RecallNum
	if n <= 0 {
		return nil, nil
	}
	switch rule.RecallType {
	case "LiveRedirect":
		return liveRedirectLane(n, rctx.UserID), nil
	case "CollaborativeFiltering":
		return collaborativeLane(n, rctx.UserID), nil
	case "LiveTag":
		return fmDemoItemLane(rule.RecallType, n, 910006), nil
	case "HotMap", "CrossTag7d", "CrossTag14d", "CrossTag30d", "ExtendTag", "HotSearch", "RandMap":
		return genericHotLane(rule.RecallType, n, rctx.UserID), nil
	default:
		return nil, fmt.Errorf("recall: unknown RecallType %q", rule.RecallType)
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

// fmDemoItemLane recalls from Redis-seeded FM item id range 910001-910010.
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
