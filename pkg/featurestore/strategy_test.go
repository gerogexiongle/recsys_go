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

func TestParseFeatureLessFlag_missingKey(t *testing.T) {
	if ParseFeatureLessFlag(nil, true) {
		t.Fatal("missing key must not mark feature-less")
	}
}

func TestParseFeatureLessFlag_present(t *testing.T) {
	if !ParseFeatureLessFlag([]byte("1"), false) {
		t.Fatal("expected feature-less")
	}
}

func TestParseItemLabel_missingKey(t *testing.T) {
	if ParseItemLabel(nil, true) != "" {
		t.Fatal("missing label key => empty")
	}
}
