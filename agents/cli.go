package agents

import (
	"context"
	"errors"
	"sync"

	pkg "github.com/bt-bridge/openai-realtime"
	"github.com/bt-bridge/openai-realtime/shared"
	"github.com/goccy/go-yaml"
	"github.com/openai/openai-go/v3/realtime"
	"github.com/pion/mediadevices"
	"github.com/pion/mediadevices/pkg/codec/opus"
	_ "github.com/pion/mediadevices/pkg/driver/microphone"
	"github.com/pion/mediadevices/pkg/prop"
)

type CLIState struct {
	micOpened bool
}

func NewCLIState() *CLIState {
	return &CLIState{
		micOpened: false,
	}
}

type CLIAgent struct {
	logger   shared.LoggerAdapter
	printer  *shared.Printer
	client   *pkg.Client
	state    *CLIState
	micTrack mediadevices.Track

	mu sync.Mutex
}

func (a *CLIAgent) Spawn(
	ctx context.Context,
	logger shared.LoggerAdapter,
	apiKey string,
	cfg *realtime.RealtimeSessionCreateRequestParam,
	printer *shared.Printer,
	baseUrl ...string,
) (<-chan struct{}, error) {
	if logger == nil {
		return nil, shared.ErrNoLogger
	}
	if apiKey == "" {
		return nil, shared.ErrNoAPIKey
	}
	if cfg == nil {
		return nil, shared.ErrNoConfig
	}
	if printer == nil {
		return nil, errors.New("no printer provided")
	}
	a.logger = logger
	a.printer = printer
	a.state = NewCLIState()
	a.logger.Info("spawning CLI agent")
	if err := a.printer.Writeln("🤖 Spawning CLI agent...\n", 0); err != nil {
		a.logger.Error("printing spawning message", err)
	}

	// Creating client
	var err error
	if len(baseUrl) > 0 {
		a.client, err = pkg.NewClient(ctx, a.logger, apiKey, baseUrl[0])
	} else {
		a.client, err = pkg.NewClient(ctx, a.logger, apiKey, "")
	}
	if err != nil {
		a.logger.Error("creating client", err)
		return nil, err
	}
	a.logger.Info("client created successfully")

	// Setting up session config
	if err := a.client.SetConfig(cfg); err != nil {
		a.logger.Error("setting up session config", err)
		return nil, err
	}
	a.logger.Info("session config set up successfully")
	if err := a.printer.Writeln("📋 Session Config\n", 0); err != nil {
		a.logger.Error("printing session config message", err)
	}
	yamlBytes, err := yaml.MarshalWithOptions(cfg, yaml.UseJSONMarshaler())
	if err != nil {
		a.logger.Error("marshaling session config to yaml", err)
		return nil, err
	}
	if err := a.printer.Write(string(yamlBytes), 1); err != nil {
		a.logger.Error("printing session config", err)
		return nil, err
	}
	// Getting microphone access and stream
	if err := a.printer.Writeln("\n\n🎤 Accessing microphone...\n", 0); err != nil {
		a.logger.Error("printing microphone access message", err)
	}
	opusParams, err := opus.NewParams()
	if err != nil {
		a.logger.Error("creating opus params", err)
		return nil, err
	}
	micStream, err := mediadevices.GetUserMedia(mediadevices.MediaStreamConstraints{
		Audio: func(c *mediadevices.MediaTrackConstraints) {
			c.SampleRate = prop.Int(48000)
			c.ChannelCount = prop.Int(1)
			c.SampleSize = prop.Int(16)
		},
		Codec: mediadevices.NewCodecSelector(
			mediadevices.WithAudioEncoders(&opusParams),
		),
	})
	if err != nil {
		a.logger.Error("getting microphone stream", err)
		if err := a.printer.Writeln("❌ Unable to access microphone. Please ensure that your microphone is connected and that you have granted permission to access it.\n", 0); err != nil {
			a.logger.Error("printing microphone access failure message", err)
		}
		return nil, err
	}
	if audioTracks := micStream.GetAudioTracks(); len(audioTracks) > 0 {
		a.micTrack = audioTracks[0]
	} else {
		a.logger.Error("no audio track found in microphone stream", errors.New("no audio track"))
		if err := a.printer.Writeln("❌ No audio track found in microphone stream.\n", 0); err != nil {
			a.logger.Error("printing no audio track found message", err)
		}
		return nil, errors.New("no audio track found in microphone stream")
	}
	a.logger.Info("microphone stream obtained successfully")
	if err := a.printer.Writeln("✅ Microphone access granted.\n", 0); err != nil {
		a.logger.Error("printing microphone access success message", err)
	}

	return nil, nil // TODO
}

func (a *CLIAgent) Close() error {
	return nil // TODO
}
