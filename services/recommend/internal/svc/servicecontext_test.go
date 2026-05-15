package svc

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/zeromicro/go-zero/core/conf"

	"recsys_go/services/recommend/internal/config"
)

func TestNewServiceContextRankClientNotNil(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller")
	}
	cfgPath := filepath.Join(filepath.Dir(file), "..", "..", "etc", "recommend-api.yaml")

	var c config.Config
	if err := conf.Load(cfgPath, &c); err != nil {
		t.Fatal(err)
	}
	ctx, err := NewServiceContext(c, cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if ctx.Rank == nil {
		t.Fatalf("Rank client nil; RankService=%+v", c.RankService)
	}
}
