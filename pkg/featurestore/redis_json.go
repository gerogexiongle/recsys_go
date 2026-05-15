package featurestore

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// RedisJSONConfig configures STRING keys for profile + strategy namespaces.
type RedisJSONConfig struct {
	Addr     string
	Password string
	DB       int
	Keys     KeyPatterns
	Strategy StrategyKeyPatterns
}

// RedisJSONFetcher reads profile and strategy keys via GET / MGET.
type RedisJSONFetcher struct {
	rdb *redis.Client
	kp  KeyPatterns
	sk  StrategyKeyPatterns
}

func NewRedisJSONFetcher(cfg RedisJSONConfig) (*RedisJSONFetcher, error) {
	if cfg.Addr == "" {
		return nil, fmt.Errorf("redis addr empty")
	}
	kp := cfg.Keys
	if kp.UserFeat == "" {
		kp = DefaultKeyPatterns()
	}
	sk := cfg.Strategy
	if sk.UserExposure == "" {
		sk = DefaultStrategyKeyPatterns()
	}
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return nil, err
	}
	return &RedisJSONFetcher{rdb: rdb, kp: kp, sk: sk}, nil
}

func (r *RedisJSONFetcher) UserJSON(ctx context.Context, uin int64) ([]byte, error) {
	return r.get(ctx, r.kp.UserKey(uin))
}

func (r *RedisJSONFetcher) ItemJSON(ctx context.Context, itemID int64) ([]byte, error) {
	return r.get(ctx, r.kp.ItemKey(itemID))
}

func (r *RedisJSONFetcher) UserExposureJSON(ctx context.Context, uin int64) ([]byte, bool, error) {
	b, err := r.rdb.Get(ctx, r.sk.UserExposureKey(uin)).Bytes()
	if err == redis.Nil {
		return nil, true, nil
	}
	if err != nil {
		return nil, false, err
	}
	return b, false, nil
}

func (r *RedisJSONFetcher) ItemsJSON(ctx context.Context, itemIDs []int64) (map[int64][]byte, error) {
	return r.mgetByItemIDs(ctx, itemIDs, r.kp.ItemKey)
}

func (r *RedisJSONFetcher) ItemsFeatureLessJSON(ctx context.Context, itemIDs []int64) (map[int64][]byte, error) {
	return r.mgetByItemIDs(ctx, itemIDs, r.sk.ItemFeatureLessKey)
}

func (r *RedisJSONFetcher) ItemsLabelJSON(ctx context.Context, itemIDs []int64) (map[int64][]byte, error) {
	return r.mgetByItemIDs(ctx, itemIDs, r.sk.ItemLabelKey)
}

func (r *RedisJSONFetcher) mgetByItemIDs(ctx context.Context, itemIDs []int64, keyFn func(int64) string) (map[int64][]byte, error) {
	out := make(map[int64][]byte, len(itemIDs))
	if len(itemIDs) == 0 {
		return out, nil
	}
	keys := make([]string, len(itemIDs))
	for i, id := range itemIDs {
		keys[i] = keyFn(id)
	}
	vals, err := r.rdb.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, err
	}
	for i, v := range vals {
		if v == nil {
			continue
		}
		switch s := v.(type) {
		case string:
			out[itemIDs[i]] = []byte(s)
		case []byte:
			out[itemIDs[i]] = s
		}
	}
	return out, nil
}

func (r *RedisJSONFetcher) get(ctx context.Context, key string) ([]byte, error) {
	s, err := r.rdb.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	return s, err
}
