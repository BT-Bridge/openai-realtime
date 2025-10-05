package shared

import "errors"

var (
	ErrUnauthorized          = errors.New("unauthorized")
	ErrForbidden             = errors.New("forbidden")
	ErrNoLogger              = errors.New("no logger provided")
	ErrNoConfig              = errors.New("no config provided")
	ErrClientNotInitialized  = errors.New("client not initialized")
	ErrNoEventHandler        = errors.New("no event handler provided")
	ErrNoAPIKey              = errors.New("no API key provided")
	ErrSessionAlreadyRunning = errors.New("session already running")
	ErrTRHandlerAlreadySet   = errors.New("track remote handler already set")
	ErrTLHandlerAlreadySet   = errors.New("track local handler already set")
	ErrEHandlerAlreadySet    = errors.New("event handler already set")
)
