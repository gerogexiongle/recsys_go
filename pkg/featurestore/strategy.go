package featurestore

import (
	"encoding/json"
	"strconv"
)

// ParseExposureJSON reads item-level exposure map (recsysgo:filter:exposure).
func ParseExposureJSON(raw []byte, keyMissing bool) map[int64]int {
	if keyMissing || len(raw) == 0 {
		return nil
	}
	var flat map[string]int
	if err := json.Unmarshal(raw, &flat); err == nil && len(flat) > 0 {
		return parseExposureMap(flat)
	}
	var items []struct {
		ItemID int64 `json:"item_id"`
		Count  int   `json:"count"`
	}
	if err := json.Unmarshal(raw, &items); err == nil && len(items) > 0 {
		out := make(map[int64]int, len(items))
		for _, it := range items {
			out[it.ItemID] = it.Count
		}
		return out
	}
	return nil
}

func parseExposureMap(m map[string]int) map[int64]int {
	out := make(map[int64]int, len(m))
	for k, v := range m {
		id, err := strconv.ParseInt(k, 10, 64)
		if err == nil {
			out[id] = v
		}
	}
	return out
}

// ParseFeatureLessSet returns item ids to drop by FeatureLess filter.
// keyMissing => empty set => no item marked feature-less.
func ParseFeatureLessSet(raw []byte, keyMissing bool) map[int64]struct{} {
	if keyMissing || len(raw) == 0 {
		return nil
	}
	var ids []int64
	if err := json.Unmarshal(raw, &ids); err == nil {
		return int64Set(ids)
	}
	var asString []string
	if err := json.Unmarshal(raw, &asString); err == nil {
		out := make(map[int64]struct{}, len(asString))
		for _, s := range asString {
			if id, err := strconv.ParseInt(s, 10, 64); err == nil {
				out[id] = struct{}{}
			}
		}
		return out
	}
	var wrap struct {
		Items []int64 `json:"items"`
	}
	if err := json.Unmarshal(raw, &wrap); err == nil {
		return int64Set(wrap.Items)
	}
	return nil
}

// ParseLabelMap item_id -> label for LabelTypeWhiteList.
func ParseLabelMap(raw []byte, keyMissing bool) map[int64]string {
	if keyMissing || len(raw) == 0 {
		return nil
	}
	var flat map[string]string
	if err := json.Unmarshal(raw, &flat); err == nil && len(flat) > 0 {
		out := make(map[int64]string, len(flat))
		for k, v := range flat {
			if id, err := strconv.ParseInt(k, 10, 64); err == nil {
				out[id] = v
			}
		}
		return out
	}
	return nil
}

// ParseRecallList reads lane / CF JSON item id list.
func ParseRecallList(raw []byte) []int64 {
	if len(raw) == 0 {
		return nil
	}
	var ids []int64
	if err := json.Unmarshal(raw, &ids); err == nil {
		return ids
	}
	var objs []struct {
		ItemID int64 `json:"item_id"`
	}
	if err := json.Unmarshal(raw, &objs); err == nil {
		out := make([]int64, 0, len(objs))
		for _, o := range objs {
			out = append(out, o.ItemID)
		}
		return out
	}
	return nil
}

func int64Set(ids []int64) map[int64]struct{} {
	out := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		out[id] = struct{}{}
	}
	return out
}
