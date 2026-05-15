package featurestore

import (
	"fmt"
	"os"

	"recsys_go/pkg/redisdecrypt"
)

// RedisConfig configures profile + strategy Redis STRING keys.
type RedisConfig struct {
	Disabled       bool   `json:"Disabled,optional"`
	Host           string `json:"Host,optional"`
	Port           int    `json:"Port,optional"`
	DB             int    `json:"DB,optional"`
	Crypto         bool   `json:"Crypto,optional"`
	PasswordHex    string `json:"PasswordHex,optional"`
	UserKeyPattern string `json:"UserKeyPattern,optional"` // profile user feat
	ItemKeyPattern string `json:"ItemKeyPattern,optional"` // profile item feat
	// Strategy keys (optional; defaults in keys.go)
	UserExposureKeyPattern    string `json:"UserExposureKeyPattern,optional"`
	ItemFeatureLessKeyPattern string `json:"ItemFeatureLessKeyPattern,optional"`
	ItemLabelKeyPattern       string `json:"ItemLabelKeyPattern,optional"`
}

// NewFetcher builds a Fetcher (profile + StrategyFetcher when Redis enabled).
func NewFetcher(c RedisConfig) (Fetcher, error) {
	if c.Disabled {
		return NoOp, nil
	}
	pwdHex := c.PasswordHex
	if pwdHex == "" {
		pwdHex = os.Getenv("RECSYS_REDIS_PASSWORD_HEX")
	}
	if pwdHex == "" {
		return nil, fmt.Errorf("feature redis enabled but PasswordHex empty and RECSYS_REDIS_PASSWORD_HEX unset")
	}
	plain := pwdHex
	if c.Crypto {
		var err error
		plain, err = redisdecrypt.DecryptPassword(pwdHex, nil)
		if err != nil {
			return nil, fmt.Errorf("redis password decrypt: %w", err)
		}
	}
	host := c.Host
	if host == "" {
		host = "algo-cn-test-redis.miniworldplus.com"
	}
	port := c.Port
	if port <= 0 {
		port = 6379
	}
	kp := DefaultKeyPatterns()
	if c.UserKeyPattern != "" {
		kp.UserFeat = c.UserKeyPattern
	}
	if c.ItemKeyPattern != "" {
		kp.ItemFeat = c.ItemKeyPattern
	}
	sk := DefaultStrategyKeyPatterns()
	if c.UserExposureKeyPattern != "" {
		sk.UserExposure = c.UserExposureKeyPattern
	}
	if c.ItemFeatureLessKeyPattern != "" {
		sk.ItemFeatureLess = c.ItemFeatureLessKeyPattern
	}
	if c.ItemLabelKeyPattern != "" {
		sk.ItemLabel = c.ItemLabelKeyPattern
	}
	return NewRedisJSONFetcher(RedisJSONConfig{
		Addr:     fmt.Sprintf("%s:%d", host, port),
		Password: plain,
		DB:       c.DB,
		Keys:     kp,
		Strategy: sk,
	})
}
