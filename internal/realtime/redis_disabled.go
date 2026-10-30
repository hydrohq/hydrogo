//go:build hydro_no_redis

package realtime

import (
	"context"
	"errors"
	"io"
)

type RedisMessageBus struct{}

func (b *RedisMessageBus) Publish(ctx context.Context, tenantID, topic string, payload []byte) error {
	return errors.New("redis is not included in this build")
}

func (b *RedisMessageBus) Subscribe(ctx context.Context, handler func(tenantID, topic string, payload []byte)) (io.Closer, error) {
	return nil, errors.New("redis is not included in this build")
}
