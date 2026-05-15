package featurestore

import "testing"

func TestMergeSemanticFiveFields(t *testing.T) {
	u := []byte(`{"age":32,"gender":1,"income_wan":5.5}`)
	i := []byte(`{"ctr_7d":0.04,"revenue_7d":3500}`)
	sp, _, err := MergeUserItemJSON(u, i)
	if err != nil {
		t.Fatal(err)
	}
	if len(sp) != 5 {
		t.Fatalf("want 5 slots got %d %+v", len(sp), sp)
	}
	want := []struct {
		key int64
		min float64
		max float64
	}{
		{fmSlotKey(1), 0.31, 0.33},
		{fmSlotKey(2), 0.99, 1.01},
		{fmSlotKey(3), 0.54, 0.56},
		{fmSlotKey(4), 0.039, 0.041},
		{fmSlotKey(5), 0.034, 0.036},
	}
	for i, w := range want {
		if sp[i].Key != w.key {
			t.Fatalf("slot %d key %d want %d", i, sp[i].Key, w.key)
		}
		if sp[i].Weight < w.min || sp[i].Weight > w.max {
			t.Fatalf("slot %d weight %v not in [%v,%v]", i, sp[i].Weight, w.min, w.max)
		}
	}
}

func TestMergeSemanticNestedSegments(t *testing.T) {
	u := []byte(`{"user_profile":{"age":40,"gender":0},"user_finance":{"income_wan":7}}`)
	i := []byte(`{"item_stats":{"ctr_7d":0.08,"revenue_7d":120000}}`)
	sp, _, err := MergeUserItemJSON(u, i)
	if err != nil {
		t.Fatal(err)
	}
	if len(sp) != 5 {
		t.Fatalf("want 5 got %d %+v", len(sp), sp)
	}
	if sp[3].Weight < 0.079 || sp[3].Weight > 0.081 {
		t.Fatalf("ctr weight %v", sp[3].Weight)
	}
	if sp[4].Weight < 1.19 || sp[4].Weight > 1.21 {
		t.Fatalf("rev weight %v", sp[4].Weight)
	}
}
