package featurestore

import (
	"encoding/json"
	"testing"
	"time"
)

func TestParseLiveRedirectItems_validityAndSort(t *testing.T) {
	now := time.Now().Unix()
	raw, _ := json.Marshal(map[string]any{
		"live_redirect": map[string]any{
			"map_list": []map[string]any{
				{"id": 910002, "ts": now - 100, "weight": 1},
				{"id": 910001, "ts": now, "weight": 1},
				{"id": 910099, "ts": now - 90000, "weight": 1},
			},
		},
	})
	items := ParseLiveRedirectItems(raw, 5)
	if len(items) != 2 {
		t.Fatalf("want 2 valid, got %d", len(items))
	}
	if items[0].ID != 910001 || items[1].ID != 910002 {
		t.Fatalf("order by ts desc: %+v", items)
	}
	if items[0].RecallType != "LiveRedirect" {
		t.Fatal(items[0].RecallType)
	}
}
