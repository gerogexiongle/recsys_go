package algolog

import (
	"strings"
	"testing"
)

func TestRecordSerialize_fieldCount(t *testing.T) {
	rec := Record{
		ServerName:   "host1",
		Timestamp:    1700000000,
		APIType:      DefaultAPIType,
		DataType:     DefaultDataType,
		RequestID:    "uuid-1",
		UIN:          900001,
		ExpList:      "[0]",
		MaterialList: "[910001:LiveRedirect:1.0000000:1.0000000:0:0.0000000:0.0000000]",
	}
	parts := strings.Split(rec.Serialize(), "|")
	if len(parts) != 28 {
		t.Fatalf("want 28 pipe fields, got %d: %q", len(parts), rec.Serialize())
	}
	if parts[0] != "host1" || parts[19] != "uuid-1" || parts[20] != "900001" {
		t.Fatalf("unexpected fields: %+v", parts)
	}
}

func TestFormatMaterialList(t *testing.T) {
	got := FormatMaterialList([]MaterialEntry{{
		ItemID: 910001, RecallType: "LiveRedirect", RankScore: 1.2, ShowScore: 1.2,
	}}, 18)
	if got != "[910001:LiveRedirect:1.2000000:1.2000000:0:0.0000000:0.0000000]" {
		t.Fatalf("got %q", got)
	}
}
