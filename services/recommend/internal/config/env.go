package config

import (
	"os"
	"strconv"
	"strings"
)

// ApplyEnvOverrides patches config after yaml load (E2E / ops without duplicating yaml files).
//
//	RECSYS_REDIS_HOST, RECSYS_REDIS_PORT, RECSYS_REDIS_PASSWORD_HEX, RECSYS_REDIS_DISABLED=1
//	RECSYS_KAFKA_PUSH=1, RECSYS_KAFKA_BROKERS (comma), RECSYS_KAFKA_TOPIC
//	RECSYS_RANK_ENDPOINTS (comma) — when set, replaces RankService.Endpoints (lab LB / multi-pod)
func ApplyEnvOverrides(c *Config) {
	if v := strings.TrimSpace(os.Getenv("RECSYS_REDIS_HOST")); v != "" {
		c.FeatureRedis.Host = v
	}
	if v := strings.TrimSpace(os.Getenv("RECSYS_REDIS_PORT")); v != "" {
		if port, err := strconv.Atoi(v); err == nil && port > 0 {
			c.FeatureRedis.Port = port
		}
	}
	if v := strings.TrimSpace(os.Getenv("RECSYS_REDIS_PASSWORD_HEX")); v != "" {
		c.FeatureRedis.PasswordHex = v
	}
	if os.Getenv("RECSYS_REDIS_DISABLED") == "1" {
		c.FeatureRedis.Disabled = true
	}

	if os.Getenv("RECSYS_KAFKA_PUSH") == "1" {
		c.KafkaPush.Enabled = true
	}
	if v := strings.TrimSpace(os.Getenv("RECSYS_KAFKA_BROKERS")); v != "" {
		c.KafkaPush.Brokers = splitComma(v)
	}
	if v := strings.TrimSpace(os.Getenv("RECSYS_KAFKA_TOPIC")); v != "" {
		c.KafkaPush.Topic = v
	}

	if v := strings.TrimSpace(os.Getenv("RECSYS_RANK_ENDPOINTS")); v != "" {
		c.RankService.Endpoints = splitComma(v)
	}
}

func splitComma(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
