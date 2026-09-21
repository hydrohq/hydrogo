// Package eventcatalog defines the stable event names that may cross the
// HydroGo Core boundary through the durable outbox and signed webhook layer.
package eventcatalog

import (
	"encoding/json"
	"errors"
	"fmt"
)

var (
	ErrUnknownEvent    = errors.New("unknown event topic or payload version")
	ErrBreakingCatalog = errors.New("breaking event catalog change")
)

type Definition struct {
	Topic          string `json:"topic"`
	PayloadVersion int    `json:"payload_version"`
	PII            bool   `json:"pii"`
}

// Document is the portable, versioned catalog artifact consumed by webhook and
// Client Edition tooling. It has no runtime delivery authority.
type Document struct {
	Version string       `json:"version"`
	Events  []Definition `json:"events"`
}

var definitions = []Definition{
	{Topic: "page.created", PayloadVersion: 1},
	{Topic: "page.updated", PayloadVersion: 1},
	{Topic: "page.published", PayloadVersion: 1},
	{Topic: "news.created", PayloadVersion: 1},
	{Topic: "news.updated", PayloadVersion: 1},
	{Topic: "news.scheduled", PayloadVersion: 1},
	{Topic: "news.published", PayloadVersion: 1},
	{Topic: "news.retracted", PayloadVersion: 1},
	{Topic: "order.created", PayloadVersion: 1},
	{Topic: "tenant.provisioned", PayloadVersion: 1},
	{Topic: "user.invited", PayloadVersion: 1, PII: true},
}

func Definitions() []Definition { return append([]Definition(nil), definitions...) }

// ExportJSON emits the catalog deterministically for operator publication and
// compatibility review. It does not expose event payloads or subscriptions.
func ExportJSON() ([]byte, error) {
	return json.Marshal(Document{Version: "v1", Events: Definitions()})
}

func Validate(topic string, version int) error {
	for _, definition := range definitions {
		if definition.Topic == topic && definition.PayloadVersion == version {
			return nil
		}
	}
	return ErrUnknownEvent
}

// CompareBaseline rejects removals and PII classification changes for already
// published topic/version pairs. New entries are additive. Payload shape is
// intentionally not inferred here: producers must version it rather than
// mutate an existing published version in place.
func CompareBaseline(baseline, current []Definition) error {
	currentIndex := make(map[string]Definition, len(current))
	for _, definition := range current {
		if definition.Topic == "" || definition.PayloadVersion < 1 {
			return fmt.Errorf("%w: invalid current event definition", ErrBreakingCatalog)
		}
		key := definition.Topic + "@" + fmt.Sprint(definition.PayloadVersion)
		if _, exists := currentIndex[key]; exists {
			return fmt.Errorf("%w: duplicate current event %s", ErrBreakingCatalog, key)
		}
		currentIndex[key] = definition
	}
	for _, definition := range baseline {
		key := definition.Topic + "@" + fmt.Sprint(definition.PayloadVersion)
		got, exists := currentIndex[key]
		if !exists || got.PII != definition.PII {
			return fmt.Errorf("%w: changed or removed event %s", ErrBreakingCatalog, key)
		}
	}
	return nil
}
