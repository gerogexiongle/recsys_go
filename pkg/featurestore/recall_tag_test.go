package featurestore

import "testing"

func TestAllocateTagRecallCounts(t *testing.T) {
	tags := []TagWeight{{Tag: 3, Weight: 0.6}, {Tag: 4, Weight: 0.4}}
	ids, per := AllocateTagRecallCounts(tags, 10)
	if len(ids) != 2 || per[0]+per[1] != 10 {
		t.Fatalf("ids=%v per=%v", ids, per)
	}
}

func TestCrossTagInterestParse(t *testing.T) {
	raw := []byte(`{"3":0.6,"4":0.4}`)
	tw := ParseTagInterestJSON(raw, false)
	if len(tw) != 2 {
		t.Fatalf("%+v", tw)
	}
}
