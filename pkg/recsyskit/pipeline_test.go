package recsyskit

import "testing"

func TestReorderByRank(t *testing.T) {
	items := []ItemInfo{{ID: 10001}, {ID: 10002}, {ID: 10003}}
	scores := []ItemScores{
		{ItemID: 10003, RankScore: 3},
		{ItemID: 10002, RankScore: 2},
		{ItemID: 10001, RankScore: 1},
	}
	got := reorderByRank(items, scores)
	if len(got) != 3 || got[0].ID != 10003 || got[1].ID != 10002 || got[2].ID != 10001 {
		t.Fatalf("got ids %v %v %v", got[0].ID, got[1].ID, got[2].ID)
	}
}
