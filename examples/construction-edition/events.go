package main

import "errors"

var ErrUnknownConstructionEvent = errors.New("unknown construction event topic or payload version")

// ConstructionEvent is owned by the separately versioned Client Edition. It
// must not be copied into HydroGo Core's generic event catalog.
type ConstructionEvent struct {
	Topic          string
	PayloadVersion int
}

var constructionEvents = []ConstructionEvent{
	{Topic: "construction.document.approved", PayloadVersion: 1},
	{Topic: "construction.daily-report.approved", PayloadVersion: 1},
	{Topic: "construction.issue.created", PayloadVersion: 1},
}

func ValidateConstructionEvent(topic string, payloadVersion int) error {
	for _, event := range constructionEvents {
		if event.Topic == topic && event.PayloadVersion == payloadVersion {
			return nil
		}
	}
	return ErrUnknownConstructionEvent
}
