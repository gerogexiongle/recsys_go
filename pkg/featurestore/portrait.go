package featurestore

import (
	"context"

	"recsys_go/pkg/recsyskit"
)

// MarkItemPortraits sets HasPortrait from feat:item Redis keys (MGET).
func MarkItemPortraits(ctx context.Context, fetch Fetcher, items []recsyskit.ItemInfo) ([]recsyskit.ItemInfo, error) {
	if len(items) == 0 || fetch == nil || fetch == NoOp {
		out := make([]recsyskit.ItemInfo, len(items))
		for i, it := range items {
			out[i] = it
			out[i].HasPortrait = true
		}
		return out, nil
	}
	ids := make([]int64, len(items))
	for i := range items {
		ids[i] = int64(items[i].ID)
	}
	var prof map[int64][]byte
	var err error
	if bf, ok := fetch.(BatchFetcher); ok {
		prof, err = bf.ItemsJSON(ctx, ids)
	} else {
		prof = make(map[int64][]byte, len(ids))
		for _, id := range ids {
			b, e := fetch.ItemJSON(ctx, id)
			if e != nil {
				return items, e
			}
			if len(b) > 0 {
				prof[id] = b
			}
		}
	}
	if err != nil {
		return items, err
	}
	out := make([]recsyskit.ItemInfo, len(items))
	for i, it := range items {
		out[i] = it
		_, ok := prof[int64(it.ID)]
		out[i].HasPortrait = ok && len(prof[int64(it.ID)]) > 0
	}
	return out, nil
}

// DropWithoutPortrait removes items that cannot be ranked (no item feat key).
func DropWithoutPortrait(items []recsyskit.ItemInfo) []recsyskit.ItemInfo {
	if len(items) == 0 {
		return items
	}
	out := make([]recsyskit.ItemInfo, 0, len(items))
	for _, it := range items {
		if it.HasPortrait {
			out = append(out, it)
		}
	}
	return out
}
