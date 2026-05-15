package transporthttp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"recsys_go/pkg/recsyskit"
	"recsys_go/pkg/upstream"
)

// RankHTTPClient implements recsyskit.RankClient over JSON HTTP with multi-endpoint LB.
type RankHTTPClient struct {
	doer *upstream.HTTPDoer
	Path string
}

// NewRankHTTPClientSingle is a shortcut for a single rank-api instance (tests / dev).
func NewRankHTTPClientSingle(baseURL string, timeout time.Duration) (*RankHTTPClient, error) {
	return NewRankHTTPClient(upstream.EndpointsConfig{BaseURL: baseURL}, timeout)
}

// NewRankHTTPClient returns a client for one or many rank-api base URLs.
// Prefer Endpoints for multi-instance; BaseURL alone remains valid for single instance / K8s Service.
func NewRankHTTPClient(eps upstream.EndpointsConfig, timeout time.Duration) (*RankHTTPClient, error) {
	doer, err := upstream.NewHTTPDoer(eps, timeout)
	if err != nil {
		return nil, err
	}
	return &RankHTTPClient{doer: doer, Path: "/v1/rank/multi"}, nil
}

// MultiRank implements recsyskit.RankClient.
func (c *RankHTTPClient) MultiRank(ctx context.Context, req *recsyskit.MultiRankRequest) (*recsyskit.MultiRankResponse, error) {
	if c == nil || c.doer == nil {
		return nil, fmt.Errorf("rank http client: not configured")
	}
	body := MultiRankRequestJSON{
		UUID:            req.Ctx.UUID,
		UserID:          req.Ctx.UserID,
		Section:         req.Ctx.Section,
		ExpIDs:          append([]int32(nil), req.Ctx.ExpIDs...),
		DisablePersonal: req.Ctx.DisablePersonal,
		DeviceID:        req.Ctx.DeviceID,
		TerminalModel:   req.Ctx.TerminalModel,
		OS:              req.Ctx.OSType,
	}
	for _, g := range req.Groups {
		ids := make([]int64, len(g.ItemIDs))
		for i, id := range g.ItemIDs {
			ids[i] = int64(id)
		}
		body.ItemGroups = append(body.ItemGroups, ItemGroupJSON{
			Name:     g.Name,
			ItemIDs:  ids,
			RetCount: g.RetCount,
		})
	}
	body.PreRankTrunc = req.PreRankTrunc
	body.RankTrunc = req.RankTrunc
	body.RankProfile = req.RankProfile
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	b, err := c.doer.PostBytes(ctx, c.Path, raw, "application/json")
	if err != nil {
		return nil, fmt.Errorf("rank http: %w", err)
	}
	var wire MultiRankResponseJSON
	if err := json.Unmarshal(b, &wire); err != nil {
		return nil, err
	}
	if len(body.ItemGroups) > 0 && len(wire.RankedGroups) == 0 {
		return nil, fmt.Errorf("rank response missing ranked_groups for non-empty request (body=%s)", string(b))
	}
	out := &recsyskit.MultiRankResponse{
		UUID:   wire.UUID,
		UserID: wire.UserID,
		Exp: recsyskit.ExpInfo{
			PreRankExpID: wire.Exp.PreRankExpID,
			RankExpID:    wire.Exp.RankExpID,
			ReRankExpID:  wire.Exp.ReRankExpID,
		},
	}
	for _, g := range wire.RankedGroups {
		rg := recsyskit.RankedItemGroup{Name: g.Name}
		for _, s := range g.ItemScores {
			rg.Items = append(rg.Items, recsyskit.ItemScores{
				ItemID:       recsyskit.ItemID(s.ItemID),
				PreRankScore: s.PreRankScore,
				RankScore:    s.RankScore,
				ReRankScore:  s.ReRankScore,
			})
		}
		out.Groups = append(out.Groups, rg)
	}
	if len(req.Groups) > 0 && len(req.Groups[0].ItemIDs) > 0 {
		if len(out.Groups) == 0 || len(out.Groups[0].Items) == 0 {
			return nil, fmt.Errorf("rank returned empty item_scores (http body=%s)", string(b))
		}
	}
	return out, nil
}
