// Package eventcatalog defines the stable event names that may cross the
// HydroGo Core boundary through the durable outbox and signed webhook layer.
package eventcatalog

import "errors"

var ErrUnknownEvent = errors.New("unknown event topic or payload version")

type Definition struct {
	Topic          string
	PayloadVersion int
	PII            bool
}

var definitions = []Definition{
	{Topic: "page.created", PayloadVersion: 1},
	{Topic: "page.updated", PayloadVersion: 1},
	{Topic: "page.published", PayloadVersion: 1},
	{Topic: "news.created", PayloadVersion: 1},
	{Topic: "news.updated", PayloadVersion: 1},
	{Topic: "news.published", PayloadVersion: 1},
	{Topic: "news.withdrawn", PayloadVersion: 1},
	{Topic: "tenant.provisioned", PayloadVersion: 1},
	{Topic: "user.invited", PayloadVersion: 1, PII: true},
	{Topic: "construction.document.approved", PayloadVersion: 1},
	{Topic: "construction.daily-report.approved", PayloadVersion: 1},
	{Topic: "construction.issue.created", PayloadVersion: 1},
}

func Definitions() []Definition { return append([]Definition(nil), definitions...) }

func Validate(topic string, version int) error {
	for _, definition := range definitions {
		if definition.Topic == topic && definition.PayloadVersion == version {
			return nil
		}
	}
	return ErrUnknownEvent
}
