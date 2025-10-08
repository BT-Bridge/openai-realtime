package realtime

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/goccy/go-yaml"
)

type EventType string

type ServerEventType EventType

type ClientEventType EventType

// Server event types
const (
	ServerEventTypeError                                            ServerEventType = "error"
	ServerEventTypeSessionCreated                                   ServerEventType = "session.created"
	ServerEventTypeSessionUpdated                                   ServerEventType = "session.updated"
	ServerEventTypeConversationItemAdded                            ServerEventType = "conversation.item.added"
	ServerEventTypeConversationItemDone                             ServerEventType = "conversation.item.done"
	ServerEventTypeConversationItemRetrieved                        ServerEventType = "conversation.item.retrieved"
	ServerEventTypeConversationItemInputAudioTranscriptionCompleted ServerEventType = "conversation.item.input_audio_transcription.completed"
	ServerEventTypeConversationItemInputAudioTranscriptionDelta     ServerEventType = "conversation.item.input_audio_transcription.delta"
	ServerEventTypeConversationItemInputAudioTranscriptionSegment   ServerEventType = "conversation.item.input_audio_transcription.segment"
	ServerEventTypeConversationItemInputAudioTranscriptionFailed    ServerEventType = "conversation.item.input_audio_transcription.failed"
	ServerEventTypeConversationItemTruncated                        ServerEventType = "conversation.item.truncated"
	ServerEventTypeConversationItemDeleted                          ServerEventType = "conversation.item.deleted"
	ServerEventTypeInputAudioBufferCommitted                        ServerEventType = "input_audio_buffer.committed"
	ServerEventTypeInputAudioBufferCleared                          ServerEventType = "input_audio_buffer.cleared"
	ServerEventTypeInputAudioBufferSpeechStarted                    ServerEventType = "input_audio_buffer.speech_started"
	ServerEventTypeInputAudioBufferSpeechStopped                    ServerEventType = "input_audio_buffer.speech_stopped"
	ServerEventTypeInputAudioBufferTimeoutTriggered                 ServerEventType = "input_audio_buffer.timeout_triggered"
	ServerEventTypeOutputAudioBufferStarted                         ServerEventType = "output_audio_buffer.started"
	ServerEventTypeOutputAudioBufferStopped                         ServerEventType = "output_audio_buffer.stopped"
	ServerEventTypeOutputAudioBufferCleared                         ServerEventType = "output_audio_buffer.cleared"
	ServerEventTypeResponseCreated                                  ServerEventType = "response.created"
	ServerEventTypeResponseDone                                     ServerEventType = "response.done"
	ServerEventTypeResponseOutputItemAdded                          ServerEventType = "response.output_item.added"
	ServerEventTypeResponseOutputItemDone                           ServerEventType = "response.output_item.done"
	ServerEventTypeResponseContentPartAdded                         ServerEventType = "response.content_part.added"
	ServerEventTypeResponseContentPartDone                          ServerEventType = "response.content_part.done"
	ServerEventTypeResponseOutputTextDelta                          ServerEventType = "response.output_text.delta"
	ServerEventTypeResponseOutputTextDone                           ServerEventType = "response.output_text.done"
	ServerEventTypeResponseOutputAudioTranscriptDelta               ServerEventType = "response.output_audio_transcript.delta"
	ServerEventTypeResponseOutputAudioTranscriptDone                ServerEventType = "response.output_audio_transcript.done"
	ServerEventTypeResponseOutputAudioDelta                         ServerEventType = "response.output_audio.delta"
	ServerEventTypeResponseOutputAudioDone                          ServerEventType = "response.output_audio.done"
	ServerEventTypeResponseFunctionCallArgumentsDelta               ServerEventType = "response.function_call_arguments.delta"
	ServerEventTypeResponseFunctionCallArgumentsDone                ServerEventType = "response.function_call_arguments.done"
	ServerEventTypeResponseMCPCallArgumentsDelta                    ServerEventType = "response.mcp_call_arguments.delta"
	ServerEventTypeResponseMCPCallArgumentsDone                     ServerEventType = "response.mcp_call_arguments.done"
	ServerEventTypeResponseMCPCallInProgress                        ServerEventType = "response.mcp_call.in_progress"
	ServerEventTypeResponseMCPCallCompleted                         ServerEventType = "response.mcp_call.completed"
	ServerEventTypeResponseMCPCallFailed                            ServerEventType = "response.mcp_call.failed"
	ServerEventTypeMCPListToolsInProgress                           ServerEventType = "mcp_list_tools.in_progress"
	ServerEventTypeMCPListToolsCompleted                            ServerEventType = "mcp_list_tools.completed"
	ServerEventTypeMCPListToolsFailed                               ServerEventType = "mcp_list_tools.failed"
	ServerEventTypeRatelimitsUpdated                                ServerEventType = "rate_limits.updated"
)

// Client event types
const (
	ClientEventTypeSessionUpdate            ClientEventType = "session.update"
	ClientEventTypeInputAudioBufferAppend   ClientEventType = "input_audio_buffer.append"
	ClientEventTypeInputAudioBufferCommit   ClientEventType = "input_audio_buffer.commit"
	ClientEventTypeInputAudioBufferClear    ClientEventType = "input_audio_buffer.clear"
	ClientEventTypeConversationItemCreate   ClientEventType = "conversation.item.create"
	ClientEventTypeConversationItemRetrieve ClientEventType = "conversation.item.retrieve"
	ClientEventTypeConversationItemTruncate ClientEventType = "conversation.item.truncate"
	ClientEventTypeConversationItemDelete   ClientEventType = "conversation.item.delete"
	ClientEventTypeResponseCreate           ClientEventType = "response.create"
	ClientEventTypeResponseCancel           ClientEventType = "response.cancel"
	ClientEventTypeOutputAudioBufferClear   ClientEventType = "output_audio_buffer.clear"
)

type Event interface {
	EventType() EventType
	IsServerEvent() bool
	IsClientEvent() bool
	MarshalYAML() ([]byte, error)
	UnmarshalYAML(data []byte) error
	MarshalJSON() ([]byte, error)
	UnmarshalJSON(data []byte) error
}

type ServerEvent struct {
	EventId string
	Type    ServerEventType
	Param   EventParam
}

var _ Event = (*ServerEvent)(nil)

func (e *ServerEvent) EventType() EventType {
	return EventType(e.Type)
}

func (e *ServerEvent) IsServerEvent() bool {
	return true
}

func (e *ServerEvent) IsClientEvent() bool {
	return false
}

func (e *ServerEvent) MarshalYAML() ([]byte, error) {
	if e.EventId == "" {
		return nil, errors.New("EventId is empty")
	}
	if e.Type == "" {
		return nil, errors.New("Type is empty")
	}
	if e.Param == nil {
		return nil, errors.New("Param is nil")
	}
	resp := map[string]any{}
	for k, v := range e.Param.Json() {
		resp[k] = v
	}
	resp["event_id"] = e.EventId
	resp["type"] = e.Type
	return yaml.MarshalWithOptions(resp, yaml.UseJSONMarshaler())
}

func (e *ServerEvent) UnmarshalYAML(data []byte) error {
	var raw map[string]any
	if err := yaml.UnmarshalWithOptions(data, &raw, yaml.UseJSONUnmarshaler()); err != nil {
		return err
	}
	if v, ok := raw["event_id"].(string); ok {
		e.EventId = v
		delete(raw, "event_id")
	} else {
		return errors.New("missing event_id")
	}
	if v, ok := raw["type"].(string); ok {
		e.Type = ServerEventType(v)
		delete(raw, "type")
	} else {
		return errors.New("missing type")
	}
	if len(raw) == 0 {
		return errors.New("missing param")
	}
	switch e.Type {
	case ServerEventTypeError:
		e.Param = new(ServerEventParamError)
	case ServerEventTypeSessionCreated:
		e.Param = new(ServerEventParamSessionCreated)
	case ServerEventTypeSessionUpdated:
		e.Param = new(ServerEventParamSessionUpdated)
	case ServerEventTypeConversationItemAdded:
		e.Param = new(ServerEventParamConversationItemAdded)
	case ServerEventTypeConversationItemDone:
		e.Param = new(ServerEventParamConversationItemDone)
	case ServerEventTypeConversationItemRetrieved:
		e.Param = new(ServerEventParamConversationItemRetrieved)
	case ServerEventTypeConversationItemInputAudioTranscriptionCompleted:
		e.Param = new(ServerEventParamConversationItemInputAudioTranscriptionCompleted)
	case ServerEventTypeConversationItemInputAudioTranscriptionDelta:
		e.Param = new(ServerEventParamConversationItemInputAudioTranscriptionDelta)
	case ServerEventTypeConversationItemInputAudioTranscriptionSegment:
		e.Param = new(ServerEventParamConversationItemInputAudioTranscriptionSegment)
	case ServerEventTypeConversationItemInputAudioTranscriptionFailed:
		e.Param = new(ServerEventParamConversationItemInputAudioTranscriptionFailed)
	case ServerEventTypeConversationItemTruncated:
		e.Param = new(ServerEventParamConversationItemTruncated)
	case ServerEventTypeConversationItemDeleted:
		e.Param = new(ServerEventParamConversationItemDeleted)
	case ServerEventTypeInputAudioBufferCommitted:
		e.Param = new(ServerEventParamInputAudioBufferCommitted)
	case ServerEventTypeInputAudioBufferCleared:
		e.Param = new(ServerEventParamInputAudioBufferCleared)
	case ServerEventTypeInputAudioBufferSpeechStarted:
		e.Param = new(ServerEventParamInputAudioBufferSpeechStarted)
	case ServerEventTypeInputAudioBufferSpeechStopped:
		e.Param = new(ServerEventParamInputAudioBufferSpeechStopped)
	case ServerEventTypeInputAudioBufferTimeoutTriggered:
		e.Param = new(ServerEventParamInputAudioBufferTimeoutTriggered)
	case ServerEventTypeOutputAudioBufferStarted:
		e.Param = new(ServerEventParamOutputAudioBufferStarted)
	case ServerEventTypeOutputAudioBufferStopped:
		e.Param = new(ServerEventParamOutputAudioBufferStopped)
	case ServerEventTypeOutputAudioBufferCleared:
		e.Param = new(ServerEventParamOutputAudioBufferCleared)
	case ServerEventTypeResponseCreated:
		e.Param = new(ServerEventParamResponseCreated)
	case ServerEventTypeResponseDone:
		e.Param = new(ServerEventParamResponseDone)
	case ServerEventTypeResponseOutputItemAdded:
		e.Param = new(ServerEventParamResponseOutputItemAdded)
	case ServerEventTypeResponseOutputItemDone:
		e.Param = new(ServerEventParamResponseOutputItemDone)
	case ServerEventTypeResponseContentPartAdded:
		e.Param = new(ServerEventParamResponseContentPartAdded)
	case ServerEventTypeResponseContentPartDone:
		e.Param = new(ServerEventParamResponseContentPartDone)
	case ServerEventTypeResponseOutputTextDelta:
		e.Param = new(ServerEventParamResponseOutputTextDelta)
	case ServerEventTypeResponseOutputTextDone:
		e.Param = new(ServerEventParamResponseOutputTextDone)
	case ServerEventTypeResponseOutputAudioTranscriptDelta:
		e.Param = new(ServerEventParamResponseOutputAudioTranscriptDelta)
	case ServerEventTypeResponseOutputAudioTranscriptDone:
		e.Param = new(ServerEventParamResponseOutputAudioTranscriptDone)
	case ServerEventTypeResponseOutputAudioDelta:
		e.Param = new(ServerEventParamResponseOutputAudioDelta)
	case ServerEventTypeResponseOutputAudioDone:
		e.Param = new(ServerEventParamResponseOutputAudioDone)
	case ServerEventTypeResponseFunctionCallArgumentsDelta:
		e.Param = new(ServerEventParamResponseFunctionCallArgumentsDelta)
	case ServerEventTypeResponseFunctionCallArgumentsDone:
		e.Param = new(ServerEventParamResponseFunctionCallArgumentsDone)
	case ServerEventTypeResponseMCPCallArgumentsDelta:
		e.Param = new(ServerEventParamResponseMCPCallArgumentsDelta)
	case ServerEventTypeResponseMCPCallArgumentsDone:
		e.Param = new(ServerEventParamResponseMCPCallArgumentsDone)
	case ServerEventTypeResponseMCPCallInProgress:
		e.Param = new(ServerEventParamResponseMCPCallInProgress)
	case ServerEventTypeResponseMCPCallCompleted:
		e.Param = new(ServerEventParamResponseMCPCallCompleted)
	case ServerEventTypeResponseMCPCallFailed:
		e.Param = new(ServerEventParamResponseMCPCallFailed)
	case ServerEventTypeMCPListToolsInProgress:
		e.Param = new(ServerEventParamMCPListToolsInProgress)
	case ServerEventTypeMCPListToolsCompleted:
		e.Param = new(ServerEventParamMCPListToolsCompleted)
	case ServerEventTypeMCPListToolsFailed:
		e.Param = new(ServerEventParamMCPListToolsFailed)
	case ServerEventTypeRatelimitsUpdated:
		e.Param = new(ServerEventParamRatelimitsUpdated)
	default:
		return fmt.Errorf("unknown event type: %s", e.Type)
	}
	return e.Param.New(raw)
}

func (e *ServerEvent) MarshalJSON() ([]byte, error) {
	if e.EventId == "" {
		return nil, errors.New("EventId is empty")
	}
	if e.Type == "" {
		return nil, errors.New("Type is empty")
	}
	if e.Param == nil {
		return nil, errors.New("Param is nil")
	}
	resp := map[string]any{}
	for k, v := range e.Param.Json() {
		resp[k] = v
	}
	resp["event_id"] = e.EventId
	resp["type"] = e.Type
	return sonic.Marshal(resp)
}

func (e *ServerEvent) UnmarshalJSON(data []byte) error {
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
		e.Type = ServerEventType(v)
		delete(raw, "type")
	} else {
		return errors.New("missing type")
	}
	if len(raw) == 0 {
		return errors.New("missing param")
	}
	switch e.Type {
	case ServerEventTypeError:
		e.Param = new(ServerEventParamError)
	case ServerEventTypeSessionCreated:
		e.Param = new(ServerEventParamSessionCreated)
	case ServerEventTypeSessionUpdated:
		e.Param = new(ServerEventParamSessionUpdated)
	case ServerEventTypeConversationItemAdded:
		e.Param = new(ServerEventParamConversationItemAdded)
	case ServerEventTypeConversationItemDone:
		e.Param = new(ServerEventParamConversationItemDone)
	case ServerEventTypeConversationItemRetrieved:
		e.Param = new(ServerEventParamConversationItemRetrieved)
	case ServerEventTypeConversationItemInputAudioTranscriptionCompleted:
		e.Param = new(ServerEventParamConversationItemInputAudioTranscriptionCompleted)
	case ServerEventTypeConversationItemInputAudioTranscriptionDelta:
		e.Param = new(ServerEventParamConversationItemInputAudioTranscriptionDelta)
	case ServerEventTypeConversationItemInputAudioTranscriptionSegment:
		e.Param = new(ServerEventParamConversationItemInputAudioTranscriptionSegment)
	case ServerEventTypeConversationItemInputAudioTranscriptionFailed:
		e.Param = new(ServerEventParamConversationItemInputAudioTranscriptionFailed)
	case ServerEventTypeConversationItemTruncated:
		e.Param = new(ServerEventParamConversationItemTruncated)
	case ServerEventTypeConversationItemDeleted:
		e.Param = new(ServerEventParamConversationItemDeleted)
	case ServerEventTypeInputAudioBufferCommitted:
		e.Param = new(ServerEventParamInputAudioBufferCommitted)
	case ServerEventTypeInputAudioBufferCleared:
		e.Param = new(ServerEventParamInputAudioBufferCleared)
	case ServerEventTypeInputAudioBufferSpeechStarted:
		e.Param = new(ServerEventParamInputAudioBufferSpeechStarted)
	case ServerEventTypeInputAudioBufferSpeechStopped:
		e.Param = new(ServerEventParamInputAudioBufferSpeechStopped)
	case ServerEventTypeInputAudioBufferTimeoutTriggered:
		e.Param = new(ServerEventParamInputAudioBufferTimeoutTriggered)
	case ServerEventTypeOutputAudioBufferStarted:
		e.Param = new(ServerEventParamOutputAudioBufferStarted)
	case ServerEventTypeOutputAudioBufferStopped:
		e.Param = new(ServerEventParamOutputAudioBufferStopped)
	case ServerEventTypeOutputAudioBufferCleared:
		e.Param = new(ServerEventParamOutputAudioBufferCleared)
	case ServerEventTypeResponseCreated:
		e.Param = new(ServerEventParamResponseCreated)
	case ServerEventTypeResponseDone:
		e.Param = new(ServerEventParamResponseDone)
	case ServerEventTypeResponseOutputItemAdded:
		e.Param = new(ServerEventParamResponseOutputItemAdded)
	case ServerEventTypeResponseOutputItemDone:
		e.Param = new(ServerEventParamResponseOutputItemDone)
	case ServerEventTypeResponseContentPartAdded:
		e.Param = new(ServerEventParamResponseContentPartAdded)
	case ServerEventTypeResponseContentPartDone:
		e.Param = new(ServerEventParamResponseContentPartDone)
	case ServerEventTypeResponseOutputTextDelta:
		e.Param = new(ServerEventParamResponseOutputTextDelta)
	case ServerEventTypeResponseOutputTextDone:
		e.Param = new(ServerEventParamResponseOutputTextDone)
	case ServerEventTypeResponseOutputAudioTranscriptDelta:
		e.Param = new(ServerEventParamResponseOutputAudioTranscriptDelta)
	case ServerEventTypeResponseOutputAudioTranscriptDone:
		e.Param = new(ServerEventParamResponseOutputAudioTranscriptDone)
	case ServerEventTypeResponseOutputAudioDelta:
		e.Param = new(ServerEventParamResponseOutputAudioDelta)
	case ServerEventTypeResponseOutputAudioDone:
		e.Param = new(ServerEventParamResponseOutputAudioDone)
	case ServerEventTypeResponseFunctionCallArgumentsDelta:
		e.Param = new(ServerEventParamResponseFunctionCallArgumentsDelta)
	case ServerEventTypeResponseFunctionCallArgumentsDone:
		e.Param = new(ServerEventParamResponseFunctionCallArgumentsDone)
	case ServerEventTypeResponseMCPCallArgumentsDelta:
		e.Param = new(ServerEventParamResponseMCPCallArgumentsDelta)
	case ServerEventTypeResponseMCPCallArgumentsDone:
		e.Param = new(ServerEventParamResponseMCPCallArgumentsDone)
	case ServerEventTypeResponseMCPCallInProgress:
		e.Param = new(ServerEventParamResponseMCPCallInProgress)
	case ServerEventTypeResponseMCPCallCompleted:
		e.Param = new(ServerEventParamResponseMCPCallCompleted)
	case ServerEventTypeResponseMCPCallFailed:
		e.Param = new(ServerEventParamResponseMCPCallFailed)
	case ServerEventTypeMCPListToolsInProgress:
		e.Param = new(ServerEventParamMCPListToolsInProgress)
	case ServerEventTypeMCPListToolsCompleted:
		e.Param = new(ServerEventParamMCPListToolsCompleted)
	case ServerEventTypeMCPListToolsFailed:
		e.Param = new(ServerEventParamMCPListToolsFailed)
	case ServerEventTypeRatelimitsUpdated:
		e.Param = new(ServerEventParamRatelimitsUpdated)
	default:
		return fmt.Errorf("unknown event type: %s", e.Type)
	}
	return e.Param.New(raw)
}

type EventParam interface {
	New(map[string]any) error
	Json() map[string]any
}

// Helpers for number conversions
func asInt(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int8:
		return int(n), true
	case int16:
		return int(n), true
	case int32:
		return int(n), true
	case int64:
		return int(n), true
	case uint:
		return int(n), true
	case uint8:
		return int(n), true
	case uint16:
		return int(n), true
	case uint32:
		return int(n), true
	case uint64:
		return int(n), true
	case float32:
		return int(n), true
	case float64:
		return int(n), true
	case json.Number:
		if i, err := n.Int64(); err == nil {
			return int(i), true
		}
	}
	return 0, false
}

func asFloat64(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int8:
		return float64(n), true
	case int16:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	case uint:
		return float64(n), true
	case uint8:
		return float64(n), true
	case uint16:
		return float64(n), true
	case uint32:
		return float64(n), true
	case uint64:
		return float64(n), true
	case json.Number:
		if f, err := n.Float64(); err == nil {
			return f, true
		}
	}
	return 0, false
}

// ServerEventParamError
type ServerEventParamError struct {
	Type    string
	EventId string
	Code    string
	Message string
	Param   any
}

func (p *ServerEventParamError) New(jsonMap map[string]any) error {
	if errObj, ok := jsonMap["error"].(map[string]any); ok {
		if v, ok := errObj["type"].(string); ok {
			p.Type = v
		} else {
			return errors.New("missing error.type")
		}
		if v, ok := errObj["code"].(string); ok {
			p.Code = v
		} else {
			return errors.New("missing error.code")
		}
		if v, ok := errObj["message"].(string); ok {
			p.Message = v
		} else {
			return errors.New("missing error.message")
		}
		if v, ok := errObj["event_id"].(string); ok {
			p.EventId = v
		} else {
			// Not always present, keep empty if missing
			p.EventId = ""
		}
		if v, ok := errObj["param"]; ok {
			p.Param = v
		} else {
			p.Param = nil
		}
		return nil
	}

	// Fallback: flattened keys (your sample format)
	if _type, ok := jsonMap["type"].(string); ok {
		p.Type = _type
	} else {
		return errors.New("missing type")
	}
	if eventId, ok := jsonMap["event_id"].(string); ok {
		p.EventId = eventId
	} else {
		// not required for flattened?
		p.EventId = ""
	}
	if code, ok := jsonMap["code"].(string); ok {
		p.Code = code
	} else {
		return errors.New("missing code")
	}
	if message, ok := jsonMap["message"].(string); ok {
		p.Message = message
	} else {
		return errors.New("missing message")
	}
	if param, ok := jsonMap["param"]; ok {
		p.Param = param
	} else {
		p.Param = nil
	}
	return nil
}

func (p *ServerEventParamError) Json() map[string]any {
	// Emit official nested shape
	return map[string]any{
		"error": map[string]any{
			"type":     p.Type,
			"event_id": p.EventId,
			"code":     p.Code,
			"message":  p.Message,
			"param":    p.Param,
		},
	}
}

// session.created
type ServerEventParamSessionCreated struct {
	Session map[string]any
}

func (p *ServerEventParamSessionCreated) New(m map[string]any) error {
	if session, ok := m["session"].(map[string]any); ok {
		p.Session = session
	} else {
		return errors.New("missing session")
	}
	return nil
}

func (p *ServerEventParamSessionCreated) Json() map[string]any {
	return map[string]any{
		"session": p.Session,
	}
}

// session.updated
type ServerEventParamSessionUpdated struct {
	Session map[string]any
}

func (p *ServerEventParamSessionUpdated) New(m map[string]any) error {
	if session, ok := m["session"].(map[string]any); ok {
		p.Session = session
	} else {
		return errors.New("missing session")
	}
	return nil
}

func (p *ServerEventParamSessionUpdated) Json() map[string]any {
	return map[string]any{
		"session": p.Session,
	}
}

// conversation.item.added
type ServerEventParamConversationItemAdded struct {
	PreviousItemId any
	Item           map[string]any
}

func (p *ServerEventParamConversationItemAdded) New(m map[string]any) error {
	if v, ok := m["previous_item_id"]; ok {
		p.PreviousItemId = v // can be string or nil
	} else {
		p.PreviousItemId = nil
	}
	if item, ok := m["item"].(map[string]any); ok {
		p.Item = item
	} else {
		return errors.New("missing item")
	}
	return nil
}

func (p *ServerEventParamConversationItemAdded) Json() map[string]any {
	return map[string]any{
		"previous_item_id": p.PreviousItemId,
		"item":             p.Item,
	}
}

// conversation.item.done
type ServerEventParamConversationItemDone struct {
	PreviousItemId any
	Item           map[string]any
}

func (p *ServerEventParamConversationItemDone) New(m map[string]any) error {
	if v, ok := m["previous_item_id"]; ok {
		p.PreviousItemId = v
	} else {
		p.PreviousItemId = nil
	}
	if item, ok := m["item"].(map[string]any); ok {
		p.Item = item
	} else {
		return errors.New("missing item")
	}
	return nil
}

func (p *ServerEventParamConversationItemDone) Json() map[string]any {
	return map[string]any{
		"previous_item_id": p.PreviousItemId,
		"item":             p.Item,
	}
}

// conversation.item.retrieved
type ServerEventParamConversationItemRetrieved struct {
	Item map[string]any
}

func (p *ServerEventParamConversationItemRetrieved) New(m map[string]any) error {
	if item, ok := m["item"].(map[string]any); ok {
		p.Item = item
	} else {
		return errors.New("missing item")
	}
	return nil
}

func (p *ServerEventParamConversationItemRetrieved) Json() map[string]any {
	return map[string]any{
		"item": p.Item,
	}
}

// conversation.item.input_audio_transcription.completed
type ServerEventParamConversationItemInputAudioTranscriptionCompleted struct {
	ItemId       string
	ContentIndex int
	Transcript   string
	Usage        map[string]any
}

func (p *ServerEventParamConversationItemInputAudioTranscriptionCompleted) New(m map[string]any) error {
	if v, ok := m["item_id"].(string); ok {
		p.ItemId = v
	} else {
		return errors.New("missing item_id")
	}
	if v, ok := asInt(m["content_index"]); ok {
		p.ContentIndex = v
	} else {
		return errors.New("missing content_index")
	}
	if v, ok := m["transcript"].(string); ok {
		p.Transcript = v
	} else {
		return errors.New("missing transcript")
	}
	if v, ok := m["usage"].(map[string]any); ok {
		p.Usage = v
	} else {
		p.Usage = nil
	}
	return nil
}

func (p *ServerEventParamConversationItemInputAudioTranscriptionCompleted) Json() map[string]any {
	return map[string]any{
		"item_id":       p.ItemId,
		"content_index": p.ContentIndex,
		"transcript":    p.Transcript,
		"usage":         p.Usage,
	}
}

// conversation.item.input_audio_transcription.delta
type ServerEventParamConversationItemInputAudioTranscriptionDelta struct {
	ItemId       string
	ContentIndex int
	Delta        string
	Obfuscation  string
}

func (p *ServerEventParamConversationItemInputAudioTranscriptionDelta) New(m map[string]any) error {
	if v, ok := m["item_id"].(string); ok {
		p.ItemId = v
	} else {
		return errors.New("missing item_id")
	}
	if v, ok := asInt(m["content_index"]); ok {
		p.ContentIndex = v
	} else {
		return errors.New("missing content_index")
	}
	if v, ok := m["delta"].(string); ok {
		p.Delta = v
	} else {
		return errors.New("missing delta")
	}
	if v, ok := m["obfuscation"].(string); ok {
		p.Obfuscation = v
	} else {
		p.Obfuscation = ""
	}
	return nil
}

func (p *ServerEventParamConversationItemInputAudioTranscriptionDelta) Json() map[string]any {
	resp := map[string]any{
		"item_id":       p.ItemId,
		"content_index": p.ContentIndex,
		"delta":         p.Delta,
	}
	if p.Obfuscation != "" {
		resp["obfuscation"] = p.Obfuscation
	}
	return resp
}

// conversation.item.input_audio_transcription.segment
type ServerEventParamConversationItemInputAudioTranscriptionSegment struct {
	ItemId       string
	ContentIndex int
	Text         string
	Id           string
	Speaker      string
	Start        float64
	End          float64
}

func (p *ServerEventParamConversationItemInputAudioTranscriptionSegment) New(m map[string]any) error {
	if v, ok := m["item_id"].(string); ok {
		p.ItemId = v
	} else {
		return errors.New("missing item_id")
	}
	if v, ok := asInt(m["content_index"]); ok {
		p.ContentIndex = v
	} else {
		return errors.New("missing content_index")
	}
	if v, ok := m["text"].(string); ok {
		p.Text = v
	} else {
		return errors.New("missing text")
	}
	if v, ok := m["id"].(string); ok {
		p.Id = v
	} else {
		return errors.New("missing id")
	}
	if v, ok := m["speaker"].(string); ok {
		p.Speaker = v
	} else {
		p.Speaker = ""
	}
	if v, ok := asFloat64(m["start"]); ok {
		p.Start = v
	} else {
		return errors.New("missing start")
	}
	if v, ok := asFloat64(m["end"]); ok {
		p.End = v
	} else {
		return errors.New("missing end")
	}
	return nil
}

func (p *ServerEventParamConversationItemInputAudioTranscriptionSegment) Json() map[string]any {
	resp := map[string]any{
		"item_id":       p.ItemId,
		"content_index": p.ContentIndex,
		"text":          p.Text,
		"id":            p.Id,
		"start":         p.Start,
		"end":           p.End,
	}
	if p.Speaker != "" {
		resp["speaker"] = p.Speaker
	}
	return resp
}

// conversation.item.input_audio_transcription.failed
type ServerEventParamConversationItemInputAudioTranscriptionFailed struct {
	ItemId       string
	ContentIndex int
	Error        map[string]any
}

func (p *ServerEventParamConversationItemInputAudioTranscriptionFailed) New(m map[string]any) error {
	if v, ok := m["item_id"].(string); ok {
		p.ItemId = v
	} else {
		return errors.New("missing item_id")
	}
	if v, ok := asInt(m["content_index"]); ok {
		p.ContentIndex = v
	} else {
		return errors.New("missing content_index")
	}
	if v, ok := m["error"].(map[string]any); ok {
		p.Error = v
	} else {
		return errors.New("missing error")
	}
	return nil
}

func (p *ServerEventParamConversationItemInputAudioTranscriptionFailed) Json() map[string]any {
	return map[string]any{
		"item_id":       p.ItemId,
		"content_index": p.ContentIndex,
		"error":         p.Error,
	}
}

// conversation.item.truncated
type ServerEventParamConversationItemTruncated struct {
	ItemId       string
	ContentIndex int
	AudioEndMs   int
}

func (p *ServerEventParamConversationItemTruncated) New(m map[string]any) error {
	if v, ok := m["item_id"].(string); ok {
		p.ItemId = v
	} else {
		return errors.New("missing item_id")
	}
	if v, ok := asInt(m["content_index"]); ok {
		p.ContentIndex = v
	} else {
		return errors.New("missing content_index")
	}
	if v, ok := asInt(m["audio_end_ms"]); ok {
		p.AudioEndMs = v
	} else {
		return errors.New("missing audio_end_ms")
	}
	return nil
}

func (p *ServerEventParamConversationItemTruncated) Json() map[string]any {
	return map[string]any{
		"item_id":       p.ItemId,
		"content_index": p.ContentIndex,
		"audio_end_ms":  p.AudioEndMs,
	}
}

// conversation.item.deleted
type ServerEventParamConversationItemDeleted struct {
	ItemId string
}

func (p *ServerEventParamConversationItemDeleted) New(m map[string]any) error {
	if v, ok := m["item_id"].(string); ok {
		p.ItemId = v
	} else {
		return errors.New("missing item_id")
	}
	return nil
}

func (p *ServerEventParamConversationItemDeleted) Json() map[string]any {
	return map[string]any{
		"item_id": p.ItemId,
	}
}

// input_audio_buffer.committed
type ServerEventParamInputAudioBufferCommitted struct {
	PreviousItemId any
	ItemId         string
}

func (p *ServerEventParamInputAudioBufferCommitted) New(m map[string]any) error {
	if v, ok := m["previous_item_id"]; ok {
		p.PreviousItemId = v
	} else {
		p.PreviousItemId = nil
	}
	if v, ok := m["item_id"].(string); ok {
		p.ItemId = v
	} else {
		return errors.New("missing item_id")
	}
	return nil
}

func (p *ServerEventParamInputAudioBufferCommitted) Json() map[string]any {
	return map[string]any{
		"previous_item_id": p.PreviousItemId,
		"item_id":          p.ItemId,
	}
}

// input_audio_buffer.cleared
type ServerEventParamInputAudioBufferCleared struct{}

func (p *ServerEventParamInputAudioBufferCleared) New(m map[string]any) error {
	return nil
}

func (p *ServerEventParamInputAudioBufferCleared) Json() map[string]any {
	return map[string]any{}
}

// input_audio_buffer.speech_started
type ServerEventParamInputAudioBufferSpeechStarted struct {
	AudioStartMs int
	ItemId       string
}

func (p *ServerEventParamInputAudioBufferSpeechStarted) New(m map[string]any) error {
	if v, ok := asInt(m["audio_start_ms"]); ok {
		p.AudioStartMs = v
	} else {
		return errors.New("missing audio_start_ms")
	}
	if v, ok := m["item_id"].(string); ok {
		p.ItemId = v
	} else {
		return errors.New("missing item_id")
	}
	return nil
}

func (p *ServerEventParamInputAudioBufferSpeechStarted) Json() map[string]any {
	return map[string]any{
		"audio_start_ms": p.AudioStartMs,
		"item_id":        p.ItemId,
	}
}

// input_audio_buffer.speech_stopped
type ServerEventParamInputAudioBufferSpeechStopped struct {
	AudioEndMs int
	ItemId     string
}

func (p *ServerEventParamInputAudioBufferSpeechStopped) New(m map[string]any) error {
	if v, ok := asInt(m["audio_end_ms"]); ok {
		p.AudioEndMs = v
	} else {
		return errors.New("missing audio_end_ms")
	}
	if v, ok := m["item_id"].(string); ok {
		p.ItemId = v
	} else {
		return errors.New("missing item_id")
	}
	return nil
}

func (p *ServerEventParamInputAudioBufferSpeechStopped) Json() map[string]any {
	return map[string]any{
		"audio_end_ms": p.AudioEndMs,
		"item_id":      p.ItemId,
	}
}

// input_audio_buffer.timeout_triggered
type ServerEventParamInputAudioBufferTimeoutTriggered struct {
	AudioStartMs int
	AudioEndMs   int
	ItemId       string
}

func (p *ServerEventParamInputAudioBufferTimeoutTriggered) New(m map[string]any) error {
	if v, ok := asInt(m["audio_start_ms"]); ok {
		p.AudioStartMs = v
	} else {
		return errors.New("missing audio_start_ms")
	}
	if v, ok := asInt(m["audio_end_ms"]); ok {
		p.AudioEndMs = v
	} else {
		return errors.New("missing audio_end_ms")
	}
	if v, ok := m["item_id"].(string); ok {
		p.ItemId = v
	} else {
		return errors.New("missing item_id")
	}
	return nil
}

func (p *ServerEventParamInputAudioBufferTimeoutTriggered) Json() map[string]any {
	return map[string]any{
		"audio_start_ms": p.AudioStartMs,
		"audio_end_ms":   p.AudioEndMs,
		"item_id":        p.ItemId,
	}
}

// output_audio_buffer.started
type ServerEventParamOutputAudioBufferStarted struct {
	ResponseId string
}

func (p *ServerEventParamOutputAudioBufferStarted) New(m map[string]any) error {
	if v, ok := m["response_id"].(string); ok {
		p.ResponseId = v
	} else {
		return errors.New("missing response_id")
	}
	return nil
}

func (p *ServerEventParamOutputAudioBufferStarted) Json() map[string]any {
	return map[string]any{
		"response_id": p.ResponseId,
	}
}

// output_audio_buffer.stopped
type ServerEventParamOutputAudioBufferStopped struct {
	ResponseId string
}

func (p *ServerEventParamOutputAudioBufferStopped) New(m map[string]any) error {
	if v, ok := m["response_id"].(string); ok {
		p.ResponseId = v
	} else {
		return errors.New("missing response_id")
	}
	return nil
}

func (p *ServerEventParamOutputAudioBufferStopped) Json() map[string]any {
	return map[string]any{
		"response_id": p.ResponseId,
	}
}

// output_audio_buffer.cleared
type ServerEventParamOutputAudioBufferCleared struct {
	ResponseId string
}

func (p *ServerEventParamOutputAudioBufferCleared) New(m map[string]any) error {
	if v, ok := m["response_id"].(string); ok {
		p.ResponseId = v
	} else {
		return errors.New("missing response_id")
	}
	return nil
}

func (p *ServerEventParamOutputAudioBufferCleared) Json() map[string]any {
	return map[string]any{
		"response_id": p.ResponseId,
	}
}

// response.created
type ServerEventParamResponseCreated struct {
	Response map[string]any
}

func (p *ServerEventParamResponseCreated) New(m map[string]any) error {
	if v, ok := m["response"].(map[string]any); ok {
		p.Response = v
	} else {
		return errors.New("missing response")
	}
	return nil
}

func (p *ServerEventParamResponseCreated) Json() map[string]any {
	return map[string]any{
		"response": p.Response,
	}
}

// response.done
type ServerEventParamResponseDone struct {
	Response map[string]any
}

func (p *ServerEventParamResponseDone) New(m map[string]any) error {
	if v, ok := m["response"].(map[string]any); ok {
		p.Response = v
	} else {
		return errors.New("missing response")
	}
	return nil
}

func (p *ServerEventParamResponseDone) Json() map[string]any {
	return map[string]any{
		"response": p.Response,
	}
}

// response.output_item.added
type ServerEventParamResponseOutputItemAdded struct {
	ResponseId  string
	OutputIndex int
	Item        map[string]any
}

func (p *ServerEventParamResponseOutputItemAdded) New(m map[string]any) error {
	if v, ok := m["response_id"].(string); ok {
		p.ResponseId = v
	} else {
		return errors.New("missing response_id")
	}
	if v, ok := asInt(m["output_index"]); ok {
		p.OutputIndex = v
	} else {
		return errors.New("missing output_index")
	}
	if v, ok := m["item"].(map[string]any); ok {
		p.Item = v
	} else {
		return errors.New("missing item")
	}
	return nil
}

func (p *ServerEventParamResponseOutputItemAdded) Json() map[string]any {
	return map[string]any{
		"response_id":  p.ResponseId,
		"output_index": p.OutputIndex,
		"item":         p.Item,
	}
}

// response.output_item.done
type ServerEventParamResponseOutputItemDone struct {
	ResponseId  string
	OutputIndex int
	Item        map[string]any
}

func (p *ServerEventParamResponseOutputItemDone) New(m map[string]any) error {
	if v, ok := m["response_id"].(string); ok {
		p.ResponseId = v
	} else {
		return errors.New("missing response_id")
	}
	if v, ok := asInt(m["output_index"]); ok {
		p.OutputIndex = v
	} else {
		return errors.New("missing output_index")
	}
	if v, ok := m["item"].(map[string]any); ok {
		p.Item = v
	} else {
		return errors.New("missing item")
	}
	return nil
}

func (p *ServerEventParamResponseOutputItemDone) Json() map[string]any {
	return map[string]any{
		"response_id":  p.ResponseId,
		"output_index": p.OutputIndex,
		"item":         p.Item,
	}
}

// response.content_part.added
type ServerEventParamResponseContentPartAdded struct {
	ResponseId   string
	ItemId       string
	OutputIndex  int
	ContentIndex int
	Part         map[string]any
}

func (p *ServerEventParamResponseContentPartAdded) New(m map[string]any) error {
	if v, ok := m["response_id"].(string); ok {
		p.ResponseId = v
	} else {
		return errors.New("missing response_id")
	}
	if v, ok := m["item_id"].(string); ok {
		p.ItemId = v
	} else {
		return errors.New("missing item_id")
	}
	if v, ok := asInt(m["output_index"]); ok {
		p.OutputIndex = v
	} else {
		return errors.New("missing output_index")
	}
	if v, ok := asInt(m["content_index"]); ok {
		p.ContentIndex = v
	} else {
		return errors.New("missing content_index")
	}
	if v, ok := m["part"].(map[string]any); ok {
		p.Part = v
	} else {
		return errors.New("missing part")
	}
	return nil
}

func (p *ServerEventParamResponseContentPartAdded) Json() map[string]any {
	return map[string]any{
		"response_id":   p.ResponseId,
		"item_id":       p.ItemId,
		"output_index":  p.OutputIndex,
		"content_index": p.ContentIndex,
		"part":          p.Part,
	}
}

// response.content_part.done
type ServerEventParamResponseContentPartDone struct {
	ResponseId   string
	ItemId       string
	OutputIndex  int
	ContentIndex int
	Part         map[string]any
}

func (p *ServerEventParamResponseContentPartDone) New(m map[string]any) error {
	if v, ok := m["response_id"].(string); ok {
		p.ResponseId = v
	} else {
		return errors.New("missing response_id")
	}
	if v, ok := m["item_id"].(string); ok {
		p.ItemId = v
	} else {
		return errors.New("missing item_id")
	}
	if v, ok := asInt(m["output_index"]); ok {
		p.OutputIndex = v
	} else {
		return errors.New("missing output_index")
	}
	if v, ok := asInt(m["content_index"]); ok {
		p.ContentIndex = v
	} else {
		return errors.New("missing content_index")
	}
	if v, ok := m["part"].(map[string]any); ok {
		p.Part = v
	} else {
		return errors.New("missing part")
	}
	return nil
}

func (p *ServerEventParamResponseContentPartDone) Json() map[string]any {
	return map[string]any{
		"response_id":   p.ResponseId,
		"item_id":       p.ItemId,
		"output_index":  p.OutputIndex,
		"content_index": p.ContentIndex,
		"part":          p.Part,
	}
}

// response.output_text.delta
type ServerEventParamResponseOutputTextDelta struct {
	ResponseId   string
	ItemId       string
	OutputIndex  int
	ContentIndex int
	Delta        string
}

func (p *ServerEventParamResponseOutputTextDelta) New(m map[string]any) error {
	if v, ok := m["response_id"].(string); ok {
		p.ResponseId = v
	} else {
		return errors.New("missing response_id")
	}
	if v, ok := m["item_id"].(string); ok {
		p.ItemId = v
	} else {
		return errors.New("missing item_id")
	}
	if v, ok := asInt(m["output_index"]); ok {
		p.OutputIndex = v
	} else {
		return errors.New("missing output_index")
	}
	if v, ok := asInt(m["content_index"]); ok {
		p.ContentIndex = v
	} else {
		return errors.New("missing content_index")
	}
	if v, ok := m["delta"].(string); ok {
		p.Delta = v
	} else {
		return errors.New("missing delta")
	}
	return nil
}

func (p *ServerEventParamResponseOutputTextDelta) Json() map[string]any {
	return map[string]any{
		"response_id":   p.ResponseId,
		"item_id":       p.ItemId,
		"output_index":  p.OutputIndex,
		"content_index": p.ContentIndex,
		"delta":         p.Delta,
	}
}

// response.output_text.done
type ServerEventParamResponseOutputTextDone struct {
	ResponseId   string
	ItemId       string
	OutputIndex  int
	ContentIndex int
	Text         string
}

func (p *ServerEventParamResponseOutputTextDone) New(m map[string]any) error {
	if v, ok := m["response_id"].(string); ok {
		p.ResponseId = v
	} else {
		return errors.New("missing response_id")
	}
	if v, ok := m["item_id"].(string); ok {
		p.ItemId = v
	} else {
		return errors.New("missing item_id")
	}
	if v, ok := asInt(m["output_index"]); ok {
		p.OutputIndex = v
	} else {
		return errors.New("missing output_index")
	}
	if v, ok := asInt(m["content_index"]); ok {
		p.ContentIndex = v
	} else {
		return errors.New("missing content_index")
	}
	if v, ok := m["text"].(string); ok {
		p.Text = v
	} else {
		return errors.New("missing text")
	}
	return nil
}

func (p *ServerEventParamResponseOutputTextDone) Json() map[string]any {
	return map[string]any{
		"response_id":   p.ResponseId,
		"item_id":       p.ItemId,
		"output_index":  p.OutputIndex,
		"content_index": p.ContentIndex,
		"text":          p.Text,
	}
}

// response.output_audio_transcript.delta
type ServerEventParamResponseOutputAudioTranscriptDelta struct {
	ResponseId   string
	ItemId       string
	OutputIndex  int
	ContentIndex int
	Delta        string
}

func (p *ServerEventParamResponseOutputAudioTranscriptDelta) New(m map[string]any) error {
	if v, ok := m["response_id"].(string); ok {
		p.ResponseId = v
	} else {
		return errors.New("missing response_id")
	}
	if v, ok := m["item_id"].(string); ok {
		p.ItemId = v
	} else {
		return errors.New("missing item_id")
	}
	if v, ok := asInt(m["output_index"]); ok {
		p.OutputIndex = v
	} else {
		return errors.New("missing output_index")
	}
	if v, ok := asInt(m["content_index"]); ok {
		p.ContentIndex = v
	} else {
		return errors.New("missing content_index")
	}
	if v, ok := m["delta"].(string); ok {
		p.Delta = v
	} else {
		return errors.New("missing delta")
	}
	return nil
}

func (p *ServerEventParamResponseOutputAudioTranscriptDelta) Json() map[string]any {
	return map[string]any{
		"response_id":   p.ResponseId,
		"item_id":       p.ItemId,
		"output_index":  p.OutputIndex,
		"content_index": p.ContentIndex,
		"delta":         p.Delta,
	}
}

// response.output_audio_transcript.done
type ServerEventParamResponseOutputAudioTranscriptDone struct {
	ResponseId   string
	ItemId       string
	OutputIndex  int
	ContentIndex int
	Transcript   string
}

func (p *ServerEventParamResponseOutputAudioTranscriptDone) New(m map[string]any) error {
	if v, ok := m["response_id"].(string); ok {
		p.ResponseId = v
	} else {
		return errors.New("missing response_id")
	}
	if v, ok := m["item_id"].(string); ok {
		p.ItemId = v
	} else {
		return errors.New("missing item_id")
	}
	if v, ok := asInt(m["output_index"]); ok {
		p.OutputIndex = v
	} else {
		return errors.New("missing output_index")
	}
	if v, ok := asInt(m["content_index"]); ok {
		p.ContentIndex = v
	} else {
		return errors.New("missing content_index")
	}
	if v, ok := m["transcript"].(string); ok {
		p.Transcript = v
	} else {
		return errors.New("missing transcript")
	}
	return nil
}

func (p *ServerEventParamResponseOutputAudioTranscriptDone) Json() map[string]any {
	return map[string]any{
		"response_id":   p.ResponseId,
		"item_id":       p.ItemId,
		"output_index":  p.OutputIndex,
		"content_index": p.ContentIndex,
		"transcript":    p.Transcript,
	}
}

// response.output_audio.delta
type ServerEventParamResponseOutputAudioDelta struct {
	ResponseId   string
	ItemId       string
	OutputIndex  int
	ContentIndex int
	Delta        string
}

func (p *ServerEventParamResponseOutputAudioDelta) New(m map[string]any) error {
	if v, ok := m["response_id"].(string); ok {
		p.ResponseId = v
	} else {
		return errors.New("missing response_id")
	}
	if v, ok := m["item_id"].(string); ok {
		p.ItemId = v
	} else {
		return errors.New("missing item_id")
	}
	if v, ok := asInt(m["output_index"]); ok {
		p.OutputIndex = v
	} else {
		return errors.New("missing output_index")
	}
	if v, ok := asInt(m["content_index"]); ok {
		p.ContentIndex = v
	} else {
		return errors.New("missing content_index")
	}
	if v, ok := m["delta"].(string); ok {
		p.Delta = v
	} else {
		return errors.New("missing delta")
	}
	return nil
}

func (p *ServerEventParamResponseOutputAudioDelta) Json() map[string]any {
	return map[string]any{
		"response_id":   p.ResponseId,
		"item_id":       p.ItemId,
		"output_index":  p.OutputIndex,
		"content_index": p.ContentIndex,
		"delta":         p.Delta,
	}
}

// response.output_audio.done
type ServerEventParamResponseOutputAudioDone struct {
	ResponseId   string
	ItemId       string
	OutputIndex  int
	ContentIndex int
}

func (p *ServerEventParamResponseOutputAudioDone) New(m map[string]any) error {
	if v, ok := m["response_id"].(string); ok {
		p.ResponseId = v
	} else {
		return errors.New("missing response_id")
	}
	if v, ok := m["item_id"].(string); ok {
		p.ItemId = v
	} else {
		return errors.New("missing item_id")
	}
	if v, ok := asInt(m["output_index"]); ok {
		p.OutputIndex = v
	} else {
		return errors.New("missing output_index")
	}
	if v, ok := asInt(m["content_index"]); ok {
		p.ContentIndex = v
	} else {
		return errors.New("missing content_index")
	}
	return nil
}

func (p *ServerEventParamResponseOutputAudioDone) Json() map[string]any {
	return map[string]any{
		"response_id":   p.ResponseId,
		"item_id":       p.ItemId,
		"output_index":  p.OutputIndex,
		"content_index": p.ContentIndex,
	}
}

// response.function_call_arguments.delta
type ServerEventParamResponseFunctionCallArgumentsDelta struct {
	ResponseId  string
	ItemId      string
	OutputIndex int
	CallId      string
	Delta       string
}

func (p *ServerEventParamResponseFunctionCallArgumentsDelta) New(m map[string]any) error {
	if v, ok := m["response_id"].(string); ok {
		p.ResponseId = v
	} else {
		return errors.New("missing response_id")
	}
	if v, ok := m["item_id"].(string); ok {
		p.ItemId = v
	} else {
		return errors.New("missing item_id")
	}
	if v, ok := asInt(m["output_index"]); ok {
		p.OutputIndex = v
	} else {
		return errors.New("missing output_index")
	}
	if v, ok := m["call_id"].(string); ok {
		p.CallId = v
	} else {
		return errors.New("missing call_id")
	}
	if v, ok := m["delta"].(string); ok {
		p.Delta = v
	} else {
		return errors.New("missing delta")
	}
	return nil
}

func (p *ServerEventParamResponseFunctionCallArgumentsDelta) Json() map[string]any {
	return map[string]any{
		"response_id":  p.ResponseId,
		"item_id":      p.ItemId,
		"output_index": p.OutputIndex,
		"call_id":      p.CallId,
		"delta":        p.Delta,
	}
}

// response.function_call_arguments.done
type ServerEventParamResponseFunctionCallArgumentsDone struct {
	ResponseId  string
	ItemId      string
	OutputIndex int
	CallId      string
	Arguments   string
}

func (p *ServerEventParamResponseFunctionCallArgumentsDone) New(m map[string]any) error {
	if v, ok := m["response_id"].(string); ok {
		p.ResponseId = v
	} else {
		return errors.New("missing response_id")
	}
	if v, ok := m["item_id"].(string); ok {
		p.ItemId = v
	} else {
		return errors.New("missing item_id")
	}
	if v, ok := asInt(m["output_index"]); ok {
		p.OutputIndex = v
	} else {
		return errors.New("missing output_index")
	}
	if v, ok := m["call_id"].(string); ok {
		p.CallId = v
	} else {
		return errors.New("missing call_id")
	}
	if v, ok := m["arguments"].(string); ok {
		p.Arguments = v
	} else {
		return errors.New("missing arguments")
	}
	return nil
}

func (p *ServerEventParamResponseFunctionCallArgumentsDone) Json() map[string]any {
	return map[string]any{
		"response_id":  p.ResponseId,
		"item_id":      p.ItemId,
		"output_index": p.OutputIndex,
		"call_id":      p.CallId,
		"arguments":    p.Arguments,
	}
}

// response.mcp_call_arguments.delta
type ServerEventParamResponseMCPCallArgumentsDelta struct {
	ResponseId  string
	ItemId      string
	OutputIndex int
	Delta       string
}

func (p *ServerEventParamResponseMCPCallArgumentsDelta) New(m map[string]any) error {
	if v, ok := m["response_id"].(string); ok {
		p.ResponseId = v
	} else {
		return errors.New("missing response_id")
	}
	if v, ok := m["item_id"].(string); ok {
		p.ItemId = v
	} else {
		return errors.New("missing item_id")
	}
	if v, ok := asInt(m["output_index"]); ok {
		p.OutputIndex = v
	} else {
		return errors.New("missing output_index")
	}
	if v, ok := m["delta"].(string); ok {
		p.Delta = v
	} else {
		return errors.New("missing delta")
	}
	return nil
}

func (p *ServerEventParamResponseMCPCallArgumentsDelta) Json() map[string]any {
	return map[string]any{
		"response_id":  p.ResponseId,
		"item_id":      p.ItemId,
		"output_index": p.OutputIndex,
		"delta":        p.Delta,
	}
}

// response.mcp_call_arguments.done
type ServerEventParamResponseMCPCallArgumentsDone struct {
	ResponseId  string
	ItemId      string
	OutputIndex int
	Arguments   string
}

func (p *ServerEventParamResponseMCPCallArgumentsDone) New(m map[string]any) error {
	if v, ok := m["response_id"].(string); ok {
		p.ResponseId = v
	} else {
		return errors.New("missing response_id")
	}
	if v, ok := m["item_id"].(string); ok {
		p.ItemId = v
	} else {
		return errors.New("missing item_id")
	}
	if v, ok := asInt(m["output_index"]); ok {
		p.OutputIndex = v
	} else {
		return errors.New("missing output_index")
	}
	if v, ok := m["arguments"].(string); ok {
		p.Arguments = v
	} else {
		return errors.New("missing arguments")
	}
	return nil
}

func (p *ServerEventParamResponseMCPCallArgumentsDone) Json() map[string]any {
	return map[string]any{
		"response_id":  p.ResponseId,
		"item_id":      p.ItemId,
		"output_index": p.OutputIndex,
		"arguments":    p.Arguments,
	}
}

// response.mcp_call.in_progress
type ServerEventParamResponseMCPCallInProgress struct {
	OutputIndex int
	ItemId      string
}

func (p *ServerEventParamResponseMCPCallInProgress) New(m map[string]any) error {
	if v, ok := asInt(m["output_index"]); ok {
		p.OutputIndex = v
	} else {
		return errors.New("missing output_index")
	}
	if v, ok := m["item_id"].(string); ok {
		p.ItemId = v
	} else {
		return errors.New("missing item_id")
	}
	return nil
}

func (p *ServerEventParamResponseMCPCallInProgress) Json() map[string]any {
	return map[string]any{
		"output_index": p.OutputIndex,
		"item_id":      p.ItemId,
	}
}

// response.mcp_call.completed
type ServerEventParamResponseMCPCallCompleted struct {
	OutputIndex int
	ItemId      string
}

func (p *ServerEventParamResponseMCPCallCompleted) New(m map[string]any) error {
	if v, ok := asInt(m["output_index"]); ok {
		p.OutputIndex = v
	} else {
		return errors.New("missing output_index")
	}
	if v, ok := m["item_id"].(string); ok {
		p.ItemId = v
	} else {
		return errors.New("missing item_id")
	}
	return nil
}

func (p *ServerEventParamResponseMCPCallCompleted) Json() map[string]any {
	return map[string]any{
		"output_index": p.OutputIndex,
		"item_id":      p.ItemId,
	}
}

// response.mcp_call.failed
type ServerEventParamResponseMCPCallFailed struct {
	OutputIndex int
	ItemId      string
}

func (p *ServerEventParamResponseMCPCallFailed) New(m map[string]any) error {
	if v, ok := asInt(m["output_index"]); ok {
		p.OutputIndex = v
	} else {
		return errors.New("missing output_index")
	}
	if v, ok := m["item_id"].(string); ok {
		p.ItemId = v
	} else {
		return errors.New("missing item_id")
	}
	return nil
}

func (p *ServerEventParamResponseMCPCallFailed) Json() map[string]any {
	return map[string]any{
		"output_index": p.OutputIndex,
		"item_id":      p.ItemId,
	}
}

// mcp_list_tools.in_progress
type ServerEventParamMCPListToolsInProgress struct {
	ItemId string
}

func (p *ServerEventParamMCPListToolsInProgress) New(m map[string]any) error {
	if v, ok := m["item_id"].(string); ok {
		p.ItemId = v
	} else {
		return errors.New("missing item_id")
	}
	return nil
}

func (p *ServerEventParamMCPListToolsInProgress) Json() map[string]any {
	return map[string]any{
		"item_id": p.ItemId,
	}
}

// mcp_list_tools.completed
type ServerEventParamMCPListToolsCompleted struct {
	ItemId string
}

func (p *ServerEventParamMCPListToolsCompleted) New(m map[string]any) error {
	if v, ok := m["item_id"].(string); ok {
		p.ItemId = v
	} else {
		return errors.New("missing item_id")
	}
	return nil
}

func (p *ServerEventParamMCPListToolsCompleted) Json() map[string]any {
	return map[string]any{
		"item_id": p.ItemId,
	}
}

// mcp_list_tools.failed
type ServerEventParamMCPListToolsFailed struct {
	ItemId string
}

func (p *ServerEventParamMCPListToolsFailed) New(m map[string]any) error {
	if v, ok := m["item_id"].(string); ok {
		p.ItemId = v
	} else {
		return errors.New("missing item_id")
	}
	return nil
}

func (p *ServerEventParamMCPListToolsFailed) Json() map[string]any {
	return map[string]any{
		"item_id": p.ItemId,
	}
}

// rate_limits.updated
type ServerEventParamRatelimitsUpdated struct {
	RateLimits []map[string]any
}

func (p *ServerEventParamRatelimitsUpdated) New(m map[string]any) error {
	v, ok := m["rate_limits"]
	if !ok {
		return errors.New("missing rate_limits")
	}
	switch rr := v.(type) {
	case []any:
		res := make([]map[string]any, 0, len(rr))
		for _, r := range rr {
			if rm, ok := r.(map[string]any); ok {
				res = append(res, rm)
			} else {
				return errors.New("invalid element in rate_limits")
			}
		}
		p.RateLimits = res
	case []map[string]any:
		p.RateLimits = rr
	default:
		return errors.New("invalid rate_limits")
	}
	return nil
}

func (p *ServerEventParamRatelimitsUpdated) Json() map[string]any {
	return map[string]any{
		"rate_limits": p.RateLimits,
	}
}
