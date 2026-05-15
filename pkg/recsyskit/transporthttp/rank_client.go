package transporthttp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"recsys_go/pkg/recsyskit"
)

// RankHTTPClient implements recsyskit.RankClient over JSON HTTP.
type RankHTTPClient struct {
	BaseURL    string
	HTTPClient *http.Client
	Path       string
}

// NewRankHTTPClient returns a client posting to baseURL + path (default "/v1/rank/multi").
func NewRankHTTPClient(baseURL string, timeout time.Duration) *RankHTTPClient {
	if timeout <= 0 {
		timeout = 800 * time.Millisecond
	}
	return &RankHTTPClient{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: timeout,
		},
		Path: "/v1/rank/multi",
	}
}

// MultiRank implements recsyskit.RankClient.
func (c *RankHTTPClient) MultiRank(ctx context.Context, req *recsyskit.MultiRankRequest) (*recsyskit.MultiRankResponse, error) {
	if c == nil || c.BaseURL == "" {
		return nil, fmt.Errorf("rank http client: empty base url")
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
	url := c.BaseURL + c.Path
	hreq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	hreq.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTPClient.Do(hreq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("rank http: status %d body %s", resp.StatusCode, string(b))
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
