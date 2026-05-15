package featurestore

import "testing"

func TestParseExposureJSON_missingKey(t *testing.T) {
	if m := ParseExposureJSON(nil, true); m != nil {
		t.Fatalf("expected nil, got %+v", m)
	}
}

func TestParseExposureJSON_flat(t *testing.T) {
	m := ParseExposureJSON([]byte(`{"910005":15}`), false)
	if m[910005] != 15 {
		t.Fatalf("%+v", m)
	}
}

func TestParseFeatureLessSet_list(t *testing.T) {
	s := ParseFeatureLessSet([]byte(`[910009,910008]`), false)
	if _, ok := s[910009]; !ok {
		t.Fatalf("%+v", s)
	}
}

func TestParseFeatureLessSet_missing(t *testing.T) {
	if ParseFeatureLessSet(nil, true) != nil {
		t.Fatal("missing key => nil set")
	}
}

func TestParseRecallList(t *testing.T) {
	ids := ParseRecallList([]byte(`[910001,910010]`))
	if len(ids) != 2 || ids[0] != 910001 {
		t.Fatalf("%v", ids)
	}
}
