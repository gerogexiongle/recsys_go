package config

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/zeromicro/go-zero/core/conf"
)

func TestApplyEnvOverridesKafkaAndRank(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller")
	}
	cfgPath := filepath.Join(filepath.Dir(file), "..", "..", "etc", "recommend-api.yaml")
	var c Config
	if err := conf.Load(cfgPath, &c); err != nil {
		t.Fatal(err)
	}
	if c.KafkaPush.Enabled {
		t.Fatal("yaml default KafkaPush.Enabled should be false")
	}
	t.Setenv("RECSYS_KAFKA_PUSH", "1")
	t.Setenv("RECSYS_KAFKA_TOPIC", "test")
	t.Setenv("RECSYS_RANK_ENDPOINTS", "http://127.0.0.1:18081,http://127.0.0.1:18081")
	ApplyEnvOverrides(&c)
	if !c.KafkaPush.Enabled {
		t.Fatal("expected kafka enabled via env")
	}
	if c.KafkaPush.Topic != "test" {
		t.Fatal(c.KafkaPush.Topic)
	}
	if len(c.RankService.Endpoints) != 2 {
		t.Fatalf("endpoints=%v", c.RankService.Endpoints)
	}
}

func TestKafkaE2EConfigFromMainYaml(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller")
	}
	cfgPath := filepath.Join(filepath.Dir(file), "..", "..", "etc", "recommend-api.yaml")
	var c Config
	if err := conf.Load(cfgPath, &c); err != nil {
		t.Fatal(err)
	}
	t.Setenv("RECSYS_KAFKA_PUSH", "1")
	ApplyEnvOverrides(&c)
	if !c.KafkaPush.Enabled {
		t.Fatalf("KafkaPush.Enabled=false, cfg=%+v", c.KafkaPush)
	}
	if c.KafkaPush.Topic != "test" {
		t.Fatal(c.KafkaPush.Topic)
	}
	if c.KafkaPush.DataType != "cn_ol_item" || c.KafkaPush.APIType != 10001 {
		t.Fatalf("wire defaults: %+v", c.KafkaPush)
	}
}
