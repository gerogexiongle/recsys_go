package featurestore

import (
	"encoding/json"
	"sort"
	"time"

	"recsys_go/pkg/recsyskit"
)

const liveRedirectValiditySec = 86400

type liveRedirectEntry struct {
	ID     int64   `json:"id"`
	Ts     int64   `json:"ts"`
	Weight float64 `json:"weight"`
}

type liveRedirectBlob struct {
	MapList []liveRedirectEntry `json:"map_list"`
}

// ParseLiveRedirectItems reads UserDataLiveRedirect-style JSON from user feat (C++ FeatureRecall::GetLiveRedirectSampleData).
func ParseLiveRedirectItems(userJSON []byte, limit int) []recsyskit.ItemInfo {
	if len(userJSON) == 0 || limit <= 0 {
		return nil
	}
	var root map[string]json.RawMessage
	if err := json.Unmarshal(userJSON, &root); err != nil {
		return nil
	}
	raw, ok := root["live_redirect"]
	if !ok {
		return nil
	}
	var blob liveRedirectBlob
	if err := json.Unmarshal(raw, &blob); err != nil || len(blob.MapList) == 0 {
		return nil
	}
	now := time.Now().Unix()
	type pair struct {
		id int64
		ts int64
	}
	var valid []pair
	for _, e := range blob.MapList {
		if e.ID <= 0 {
			continue
		}
		if e.Ts > 0 && now-e.Ts >= liveRedirectValiditySec {
			continue
		}
		valid = append(valid, pair{id: e.ID, ts: e.Ts})
	}
	if len(valid) == 0 {
		return nil
	}
	sort.Slice(valid, func(i, j int) bool { return valid[i].ts > valid[j].ts })
	if len(valid) > limit {
		valid = valid[:limit]
	}
	out := make([]recsyskit.ItemInfo, len(valid))
	for i, p := range valid {
		out[i] = recsyskit.ItemInfo{ID: recsyskit.ItemID(p.id), RecallType: "LiveRedirect"}
	}
	return out
}
