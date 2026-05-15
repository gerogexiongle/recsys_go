package config

import (
	"github.com/zeromicro/go-zero/rest"

	"recsys_go/pkg/featurestore"
	"recsys_go/pkg/upstream"
)

// RankEndpoints builds upstream config from yaml (flat fields for go-zero).
func (c Config) RankEndpoints() upstream.EndpointsConfig {
	return upstream.EndpointsConfig{
		BaseURL:     c.RankService.BaseURL,
		Endpoints:   c.RankService.Endpoints,
		LoadBalance: c.RankService.LoadBalance,
	}
}

// Config is the domain-agnostic recommend funnel HTTP service (recall → filter → rank → show).
type Config struct {
	rest.RestConf
	RankService struct {
		BaseURL     string   `json:"BaseURL,optional"`
		Endpoints   []string `json:"Endpoints,optional"`
		LoadBalance string   `json:"LoadBalance,optional"`
		TimeoutMs   int      `json:"TimeoutMs,optional"`
	} `json:"RankService,optional"`
	// FunnelConfigPath JSON (Config_Recall-style). Empty = legacy stub recall. Relative paths resolve from the recommend yaml directory.
	FunnelConfigPath string `json:"FunnelConfigPath,optional"`
	// CenterRecallPath / CenterFilterPath / CenterShowControlPath split configs like C++ online_map_center.
	// When CenterRecallPath is set, it takes precedence over FunnelConfigPath for the recommend pipeline.
	CenterRecallPath        string `json:"CenterRecallPath,optional"`
	CenterFilterPath        string `json:"CenterFilterPath,optional"`
	CenterShowControlPath   string `json:"CenterShowControlPath,optional"`
	// FeatureRedis: same shape as rank FeatureRedis (user/item STRING JSON keys).
	FeatureRedis featurestore.RedisConfig `json:"FeatureRedis,optional"`
}
