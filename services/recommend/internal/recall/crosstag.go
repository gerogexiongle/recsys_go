package recall

import (
	"context"

	"recsys_go/pkg/featurestore"
	"recsys_go/pkg/recsyskit"
)

// crossTagLane implements C++ TagRecall for CrossTag7d/14d/30d:
// user tag interest (personalized) -> weight split -> tag invert index (per tag id).
func crossTagLane(
	ctx context.Context,
	fetch featurestore.RecallFetcher,
	userID int64,
	rule recsyskit.RecallMergeRule,
) ([]recsyskit.ItemInfo, bool) {
	if fetch == nil {
		return nil, false
	}
	window := featurestore.TagInterestWindow(rule.RecallType)
	raw, missing, err := fetch.UserTagInterestJSON(ctx, window, userID)
	if err != nil || missing {
		return nil, false
	}
	tags := featurestore.ParseTagInterestJSON(raw, missing)
	if len(tags) == 0 {
		return nil, false
	}
	tags = featurestore.TruncateTopKTags(tags, rule.UseTopKIndex)
	budget := featurestore.RoundUpRecallBudget(rule.RecallNum, rule.SampleFold)
	tagIDs, perTag := featurestore.AllocateTagRecallCounts(tags, budget)
	if len(tagIDs) == 0 {
		return nil, false
	}
	var pooled []int64
	for i, tagID := range tagIDs {
		invRaw, invMiss, err := fetch.TagInvertJSON(ctx, tagID)
		if err != nil || invMiss {
			continue
		}
		ids := featurestore.ParseRecallList(invRaw)
		pooled = append(pooled, featurestore.SamplePrefix(ids, perTag[i])...)
	}
	pooled = featurestore.DedupeItemIDsStable(pooled)
	if len(pooled) == 0 {
		return nil, false
	}
	if rule.RecallNum > 0 && len(pooled) > rule.RecallNum {
		pooled = pooled[:rule.RecallNum]
	}
	out := make([]recsyskit.ItemInfo, len(pooled))
	for i, id := range pooled {
		out[i] = recsyskit.ItemInfo{ID: recsyskit.ItemID(id), RecallType: rule.RecallType}
	}
	return out, true
}
