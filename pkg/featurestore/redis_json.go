package featurestore

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisJSONConfig struct {
	Addr     string
	Password string
	DB       int
	Keys     KeyPatterns
	Strategy StrategyKeyPatterns
}

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
	if sk.FilterExposure == "" {
		sk = DefaultStrategyKeyPatterns()
	}
	rdb := redis.NewClient(&redis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		DialTimeout:  3 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
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

func (r *RedisJSONFetcher) ItemsJSON(ctx context.Context, itemIDs []int64) (map[int64][]byte, error) {
	return r.mgetByItemIDs(ctx, itemIDs, r.kp.ItemKey)
}

func (r *RedisJSONFetcher) FilterExposureJSON(ctx context.Context) ([]byte, bool, error) {
	return r.getMissing(ctx, r.sk.FilterExposure)
}

func (r *RedisJSONFetcher) FilterFeatureLessJSON(ctx context.Context) ([]byte, bool, error) {
	return r.getMissing(ctx, r.sk.FilterFeatureLess)
}

func (r *RedisJSONFetcher) FilterLabelJSON(ctx context.Context) ([]byte, bool, error) {
	return r.getMissing(ctx, r.sk.FilterLabel)
}

func (r *RedisJSONFetcher) HomogenExchangeJSON(ctx context.Context) ([]byte, bool, error) {
	return r.getMissing(ctx, r.sk.HomogenExchange)
}

func (r *RedisJSONFetcher) RecallLaneJSON(ctx context.Context, lane string) ([]byte, bool, error) {
	return r.getMissing(ctx, r.sk.RecallLaneKey(lane))
}

func (r *RedisJSONFetcher) RecallCFUserJSON(ctx context.Context, uin int64) ([]byte, bool, error) {
	return r.getMissing(ctx, r.sk.RecallCFUserKey(uin))
}

func (r *RedisJSONFetcher) UserTagInterestJSON(ctx context.Context, window string, uin int64) ([]byte, bool, error) {
	return r.getMissing(ctx, r.sk.TagInterestUserKey(window, uin))
}

func (r *RedisJSONFetcher) TagInvertJSON(ctx context.Context, tagID int) ([]byte, bool, error) {
	return r.getMissing(ctx, r.sk.TagInvertKey(tagID))
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
	b, _, err := r.getMissing(ctx, key)
	return b, err
}

func (r *RedisJSONFetcher) getMissing(ctx context.Context, key string) ([]byte, bool, error) {
	s, err := r.rdb.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, true, nil
	}
	if err != nil {
		return nil, false, err
	}
	return s, false, nil
}
