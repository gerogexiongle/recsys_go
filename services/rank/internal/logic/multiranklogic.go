package logic

import (
	"context"

	"recsys_go/pkg/recsyskit/transporthttp"
	"recsys_go/services/rank/internal/config"
	"recsys_go/services/rank/internal/rankengine"
)

// rankServeEnv is rank handler dependencies (engine registry + optional RankExpConf).
type rankServeEnv interface {
	EngineFor(profile string) *rankengine.Engine
	RankExpConf() *config.RankExpConf
}

// MultiRank runs coarse (FM) → fine (TF-Serving) → re-rank (mock) via rankengine.Engine.
type MultiRank struct {
	env rankServeEnv
}

func NewMultiRank(env rankServeEnv) *MultiRank {
	return &MultiRank{env: env}
}

func (l *MultiRank) Handle(ctx context.Context, req *transporthttp.MultiRankRequestJSON) (*transporthttp.MultiRankResponseJSON, error) {
	preTrunc := req.PreRankTrunc
	rankTrunc := req.RankTrunc
	profile := req.RankProfile

	var preExpID, rankExpID, reExpID int32
	var ps, rs *config.RankStrategy

	if rc := l.env.RankExpConf(); rc != nil {
		preExpID, ps = config.ResolvePreRank(rc, req.ExpIDs)
		rankExpID, rs = config.ResolveRank(rc, req.ExpIDs)
		reExpID, _ = config.ResolveReRank(rc, req.ExpIDs)

		if profile == "" && ps != nil && rs != nil && ps.ModelName != "" && rs.ModelName != "" {
			profile = config.RankModelBundleKey(ps.ModelName, rs.ModelName)
		}
		if preTrunc <= 0 && ps != nil && ps.TruncCount > 0 {
			preTrunc = int32(ps.TruncCount)
		}
		if rankTrunc <= 0 && rs != nil && rs.TruncCount > 0 {
			rankTrunc = int32(rs.TruncCount)
		}
	} else {
		e0 := firstExp(req.ExpIDs)
		preExpID, rankExpID, reExpID = e0, e0, e0
	}

	eng := l.env.EngineFor(profile)
	if eng == nil {
		eng = l.env.EngineFor("")
	}
	resp := &transporthttp.MultiRankResponseJSON{
		UUID:   req.UUID,
		UserID: req.UserID,
		Exp: transporthttp.ExpInfoJSON{
			PreRankExpID: preExpID,
			RankExpID:    rankExpID,
			ReRankExpID:  reExpID,
		},
	}
	for _, g := range req.ItemGroups {
		scored := eng.RankGroupExt(ctx, req.UserID, req.ExpIDs, g.ItemIDs, g.RetCount, preTrunc, rankTrunc)
		rg := transporthttp.RankedGroupJSON{Name: g.Name}
		for _, s := range scored {
			rg.ItemScores = append(rg.ItemScores, transporthttp.ItemScoresJSON{
				ItemID:       s.ItemID,
				PreRankScore: s.PreRank,
				RankScore:    s.Rank,
				ReRankScore:  s.ReRank,
			})
		}
		resp.RankedGroups = append(resp.RankedGroups, rg)
	}
	return resp, nil
}

func firstExp(exp []int32) int32 {
	if len(exp) == 0 {
		return 0
	}
	return exp[0]
}
