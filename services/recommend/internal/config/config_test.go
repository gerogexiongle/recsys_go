package config

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/zeromicro/go-zero/core/conf"
)

func TestLoadRankServiceBaseURL(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller")
	}
	cfgPath := filepath.Join(filepath.Dir(file), "..", "..", "etc", "recommend-api.yaml")

	var c Config
	if err := conf.Load(cfgPath, &c); err != nil {
		t.Fatal(err)
	}
	if c.RankService.BaseURL == "" {
		t.Fatalf("RankService.BaseURL empty after load from %s", cfgPath)
	}
	if c.RankService.BaseURL != "http://127.0.0.1:18081" {
		t.Fatalf("unexpected BaseURL %q", c.RankService.BaseURL)
	}
}
