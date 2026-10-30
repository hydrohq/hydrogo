package realtime

import (
	"context"
	"io"
)

// MessageBus transports realtime events across server instances.
type MessageBus interface {
	Publish(ctx context.Context, tenantID, topic string, payload []byte) error
	Subscribe(ctx context.Context, handler func(tenantID, topic string, payload []byte)) (io.Closer, error)
}
