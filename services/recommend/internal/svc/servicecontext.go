package svc

import (
	"path/filepath"
	"time"

	"recsys_go/pkg/featurestore"
	"recsys_go/pkg/kafkapush"
	"recsys_go/pkg/recsyskit"
	"recsys_go/pkg/recsyskit/transporthttp"
	"recsys_go/services/recommend/internal/centerconfig"
	"recsys_go/services/recommend/internal/config"
	"recsys_go/services/recommend/internal/recall"
)

type ServiceContext struct {
	Config   config.Config
	Rank     recsyskit.RankClient
	Features featurestore.Fetcher
	Funnel   *recsyskit.FunnelLibrary
	Center    *centerconfig.CenterBundle
	Recall    *recall.Registry
	AlgoKafka *kafkapush.Pool
}

func NewServiceContext(c config.Config, configFilePath string) (*ServiceContext, error) {
	timeout := time.Duration(c.RankService.TimeoutMs) * time.Millisecond
	if c.RankService.TimeoutMs <= 0 {
		timeout = 800 * time.Millisecond
	}
	var rank recsyskit.RankClient
	if len(c.RankEndpoints().Resolve()) > 0 {
		rc, err := transporthttp.NewRankHTTPClient(c.RankEndpoints(), timeout)
		if err != nil {
			return nil, err
		}
		rank = rc
	}
	fetch, err := featurestore.NewFetcher(c.FeatureRedis)
	if err != nil {
		return nil, err
	}
	baseDir := filepath.Dir(configFilePath)

	var center *centerconfig.CenterBundle
	if c.CenterRecallPath != "" {
		p := c.CenterRecallPath
		if !filepath.IsAbs(p) {
			p = filepath.Join(baseDir, p)
		}
		rl, err := centerconfig.LoadRecallLibrary(p)
		if err != nil {
			return nil, err
		}
		center = &centerconfig.CenterBundle{Recall: rl}
		if c.CenterFilterPath != "" {
			fp := c.CenterFilterPath
			if !filepath.IsAbs(fp) {
				fp = filepath.Join(baseDir, fp)
			}
			fl, err := centerconfig.LoadFilterLibrary(fp)
			if err != nil {
				return nil, err
			}
			center.Filter = fl
		}
		if c.CenterShowControlPath != "" {
			sp := c.CenterShowControlPath
			if !filepath.IsAbs(sp) {
				sp = filepath.Join(baseDir, sp)
			}
			sl, err := centerconfig.LoadShowLibrary(sp)
			if err != nil {
				return nil, err
			}
			center.Show = sl
		}
	}

	var funnel *recsyskit.FunnelLibrary
	if center == nil && c.FunnelConfigPath != "" {
		p := c.FunnelConfigPath
		if !filepath.IsAbs(p) {
			if configFilePath != "" {
				p = filepath.Join(baseDir, p)
			}
		}
		lib, err := recsyskit.LoadFunnelLibrary(p)
		if err != nil {
			return nil, err
		}
		funnel = lib
	}
	var rec *recall.Registry
	if funnel != nil || center != nil {
		var rf featurestore.RecallFetcher
		if rff, ok := fetch.(featurestore.RecallFetcher); ok {
			rf = rff
		}
		rec = recall.NewRegistry(rf)
	}
	algoKafka, err := kafkapush.New(c.KafkaPush)
	if err != nil {
		return nil, err
	}
	return &ServiceContext{Config: c, Rank: rank, Features: fetch, Funnel: funnel, Center: center, Recall: rec, AlgoKafka: algoKafka}, nil
}
