package config

import (
	"encoding/json"
	"os"
)

// RankExpConf mirrors online_map_rank Release/config/RankExpConf.json (PreRankExp / RankExp / ReRankExp).
type RankExpConf struct {
	PreRankExp []RankExpEntry `json:"PreRankExp"`
	RankExp    []RankExpEntry `json:"RankExp"`
	ReRankExp  []RankExpEntry `json:"ReRankExp"`
}

// RankExpEntry is one experiment bucket (exp_id + strategy list).
type RankExpEntry struct {
	ExpID         int32          `json:"exp_id"`
	StrategyList  []RankStrategy `json:"StrategyList"`
}

// RankStrategy is one stage strategy (FMModel / TensorFlow / ...).
type RankStrategy struct {
	Platform    string `json:"Platform"`
	FeatureName string `json:"FeatureName"`
	ModelName   string `json:"ModelName"`
	TruncCount  int    `json:"TruncCount"`
}

// LoadRankExpConf loads RankExpConf.json from path.
func LoadRankExpConf(path string) (*RankExpConf, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c RankExpConf
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

func pickRankExpEntry(expIDs []int32, rows []RankExpEntry) *RankExpEntry {
	if len(rows) == 0 {
		return nil
	}
	for _, e := range expIDs {
		for i := range rows {
			if rows[i].ExpID == e {
				return &rows[i]
			}
		}
	}
	for i := range rows {
		if rows[i].ExpID == 0 {
			return &rows[i]
		}
	}
	return &rows[0]
}

// ResolvePreRank picks PreRankExp entry + first strategy (C++ coarse / FMModel).
func ResolvePreRank(conf *RankExpConf, expIDs []int32) (expID int32, strat *RankStrategy) {
	if conf == nil {
		return 0, nil
	}
	ent := pickRankExpEntry(expIDs, conf.PreRankExp)
	if ent == nil || len(ent.StrategyList) == 0 {
		return 0, nil
	}
	return ent.ExpID, &ent.StrategyList[0]
}

// ResolveRank picks RankExp entry + first strategy (C++ fine / TensorFlow + TruncCount + model name).
func ResolveRank(conf *RankExpConf, expIDs []int32) (expID int32, strat *RankStrategy) {
	if conf == nil {
		return 0, nil
	}
	ent := pickRankExpEntry(expIDs, conf.RankExp)
	if ent == nil || len(ent.StrategyList) == 0 {
		return 0, nil
	}
	return ent.ExpID, &ent.StrategyList[0]
}

// ResolveReRank picks ReRankExp entry + first strategy (optional).
func ResolveReRank(conf *RankExpConf, expIDs []int32) (expID int32, strat *RankStrategy) {
	if conf == nil {
		return 0, nil
	}
	ent := pickRankExpEntry(expIDs, conf.ReRankExp)
	if ent == nil || len(ent.StrategyList) == 0 {
		return 0, nil
	}
	return ent.ExpID, &ent.StrategyList[0]
}

// RankModelBundleKey builds the engine map key: coarse FM ModelName | fine TF ModelName (same convention as RankModelBundles yaml keys).
func RankModelBundleKey(preModelName, rankModelName string) string {
	if preModelName == "" {
		preModelName = "_"
	}
	if rankModelName == "" {
		rankModelName = "_"
	}
	return preModelName + "|" + rankModelName
}
