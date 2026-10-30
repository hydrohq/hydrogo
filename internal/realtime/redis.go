//go:build !hydro_no_redis

package realtime

import (
	"context"
	"errors"
	"io"
	"strings"

	"github.com/redis/go-redis/v9"
)

type RedisMessageBus struct {
	client *redis.Client
	prefix string
}

func NewRedisMessageBus(client *redis.Client, prefix string) *RedisMessageBus {
	if prefix != "" && !strings.HasSuffix(prefix, ":") {
		prefix += ":"
	}
	return &RedisMessageBus{
		client: client,
		prefix: prefix,
	}
}

func (b *RedisMessageBus) channel(tenantID, topic string) string {
	return b.prefix + "hydro:realtime:" + tenantID + ":" + topic
}

func (b *RedisMessageBus) Publish(ctx context.Context, tenantID, topic string, payload []byte) error {
	if b.client == nil {
		return errors.New("redis client unavailable")
	}
	channel := b.channel(tenantID, topic)
	return b.client.Publish(ctx, channel, payload).Err()
}

type redisBusCloser struct {
	pubsub *redis.PubSub
	cancel context.CancelFunc
}

func (c *redisBusCloser) Close() error {
	c.cancel()
	return c.pubsub.Close()
}

func (b *RedisMessageBus) Subscribe(ctx context.Context, handler func(tenantID, topic string, payload []byte)) (io.Closer, error) {
	if b.client == nil {
		return nil, errors.New("redis client unavailable")
	}
	pattern := b.prefix + "hydro:realtime:*"
	pubsub := b.client.PSubscribe(ctx, pattern)
	if _, err := pubsub.Receive(ctx); err != nil {
		_ = pubsub.Close()
		return nil, err
	}

	subCtx, cancel := context.WithCancel(context.Background())
	go func() {
		ch := pubsub.Channel()
		for {
			select {
			case <-subCtx.Done():
				return
			case msg, ok := <-ch:
				if !ok {
					return
				}
				parts := strings.TrimPrefix(msg.Channel, b.prefix+"hydro:realtime:")
				tenantID, topic, found := strings.Cut(parts, ":")
				if found {
					handler(tenantID, topic, []byte(msg.Payload))
				}
			}
		}
	}()

	return &redisBusCloser{pubsub: pubsub, cancel: cancel}, nil
}
