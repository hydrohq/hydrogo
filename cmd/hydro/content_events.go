package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/config"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	outboxstore "github.com/ardaozyildirim/hydrogo/internal/outbox/sqlstore"
)

// contentEvents returns the writer that records content.created,
// content.updated and content.deleted in the outbox, in the transaction of
// the record change. It is nil (nothing is written, so the table cannot grow
// unread) unless a configured webhook subscribes to a content event.
func contentEvents(cfg config.Config, store outboxstore.Store) func(ctx context.Context, tx database.Executor, topic, aggregateID string, version int) error {
	if cfg.SafeMode || !subscribesToContent(cfg.Webhooks) {
		return nil
	}
	return func(ctx context.Context, tx database.Executor, topic, aggregateID string, version int) error {
		id := fmt.Sprintf("content-%s-%s-%d", strings.TrimPrefix(topic, "content."), aggregateID, version)
		if version == 0 {
			// A deleted record has no version; ids never repeat, but keep
			// events of a re-created id apart.
			random := make([]byte, 6)
			_, _ = rand.Read(random)
			id += hex.EncodeToString(random)
		}
		return store.Enqueue(ctx, tx, id, topic, aggregateID, 1)
	}
}

func subscribesToContent(webhooks []config.Webhook) bool {
	for _, webhook := range webhooks {
		for _, topic := range webhook.Topics {
			if strings.HasPrefix(strings.TrimSpace(topic), "content.") {
				return true
			}
		}
	}
	return false
}
