package featurestore

import (
	"encoding/json"
	"strconv"
	"strings"
)

// ParseExposureJSON reads LiveExposure side data (C++ user_feature field uin:game_exposure).
// exposureKeyMissing=true when Redis returned nil — treat as no exposure, do not filter by exposure.
func ParseExposureJSON(raw []byte, exposureKeyMissing bool) map[int64]int {
	if exposureKeyMissing || len(raw) == 0 {
		return nil
	}
	var flat map[string]int
	if err := json.Unmarshal(raw, &flat); err == nil && len(flat) > 0 {
		return parseExposureMap(flat)
	}
	var wrap struct {
		Exposure map[string]int `json:"exposure"`
	}
	if err := json.Unmarshal(raw, &wrap); err == nil && len(wrap.Exposure) > 0 {
		return parseExposureMap(wrap.Exposure)
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

// ParseFeatureLessFlag returns true only when the strategy key exists and marks feature-less.
// keyMissing=true => item has features (FeatureLess filter keeps the item).
func ParseFeatureLessFlag(raw []byte, keyMissing bool) bool {
	if keyMissing || len(raw) == 0 {
		return false
	}
	s := strings.TrimSpace(string(raw))
	if s == "1" || strings.EqualFold(s, "true") {
		return true
	}
	var doc struct {
		FeatureLess string `json:"feature_less"`
	}
	if err := json.Unmarshal(raw, &doc); err == nil {
		return doc.FeatureLess == "1" || strings.EqualFold(doc.FeatureLess, "true")
	}
	return false
}

// ParseItemLabel returns label for LabelTypeWhiteList; keyMissing => empty (no match).
func ParseItemLabel(raw []byte, keyMissing bool) string {
	if keyMissing || len(raw) == 0 {
		return ""
	}
	s := strings.TrimSpace(string(raw))
	if s == "" {
		return ""
	}
	if s[0] == '{' {
		var doc struct {
			Label string `json:"label"`
		}
		if err := json.Unmarshal(raw, &doc); err == nil {
			return doc.Label
		}
		return ""
	}
	return s
}
