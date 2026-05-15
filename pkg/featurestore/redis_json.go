package featurestore

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// RedisJSONConfig configures STRING keys holding whole JSON per user / item.
type RedisJSONConfig struct {
	Addr           string
	Password       string
	DB             int
	UserKeyPattern string
	ItemKeyPattern string
}

// RedisJSONFetcher reads GET / MGET keys built from uin / item id.
type RedisJSONFetcher struct {
	rdb *redis.Client
	kp  KeyPatterns
}

func NewRedisJSONFetcher(cfg RedisJSONConfig) (*RedisJSONFetcher, error) {
	if cfg.Addr == "" {
		return nil, fmt.Errorf("redis addr empty")
	}
	kp := DefaultKeyPatterns()
	if cfg.UserKeyPattern != "" {
		kp.User = cfg.UserKeyPattern
	}
	if cfg.ItemKeyPattern != "" {
		kp.Item = cfg.ItemKeyPattern
	}
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return nil, err
	}
	return &RedisJSONFetcher{rdb: rdb, kp: kp}, nil
}

func (r *RedisJSONFetcher) UserJSON(ctx context.Context, uin int64) ([]byte, error) {
	key := r.kp.UserKey(uin)
	s, err := r.rdb.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	return s, err
}

func (r *RedisJSONFetcher) ItemJSON(ctx context.Context, itemID int64) ([]byte, error) {
	key := r.kp.ItemKey(itemID)
	s, err := r.rdb.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	return s, err
}

// ItemsJSON MGETs item keys in one round trip (center filter / rank batch).
func (r *RedisJSONFetcher) ItemsJSON(ctx context.Context, itemIDs []int64) (map[int64][]byte, error) {
	out := make(map[int64][]byte, len(itemIDs))
	if len(itemIDs) == 0 {
		return out, nil
	}
	keys := make([]string, len(itemIDs))
	for i, id := range itemIDs {
		keys[i] = r.kp.ItemKey(id)
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
