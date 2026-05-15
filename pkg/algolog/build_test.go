package algolog

import (
	"strings"
	"testing"

	"recsys_go/pkg/recsyskit"
)

func TestBuildRecord_matchesCppFields(t *testing.T) {
	rec := BuildRecord(Input{
		UUID:   "req-uuid",
		UserID: 900001,
		ExpIDs: []int32{0, 1},
		Items: []recsyskit.ItemInfo{
			{ID: 910001, RecallType: "LiveRedirect", Score: 1.5},
		},
	})
	if rec.DataType != DefaultDataType || rec.APIType != DefaultAPIType {
		t.Fatalf("rec %+v", rec)
	}
	if !strings.Contains(rec.MaterialList, "910001:LiveRedirect") {
		t.Fatalf("material %q", rec.MaterialList)
	}
	if rec.ExpList != "[0,1]" {
		t.Fatalf("exp %q", rec.ExpList)
	}
}
