package merge

import "recsys_go/pkg/recsyskit"

// Lanes merges exclusive + main recall batches (dedupe, AllMergeNum cap).
func Lanes(exclusive, main [][]recsyskit.ItemInfo, allMergeNum int) []recsyskit.ItemInfo {
	return recsyskit.MergeRecallLanes(exclusive, main, allMergeNum)
}

// MainOnly merges RecallAndMergeList batches only (C++ main_item_list; exclusive stays separate).
func MainOnly(main [][]recsyskit.ItemInfo, allMergeNum int) []recsyskit.ItemInfo {
	return recsyskit.MergeRecallLanes(nil, main, allMergeNum)
}
