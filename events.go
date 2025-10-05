package realtime

import (
	"github.com/bt-bridge/openai-realtime/shared"
)

type EventType string

const (
	SessionUpdate EventType = "session.update"
)

var (
	KnownEventTypes = shared.NewPtrSet(
		SessionUpdate,
	)
)

type Event struct {
	Type    EventType
	Content map[string]any
}
