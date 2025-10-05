package realtime

import (
	"sync"

	"github.com/openai/openai-go/v3/realtime"
	"github.com/pion/webrtc/v4"
)

// SessionState holds the WebRTC session state
type SessionState struct {
	cfg           *realtime.RealtimeSessionCreateRequestParam
	pc            *webrtc.PeerConnection
	dc            *webrtc.DataChannel
	modelMessages map[string]string
	userMessages  map[string]string
	modelSpeaking bool
	inputReady    bool
	mu            sync.Mutex
}

// NewSessionState initializes a new session state
func NewSessionState() *SessionState {
	return &SessionState{
		modelMessages: make(map[string]string),
		userMessages:  make(map[string]string),
	}
}
