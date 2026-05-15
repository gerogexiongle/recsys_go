package svc

import (
	"fmt"
	"path/filepath"

	"recsys_go/pkg/featurestore"
	"recsys_go/services/rank/internal/config"
	"recsys_go/services/rank/internal/rankengine"
)

type ServiceContext struct {
	Config    config.Config
	engines   map[string]*rankengine.Engine
	rankExp   *config.RankExpConf
}

func (s *ServiceContext) RankExpConf() *config.RankExpConf {
	if s == nil {
		return nil
	}
	return s.rankExp
}

// EngineFor returns the engine for rank_profile or RankModelBundleKey (empty / unknown -> "default").
func (s *ServiceContext) EngineFor(profile string) *rankengine.Engine {
	if s == nil || s.engines == nil {
		return nil
	}
	if profile != "" {
		if e, ok := s.engines[profile]; ok && e != nil {
			return e
		}
	}
	return s.engines["default"]
}

func NewServiceContext(c config.Config, configFilePath string) (*ServiceContext, error) {
	fetch, err := featurestore.NewFetcher(c.FeatureRedis.AsFeaturestore())
	if err != nil {
		return nil, fmt.Errorf("feature redis: %w", err)
	}

	var rankExp *config.RankExpConf
	if c.RankExpConfPath != "" {
		p := c.RankExpConfPath
		if !filepath.IsAbs(p) && configFilePath != "" {
			p = filepath.Join(filepath.Dir(configFilePath), p)
		}
		re, err := config.LoadRankExpConf(p)
		if err != nil {
			return nil, fmt.Errorf("rank exp conf: %w", err)
		}
		rankExp = re
	}

	engines := make(map[string]*rankengine.Engine)
	def, err := rankengine.NewEngine(c.RankEngine, fetch)
	if err != nil {
		return nil, fmt.Errorf("rankengine default: %w", err)
	}
	engines["default"] = def
	engines[""] = def

	for k, rc := range c.RankModelBundles {
		e, err := rankengine.NewEngine(rc, fetch)
		if err != nil {
			return nil, fmt.Errorf("rankengine bundle %q: %w", k, err)
		}
		engines[k] = e
	}

	for name, rc := range c.RankProfiles {
		if name == "" || name == "default" {
			continue
		}
		if _, ok := engines[name]; ok {
			continue
		}
		e, err := rankengine.NewEngine(rc, fetch)
		if err != nil {
			return nil, fmt.Errorf("rankengine profile %q: %w", name, err)
		}
		engines[name] = e
	}

	return &ServiceContext{Config: c, engines: engines, rankExp: rankExp}, nil
}
