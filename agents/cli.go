package agents

import (
	"github.com/bt-bridge/openai-realtime/shared"
	"github.com/openai/openai-go/v3/realtime"
)

type CLIAgent struct {
	printer *shared.Printer
}

func (a *CLIAgent) Spawn(
	logger shared.LoggerAdapter,
	apiKey string,
	cfg *realtime.RealtimeSessionCreateRequestParam,
	printer *shared.Printer,
	baseUrl ...string,
) (<-chan struct{}, error) {
	return nil, nil // TODO
}

func (a *CLIAgent) Close() error {
	return nil // TODO
}
