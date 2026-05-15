package featurestore

import (
	"encoding/json"
	"sort"
	"strconv"
)

type TagWeight struct {
	Tag    int     `json:"tag"`
	Weight float64 `json:"weight"`
}

func ParseTagInterestJSON(raw []byte, keyMissing bool) []TagWeight {
	if keyMissing || len(raw) == 0 {
		return nil
	}
	var list []TagWeight
	if err := json.Unmarshal(raw, &list); err == nil && len(list) > 0 {
		return filterPositiveTagWeights(list)
	}
	var flat map[string]float64
	if err := json.Unmarshal(raw, &flat); err == nil {
		for k, w := range flat {
			t, err := strconv.Atoi(k)
			if err == nil && t >= 0 && w > 1e-6 {
				list = append(list, TagWeight{Tag: t, Weight: w})
			}
		}
		return filterPositiveTagWeights(list)
	}
	return nil
}

func filterPositiveTagWeights(in []TagWeight) []TagWeight {
	var out []TagWeight
	for _, tw := range in {
		if tw.Tag >= 0 && tw.Weight > 1e-6 {
			out = append(out, tw)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Weight > out[j].Weight })
	return out
}

func AllocateTagRecallCounts(tags []TagWeight, recallNum int) (tagIDs []int, perTag []int) {
	if recallNum <= 0 || len(tags) == 0 {
		return nil, nil
	}
	var sum float64
	for _, t := range tags {
		sum += t.Weight
	}
	if sum < 1e-6 {
		return nil, nil
	}
	tagIDs = make([]int, len(tags))
	perTag = make([]int, len(tags))
	unit := float64(recallNum) / sum
	remain := recallNum
	for i, t := range tags {
		tagIDs[i] = t.Tag
		n := int(t.Weight*unit + 0.5)
		if n > 50 {
			n = 50
		}
		if n > remain {
			n = remain
		}
		perTag[i] = n
		remain -= n
	}
	return tagIDs, perTag
}

func ParseItemTag(itemJSON []byte) int {
	if len(itemJSON) == 0 {
		return -1
	}
	var doc struct {
		Tag int `json:"tag"`
	}
	if err := json.Unmarshal(itemJSON, &doc); err == nil && doc.Tag >= 0 {
		return doc.Tag
	}
	return -1
}

func RoundUpRecallBudget(recallNum, sampleFold int) int {
	if recallNum <= 0 {
		return 0
	}
	if sampleFold <= 0 {
		sampleFold = 1
	}
	return recallNum * sampleFold
}

func TruncateTopKTags(tags []TagWeight, topK int) []TagWeight {
	if topK <= 0 || len(tags) <= topK {
		return tags
	}
	return tags[:topK]
}

func DedupeItemIDsStable(ids []int64) []int64 {
	seen := make(map[int64]struct{}, len(ids))
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func SamplePrefix(ids []int64, n int) []int64 {
	if n <= 0 || len(ids) == 0 {
		return nil
	}
	if len(ids) <= n {
		return append([]int64(nil), ids...)
	}
	return append([]int64(nil), ids[:n]...)
}

func TagInterestWindow(recallType string) string {
	switch recallType {
	case "CrossTag14d":
		return "14d"
	case "CrossTag30d":
		return "30d"
	default:
		return "7d"
	}
}

func IsCrossTagRecallType(recallType string) bool {
	switch recallType {
	case "CrossTag7d", "CrossTag14d", "CrossTag30d":
		return true
	default:
		return false
	}
}
