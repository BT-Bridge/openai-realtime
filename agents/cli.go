package agents

import (
	"context"
	"errors"
	"sync"
	"time"

	pkg "github.com/bt-bridge/openai-realtime"
	"github.com/bt-bridge/openai-realtime/shared"
	"github.com/bt-bridge/openai-realtime/tools"
	"github.com/goccy/go-yaml"
	"github.com/openai/openai-go/v3/realtime"
	"github.com/pion/mediadevices"
	"github.com/pion/mediadevices/pkg/codec/opus"
	_ "github.com/pion/mediadevices/pkg/driver/microphone"
	"github.com/pion/mediadevices/pkg/prop"
	"github.com/pion/webrtc/v4"
	"go.uber.org/zap"
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

func (a *CLIAgent) Done() <-chan struct{} {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.client == nil {
		panic("client is nil")
	}
	return a.client.Done()
}

func (a *CLIAgent) Spawn(
	ctx context.Context,
	logger shared.LoggerAdapter,
	apiKey string,
	cfg *realtime.RealtimeSessionCreateRequestParam,
	language string,
	printer *shared.Printer,
	baseUrl ...string,
) error {
	if logger == nil {
		return shared.ErrNoLogger
	}
	if apiKey == "" {
		return shared.ErrNoAPIKey
	}
	if cfg == nil {
		return shared.ErrNoConfig
	}
	if printer == nil {
		return errors.New("no printer provided")
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
		a.client, err = pkg.NewClient(ctx, a.logger, apiKey, language, baseUrl[0])
	} else {
		a.client, err = pkg.NewClient(ctx, a.logger, apiKey, language, "")
	}
	if err != nil {
		a.logger.Error("creating client", err)
		return err
	}
	a.logger.Info("client created successfully")

	// Setting up session config
	if err := a.client.SetConfig(cfg); err != nil {
		a.logger.Error("setting up session config", err)
		return err
	}
	a.logger.Info("session config set up successfully")
	if err := a.printer.Writeln("📋 Session Config\n", 0); err != nil {
		a.logger.Error("printing session config message", err)
	}
	yamlBytes, err := yaml.MarshalWithOptions(cfg, yaml.UseJSONMarshaler())
	if err != nil {
		a.logger.Error("marshaling session config to yaml", err)
		return err
	}
	if err := a.printer.Write(string(yamlBytes), 1); err != nil {
		a.logger.Error("printing session config", err)
		return err
	}
	// Getting microphone access and stream
	if err := a.printer.Writeln("\n\n🎤 Accessing microphone...", 0); err != nil {
		a.logger.Error("printing microphone access message", err)
	}
	opusParams, err := opus.NewParams()
	if err != nil {
		a.logger.Error("creating opus params", err)
		return err
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
		return err
	}
	if audioTracks := micStream.GetAudioTracks(); len(audioTracks) > 0 {
		a.micTrack = audioTracks[0]
	} else {
		a.logger.Error("no audio track found in microphone stream", errors.New("no audio track"))
		if err := a.printer.Writeln("❌ No audio track found in microphone stream.\n", 0); err != nil {
			a.logger.Error("printing no audio track found message", err)
		}
		return errors.New("no audio track found in microphone stream")
	}
	a.logger.Info("microphone stream obtained successfully")
	if err := a.printer.Writeln("✅ Microphone access granted.\n", 0); err != nil {
		a.logger.Error("printing microphone access success message", err)
	}

	// Setting up track remote handler
	// This will play the audio received from the session
	// to the default audio output device (e.g., speakers or headphones)
	if err := a.printer.Writeln("🔈 Setting up track remote handler...", 0); err != nil {
		a.logger.Error("printing track remote handler setup message", err)
	}
	err = a.client.RegisterTrackRemoteHandler(func(track *webrtc.TrackRemote) {
		a.logger.Info(
			"received remote track",
			zap.String("kind", track.Kind().String()),
			zap.String("codec", track.Codec().MimeType),
		)
		tools.PlayRemoteAudio(ctx, a.logger, track, 200, 10)
	})
	if err != nil {
		a.logger.Error("registering track remote handler", err)
		if err := a.printer.Writeln("❌ Failed to set up track remote handler. Audio playback may not work as expected.\n", 0); err != nil {
			a.logger.Error("printing track remote handler setup failure message", err)
		}
		return err
	}
	a.logger.Info("track remote handler registered successfully")
	if err := a.printer.Writeln("✅ Track remote handler set up successfully.\n", 0); err != nil {
		a.logger.Error("printing track remote handler setup success message", err)
	}

	// Setting up track local handler
	// This will send the audio from the microphone to the session
	if err := a.printer.Writeln("🎧 Setting up track local handler...", 0); err != nil {
		a.logger.Error("printing track local handler setup message", err)
	}
	err = a.client.RegisterTrackLocalHandler(func(track *webrtc.TrackLocalStaticSample) {
		tools.StreamLocalAudio(ctx, a.logger, track, a.micTrack, time.Duration(opusParams.Latency))
	})
	if err != nil {
		a.logger.Error("registering track local handler", err)
		if err := a.printer.Writeln("❌ Failed to set up track local handler. Audio streaming may not work as expected.\n", 0); err != nil {
			a.logger.Error("printing track local handler setup failure message", err)
		}
		return err
	}
	a.logger.Info("track local handler registered successfully")
	if err := a.printer.Writeln("✅ Track local handler set up successfully.\n", 0); err != nil {
		a.logger.Error("printing track local handler setup success message", err)
	}

	// Setting up event handler
	if err := a.printer.Writeln("🚥 Setting up event handler...", 0); err != nil {
		a.logger.Error("printing event handler setup message", err)
	}
	err = a.client.RegisterEventHandler(a.eventHandler)
	if err != nil {
		a.logger.Error("registering event handler", err)
		if err := a.printer.Writeln("❌ Failed to set up event handler. Events may not be handled as expected.\n", 0); err != nil {
			a.logger.Error("printing event handler setup failure message", err)
		}
		return err
	}
	a.logger.Info("event handler registered successfully")
	if err := a.printer.Writeln("✅ Event handler set up successfully.\n", 0); err != nil {
		a.logger.Error("printing event handler setup success message", err)
	}

	// Starting session
	if err := a.printer.Writeln("🚀 Starting session...", 0); err != nil {
		a.logger.Error("printing session starting message", err)
	}
	if err := a.client.Start(); err != nil {
		a.logger.Error("starting session", err)
		if err := a.printer.Writeln("❌ Failed to start session. Please check the error message above for details.\n", 0); err != nil {
			a.logger.Error("printing session starting failure message", err)
		}
		return err
	}
	a.logger.Info("session started successfully")
	if err := a.printer.Writeln("✅ Session started successfully.\n", 0); err != nil {
		a.logger.Error("printing session started success message", err)
	}
	<-a.client.Connected()
	return nil
}

func (a *CLIAgent) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.client != nil {
		if err := a.client.Close(); err != nil {
			a.logger.Error("closing client", err)
			return err
		}
	}
	return nil
}

func (a *CLIAgent) eventHandler(event *pkg.Event) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.logger.Info("received event", zap.String("type", string(event.Type)))
}
