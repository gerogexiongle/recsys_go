package featurestore

import (
	"encoding/json"
	"strconv"
)

// ParseHomogenExchange reads item_id -> homogen_item_id (C++ ExchangeOnlineHomogenMap).
func ParseHomogenExchange(raw []byte, keyMissing bool) map[int64]int64 {
	if keyMissing || len(raw) == 0 {
		return nil
	}
	var flat map[string]int64
	if err := json.Unmarshal(raw, &flat); err != nil {
		return nil
	}
	out := make(map[int64]int64, len(flat))
	for k, v := range flat {
		id, err := strconv.ParseInt(k, 10, 64)
		if err != nil || v <= 0 {
			continue
		}
		out[id] = v
	}
	return out
}
