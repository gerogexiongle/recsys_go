package config

import "testing"

func TestResolvePreRankRankTrunc(t *testing.T) {
	conf := &RankExpConf{
		PreRankExp: []RankExpEntry{{
			ExpID: 0,
			StrategyList: []RankStrategy{{
				Platform: "FMModel", ModelName: "online_prerank_ctr_1", TruncCount: 0,
			}},
		}},
		RankExp: []RankExpEntry{{
			ExpID: 0,
			StrategyList: []RankStrategy{{
				Platform: "TensorFlow", ModelName: "online_model_1", TruncCount: 200,
			}},
		}},
	}
	_, ps := ResolvePreRank(conf, []int32{0})
	if ps == nil || ps.ModelName != "online_prerank_ctr_1" {
		t.Fatalf("pre %+v", ps)
	}
	_, rs := ResolveRank(conf, []int32{0})
	if rs == nil || rs.ModelName != "online_model_1" || rs.TruncCount != 200 {
		t.Fatalf("rank %+v", rs)
	}
	if RankModelBundleKey(ps.ModelName, rs.ModelName) != "online_prerank_ctr_1|online_model_1" {
		t.Fatal(RankModelBundleKey(ps.ModelName, rs.ModelName))
	}
}
