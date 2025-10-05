package realtime

import (
	"errors"
	"fmt"

	"github.com/bytedance/sonic"
)

type EventSide string

const (
	ClientEventSide EventSide = "client"
	SeverEventSide  EventSide = "server"
)

type EventType string

// Server event types
const (
	ErrorEventType EventType = "error"
)

type Event struct {
	EventId string
	Type    EventType
	Param   EventParam
}

func (e *Event) MarshalJSON() ([]byte, error) {
	resp := map[string]any{}
	for k, v := range e.Param.Json() {
		resp[k] = v
	}
	resp["event_id"] = e.EventId
	resp["type"] = e.Type
	return sonic.Marshal(resp)
}

func (e *Event) UnmarshalJSON(data []byte) error {
	return e.UnmarshalServerJSON(data)
}

func (e *Event) UnmarshalServerJSON(data []byte) error {
	var raw map[string]any
	if err := sonic.Unmarshal(data, &raw); err != nil {
		return err
	}
	if v, ok := raw["event_id"].(string); ok {
		e.EventId = v
		delete(raw, "event_id")
	} else {
		return errors.New("missing event_id")
	}
	if v, ok := raw["type"].(string); ok {
		e.Type = EventType(v)
		delete(raw, "type")
	} else {
		return errors.New("missing type")
	}
	switch e.Type {
	case ErrorEventType:
		e.Param = new(ErrorEventParam)
	default:
		return fmt.Errorf("unknown event type: %s", e.Type)
	}
	return e.Param.New(raw)
}

type EventParam interface {
	New(map[string]any) error
	Json() map[string]any
}

type ErrorEventParam struct {
	Type    string
	EventId string
	Code    string
	Message string
	Param   any
}

func (p *ErrorEventParam) New(json map[string]any) error {
	if _type, ok := json["type"].(string); ok {
		p.Type = _type
	} else {
		return errors.New("missing type")
	}

	if eventId, ok := json["event_id"].(string); ok {
		p.EventId = eventId
	} else {
		return errors.New("missing event_id")
	}

	if code, ok := json["code"].(string); ok {
		p.Code = code
	} else {
		return errors.New("missing code")
	}

	if message, ok := json["message"].(string); ok {
		p.Message = message
	} else {
		return errors.New("missing message")
	}

	if param, ok := json["param"]; ok {
		p.Param = param
	} else {
		p.Param = nil
	}

	return nil
}

func (p *ErrorEventParam) Json() map[string]any {
	return map[string]any{
		"type":     p.Type,
		"event_id": p.EventId,
		"code":     p.Code,
		"message":  p.Message,
		"param":    p.Param,
	}
}
