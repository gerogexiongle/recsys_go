package filter

import "recsys_go/pkg/recsyskit"

// ApplyFeatureLess drops items without item portrait (C++ FeatureLessFilter: missing map feature).
func ApplyFeatureLess(items []recsyskit.ItemInfo) []recsyskit.ItemInfo {
	var out []recsyskit.ItemInfo
	for _, it := range items {
		if it.HasPortrait {
			out = append(out, it)
		}
	}
	return out
}
