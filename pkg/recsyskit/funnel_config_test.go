package recsyskit

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestResolveFunnelAB(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller")
	}
	p := filepath.Join(filepath.Dir(file), "..", "..", "services", "recommend", "etc", "recommend-funnel.json")
	lib, err := LoadFunnelLibrary(p)
	if err != nil {
		t.Fatal(err)
	}
	g0 := lib.ResolveFunnel([]int32{0}, "def_group")
	if g0 == nil || g0.AllMergeNum != 2000 {
		t.Fatalf("def exp0 %+v", g0)
	}
	g1 := lib.ResolveFunnel([]int32{1}, "def_group")
	if g1 == nil || g1.AllMergeNum != 500 {
		t.Fatalf("exp1 %+v", g1)
	}
}

func TestMergeRecallLanesCap(t *testing.T) {
	ex := [][]ItemInfo{{{ID: 1}}, {{ID: 2}, {ID: 3}}}
	main := [][]ItemInfo{{{ID: 3}, {ID: 4}}}
	got := MergeRecallLanes(ex, main, 3)
	if len(got) != 3 || got[0].ID != 1 || got[1].ID != 2 || got[2].ID != 3 {
		t.Fatalf("%+v", got)
	}
}
