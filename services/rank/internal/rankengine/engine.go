package rankengine

import (
	"context"
	"math"
	"sort"

	"recsys_go/pkg/featurestore"
	"recsys_go/services/rank/internal/config"
)

// ScoredItem is one candidate after coarse / fine / re-rank (proto map_info equivalent).
type ScoredItem struct {
	ItemID              int64
	PreRank, Rank, ReRank float32
}

type itemWork struct {
	id     int64
	sparse []SparseFeature
	dense  []float64
	pre    float64
	rank   float64
	re     float64
}

// Engine wires FM coarse + TF-Serving fine + optional re-rank mock.
type Engine struct {
	cfg   config.RankEngineConfig
	fm    *FMModel
	trans *FMTrans
	tf    *TFPredictor
	feat  featurestore.Fetcher
}

// NewEngine loads FM/trans for pipeline mode. feat may be nil (treated as no-op fetcher).
func NewEngine(cfg config.RankEngineConfig, feat featurestore.Fetcher) (*Engine, error) {
	if feat == nil {
		feat = featurestore.NoOp
	}
	e := &Engine{cfg: cfg, feat: feat}
	if cfg.Mode == "pipeline" {
		tr, err := LoadFMTrans(cfg.FM.TransPath)
		if err != nil {
			return nil, err
		}
		e.trans = tr
		if cfg.FM.ModelPath != "" {
			factor := cfg.FM.Factor
			if factor <= 0 {
				factor = 8
			}
			fm, err := LoadFMModel(cfg.FM.ModelPath, factor)
			if err != nil {
				return nil, err
			}
			e.fm = fm
		}
	}
	tf, err := NewTFPredictor(cfg.TFServing)
	if err != nil {
		return nil, err
	}
	e.tf = tf
	return e, nil
}

// RankGroup runs PreRank → Rank → ReRank and returns items sorted by final ReRank descending.
func (e *Engine) RankGroup(ctx context.Context, userID int64, exp []int32, ids []int64, retCount int32) []ScoredItem {
	return e.RankGroupExt(ctx, userID, exp, ids, retCount, 0, 0)
}

// RankGroupExt is like RankGroup but per-request PreRankTrunc/RankTrunc (>0 override server config, C++ AB / funnel).
func (e *Engine) RankGroupExt(ctx context.Context, userID int64, _ []int32, ids []int64, retCount int32, preRankTrunc, rankTrunc int32) []ScoredItem {
	if len(ids) == 0 {
		return nil
	}
	if e.cfg.Mode != "pipeline" {
		return mockRankGroup(userID, ids, retCount)
	}

	work := make([]itemWork, len(ids))
	userJSON, _ := e.feat.UserJSON(ctx, userID)
	for i, id := range ids {
		work[i].id = id
		itemJSON, _ := e.feat.ItemJSON(ctx, id)
		kv, dense, _ := featurestore.MergeUserItemJSON(userJSON, itemJSON)
		if len(kv) > 0 {
			work[i].sparse = make([]SparseFeature, 0, len(kv))
			for _, s := range kv {
				work[i].sparse = append(work[i].sparse, SparseFeature{Key: s.Key, Weight: s.Weight})
			}
			work[i].dense = dense
		} else {
			work[i].sparse = BuildPlaceholderSparse(userID, id, e.trans)
		}
	}

	for i := range work {
		if e.fm != nil {
			work[i].pre = e.fm.Predict(work[i].sparse)
		} else {
			work[i].pre = mockFloat64(userID, work[i].id, 1)
		}
	}
	sort.Slice(work, func(i, j int) bool { return work[i].pre > work[j].pre })
	preN := e.cfg.PreRankTrunc
	if preRankTrunc > 0 {
		preN = int(preRankTrunc)
	}
	if preN > 0 && len(work) > preN {
		work = work[:preN]
	}

	rankN := e.cfg.RankTrunc
	if rankTrunc > 0 {
		rankN = int(rankTrunc)
	}
	if rankN > 0 && len(work) > rankN {
		work = work[:rankN]
	}
	for i := range work {
		if e.tf != nil && e.tf.Configured() {
			vec := work[i].dense
			if len(vec) == 0 {
				vec = denseFeatures(userID, work[i].id, e.tf.FeatureDim)
			}
			if sc, err := e.tf.Predict(ctx, vec); err == nil {
				work[i].rank = sc
			} else {
				work[i].rank = mockFloat64(userID, work[i].id, 2)
			}
		} else {
			// No TF-Serving: use FM PreRank as fine score so Redis/FM ordering stays interpretable in lab.
			work[i].rank = work[i].pre
		}
	}
	sort.Slice(work, func(i, j int) bool { return work[i].rank > work[j].rank })

	for i := range work {
		work[i].re = work[i].rank + 0.01*mockFloat64(userID, work[i].id, 3)
	}
	sort.Slice(work, func(i, j int) bool { return work[i].re > work[j].re })

	out := make([]ScoredItem, 0, len(work))
	for _, w := range work {
		out = append(out, ScoredItem{
			ItemID:  w.id,
			PreRank: float32(w.pre),
			Rank:    float32(w.rank),
			ReRank:  float32(w.re),
		})
	}
	if retCount > 0 && int(retCount) < len(out) {
		out = out[:retCount]
	}
	return out
}

func mockRankGroup(userID int64, ids []int64, retCount int32) []ScoredItem {
	n := len(ids)
	type row struct {
		id  int64
		s   float32
		idx int
	}
	rows := make([]row, n)
	for i, id := range ids {
		rows[i] = row{id: id, s: float32(i + 1), idx: i}
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].s > rows[j].s })
	out := make([]ScoredItem, 0, n)
	for _, r := range rows {
		s := r.s
		out = append(out, ScoredItem{
			ItemID:  r.id,
			PreRank: s,
			Rank:    s,
			ReRank:  s,
		})
	}
	if retCount > 0 && int(retCount) < len(out) {
		out = out[:retCount]
	}
	_ = userID
	return out
}

func mockFloat64(userID, itemID int64, salt int) float64 {
	return float64((userID*31+itemID*17+int64(salt)*13)%1000) / 1000.0
}

func denseFeatures(userID, itemID int64, dim int) []float64 {
	out := make([]float64, dim)
	for i := 0; i < dim; i++ {
		v := float64((userID+itemID*int64(i+3)+int64(i)*7)%997) / 997.0
		out[i] = math.Sin(v) * math.Cos(float64(i+1)/10.0)
	}
	return out
}
