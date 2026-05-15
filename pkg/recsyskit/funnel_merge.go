package recsyskit

// MergeRecallLanes runs exclusive lanes first (higher priority), then main lanes, deduping by item id and capping to allMergeNum.
func MergeRecallLanes(exclusive [][]ItemInfo, main [][]ItemInfo, allMergeNum int) []ItemInfo {
	seen := make(map[ItemID]struct{}, 1024)
	var out []ItemInfo
	appendUnique := func(batch []ItemInfo) {
		for _, it := range batch {
			if _, ok := seen[it.ID]; ok {
				continue
			}
			seen[it.ID] = struct{}{}
			out = append(out, it)
			if allMergeNum > 0 && len(out) >= allMergeNum {
				return
			}
		}
	}
	for _, batch := range exclusive {
		appendUnique(batch)
		if allMergeNum > 0 && len(out) >= allMergeNum {
			return out
		}
	}
	for _, batch := range main {
		appendUnique(batch)
		if allMergeNum > 0 && len(out) >= allMergeNum {
			return out
		}
	}
	return out
}

// ApplySampleFoldAndCap applies SampleFold subsampling then caps to mergeMax (0 = use recall slice length).
func ApplySampleFoldAndCap(items []ItemInfo, sampleFold, mergeMax int) []ItemInfo {
	if len(items) == 0 {
		return nil
	}
	fold := sampleFold
	if fold <= 0 {
		fold = 1
	}
	var picked []ItemInfo
	for i := 0; i < len(items); i += fold {
		picked = append(picked, items[i])
	}
	if mergeMax > 0 && len(picked) > mergeMax {
		picked = picked[:mergeMax]
	}
	return picked
}
