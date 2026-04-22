package service

import (
	"Project_sekai_search/internal/config"
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	musicListCacheTTL = 10 * time.Minute
	b30CacheTTL       = 5 * time.Minute
)

func redisGetJSON[T any](ctx context.Context, key string, dst *T) (bool, error) {
	if config.RDB == nil {
		return false, nil
	}
	raw, err := config.RDB.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return false, nil
		}
		return false, err
	}
	if err := json.Unmarshal(raw, dst); err != nil {
		return false, err
	}
	return true, nil
}

func redisSetJSON(ctx context.Context, key string, value any, ttl time.Duration) error {
	if config.RDB == nil {
		return nil
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return config.RDB.Set(ctx, key, payload, ttl).Err()
}

func redisDel(ctx context.Context, keys ...string) error {
	if config.RDB == nil || len(keys) == 0 {
		return nil
	}
	return config.RDB.Del(ctx, keys...).Err()
}
