//go:build !hydro_no_redis

package cache

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisJSONRemote stores only reproducible cache values. It is intentionally
// generic so module services do not depend on Redis or serialization details.
type RedisJSONRemote[T any] struct {
	Client *redis.Client
	Prefix string
	TTL    time.Duration
}

func (r RedisJSONRemote[T]) Get(key string) (T, bool, error) {
	var zero T
	if r.Client == nil {
		return zero, false, errors.New("redis cache client is required")
	}
	value, err := r.Client.Get(context.Background(), r.Prefix+key).Bytes()
	if errors.Is(err, redis.Nil) {
		return zero, false, nil
	}
	if err != nil {
		return zero, false, err
	}
	var decoded T
	if err := json.Unmarshal(value, &decoded); err != nil {
		return zero, false, err
	}
	return decoded, true, nil
}

func (r RedisJSONRemote[T]) Set(key string, value T) error {
	if r.Client == nil || r.TTL <= 0 {
		return errors.New("redis cache client and TTL are required")
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return r.Client.Set(context.Background(), r.Prefix+key, payload, r.TTL).Err()
}
