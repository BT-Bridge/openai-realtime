package agents

import (
	"context"
	"errors"
	"sync"
	"time"

	pkg "github.com/bridge-packages/go-openai-realtime"
	"github.com/bridge-packages/go-openai-realtime/shared"
	"github.com/bridge-packages/go-openai-realtime/tools"
	"github.com/goccy/go-yaml"
	"github.com/openai/openai-go/v3/realtime"
	"github.com/pion/mediadevices"
	"github.com/pion/mediadevices/pkg/codec/opus"
	_ "github.com/pion/mediadevices/pkg/driver/microphone"
	"github.com/pion/mediadevices/pkg/prop"
	"github.com/pion/webrtc/v4"
	"go.uber.org/zap"
)

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
	greeting string,
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
		a.client, err = pkg.NewClient(ctx, a.logger, apiKey, greeting, baseUrl[0])
	} else {
		a.client, err = pkg.NewClient(ctx, a.logger, apiKey, greeting, "")
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

func (a *CLIAgent) eventHandler(event *pkg.ServerEvent) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.logger.Info(
		"received event",
		zap.String("type", string(event.Type)),
		zap.String("event_id", event.EventId),
	)
	msg := a.state.PipeEvent(event)
	if msg != "" {
		a.printHelper(msg, 0)
	}
	switch event.Type {
	case pkg.ServerEventTypeError:
		a.logger.Error(
			"error event received",
			nil,
			zap.Any("error", event.Param),
			zap.String("event_id", event.EventId),
		)
		a.printEventHelper("❌ Error Event Received", event)
	case pkg.ServerEventTypeSessionCreated:
		a.printEventHelper("✅ Session Created", nil)
	case pkg.ServerEventTypeSessionUpdated:
		a.printEventHelper("ℹ️ Session Updated", nil)
	case pkg.ServerEventTypeConversationItemAdded:
		a.logger.Info(
			"conversation item added",
			zap.Any("item", event.Param),
			zap.String("event_id", event.EventId),
		)
	case pkg.ServerEventTypeConversationItemDone:
		a.logger.Info(
			"conversation item done",
			zap.Any("item", event.Param),
			zap.String("event_id", event.EventId),
		)
	case pkg.ServerEventTypeConversationItemRetrieved:
		a.logger.Info(
			"conversation item retrieved",
			zap.Any("item", event.Param),
			zap.String("event_id", event.EventId),
		)
	case pkg.ServerEventTypeConversationItemInputAudioTranscriptionCompleted:
		a.logger.Info(
			"conversation item input audio transcription completed",
			zap.Any("item", event.Param),
			zap.String("event_id", event.EventId),
		)
	case pkg.ServerEventTypeConversationItemInputAudioTranscriptionDelta:
		a.logger.Info(
			"conversation item input audio transcription delta",
			zap.Any("item", event.Param),
			zap.String("event_id", event.EventId),
		)
	case pkg.ServerEventTypeConversationItemInputAudioTranscriptionSegment:
		a.logger.Info(
			"conversation item input audio transcription segment",
			zap.Any("item", event.Param),
			zap.String("event_id", event.EventId),
		)
	case pkg.ServerEventTypeConversationItemInputAudioTranscriptionFailed:
		a.logger.Info(
			"conversation item input audio transcription failed",
			zap.Any("item", event.Param),
			zap.String("event_id", event.EventId),
		)
		a.printEventHelper("❌ Input Audio Transcription Failed", event)
	case pkg.ServerEventTypeConversationItemTruncated:
		a.logger.Info(
			"conversation item truncated",
			zap.Any("item", event.Param),
			zap.String("event_id", event.EventId),
		)
	case pkg.ServerEventTypeConversationItemDeleted:
		a.logger.Info(
			"conversation item deleted",
			zap.Any("item", event.Param),
			zap.String("event_id", event.EventId),
		)
	case pkg.ServerEventTypeInputAudioBufferCommitted:
		a.logger.Info(
			"input audio buffer committed",
			zap.Any("item", event.Param),
			zap.String("event_id", event.EventId),
		)
	case pkg.ServerEventTypeInputAudioBufferCleared:
		a.logger.Info(
			"input audio buffer cleared",
			zap.Any("item", event.Param),
			zap.String("event_id", event.EventId),
		)
	case pkg.ServerEventTypeInputAudioBufferSpeechStarted:
		a.logger.Info(
			"input audio buffer speech started",
			zap.Any("item", event.Param),
			zap.String("event_id", event.EventId),
		)
	case pkg.ServerEventTypeInputAudioBufferSpeechStopped:
		a.logger.Info(
			"input audio buffer speech stopped",
			zap.Any("item", event.Param),
			zap.String("event_id", event.EventId),
		)
	case pkg.ServerEventTypeInputAudioBufferTimeoutTriggered:
		a.logger.Warn(
			"input audio buffer timeout triggered",
			zap.Any("item", event.Param),
			zap.String("event_id", event.EventId),
		)
	case pkg.ServerEventTypeOutputAudioBufferStarted:
		a.logger.Info(
			"output audio buffer started",
			zap.Any("item", event.Param),
			zap.String("event_id", event.EventId),
		)
	case pkg.ServerEventTypeOutputAudioBufferStopped:
		a.logger.Info(
			"output audio buffer stopped",
			zap.Any("item", event.Param),
			zap.String("event_id", event.EventId),
		)
	case pkg.ServerEventTypeOutputAudioBufferCleared:
		a.logger.Info(
			"output audio buffer cleared",
			zap.Any("item", event.Param),
			zap.String("event_id", event.EventId),
		)
	case pkg.ServerEventTypeResponseCreated:
		a.logger.Info(
			"response created",
			zap.Any("item", event.Param),
			zap.String("event_id", event.EventId),
		)
	case pkg.ServerEventTypeResponseDone:
	case pkg.ServerEventTypeResponseOutputItemAdded:
	case pkg.ServerEventTypeResponseOutputItemDone:
	case pkg.ServerEventTypeResponseContentPartAdded:
	case pkg.ServerEventTypeResponseContentPartDone:
	case pkg.ServerEventTypeResponseOutputTextDelta:
	case pkg.ServerEventTypeResponseOutputTextDone:
	case pkg.ServerEventTypeResponseOutputAudioTranscriptDelta:
		a.logger.Info(
			"response output audio transcript delta",
			zap.Any("item", event.Param),
			zap.String("event_id", event.EventId),
		)
	case pkg.ServerEventTypeResponseOutputAudioTranscriptDone:
		a.logger.Info(
			"response output audio transcript done",
			zap.Any("item", event.Param),
			zap.String("event_id", event.EventId),
		)
	case pkg.ServerEventTypeResponseOutputAudioDelta:
	case pkg.ServerEventTypeResponseOutputAudioDone:
	case pkg.ServerEventTypeResponseFunctionCallArgumentsDelta:
	case pkg.ServerEventTypeResponseFunctionCallArgumentsDone:
	case pkg.ServerEventTypeResponseMCPCallArgumentsDelta:
	case pkg.ServerEventTypeResponseMCPCallArgumentsDone:
	case pkg.ServerEventTypeResponseMCPCallInProgress:
	case pkg.ServerEventTypeResponseMCPCallCompleted:
	case pkg.ServerEventTypeResponseMCPCallFailed:
	case pkg.ServerEventTypeMCPListToolsInProgress:
	case pkg.ServerEventTypeMCPListToolsCompleted:
	case pkg.ServerEventTypeMCPListToolsFailed:
	case pkg.ServerEventTypeRatelimitsUpdated:
	default:
		a.logger.Warn(
			"unknown event type received",
			zap.String("type", string(event.Type)),
			zap.String("event_id", event.EventId),
			zap.Any("event", event.Param),
		)
	}

}

type CLIState struct {
	mu         sync.Mutex
	userTrans  string
	coachTrans string
	turn       int // 0 for assistant, 1 for user
}

func NewCLIState() *CLIState {
	return &CLIState{
		userTrans:  "🗣️ User  - ",
		coachTrans: "🧑 Coach - ",
		turn:       0,
	}
}

func (s *CLIState) PipeEvent(event *pkg.ServerEvent) (resp string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	switch event.Type {
	case pkg.ServerEventTypeConversationItemInputAudioTranscriptionDelta:
		delta := event.Param.(*pkg.ServerEventParamConversationItemInputAudioTranscriptionDelta).Delta
		if s.turn == 1 {
			if s.userTrans == "" {
				resp = delta
			} else {
				resp = s.userTrans + delta
				s.userTrans = ""
			}
		} else {
			s.userTrans += delta
		}
	case pkg.ServerEventTypeResponseOutputAudioTranscriptDelta:
		delta := event.Param.(*pkg.ServerEventParamResponseOutputAudioTranscriptDelta).Delta
		if s.turn == 0 {
			if s.coachTrans == "" {
				resp = delta
			} else {
				resp = s.coachTrans + delta
				s.coachTrans = ""
			}
		} else {
			s.coachTrans += delta
		}
	case pkg.ServerEventTypeConversationItemInputAudioTranscriptionCompleted:
		if s.turn == 1 {
			resp = s.userTrans + "\n\n"
			s.userTrans = "🗣️ User  - "
			s.turn = 0
		} else {
			s.userTrans = "🗣️ User  - "
		}
	case pkg.ServerEventTypeResponseOutputAudioTranscriptDone:
		if s.turn == 0 {
			resp = s.coachTrans + "\n\n"
			s.coachTrans = "🧑 Coach - "
			s.turn = 1
		} else {
			s.coachTrans = "🧑 Coach - "
		}
	}
	return resp
}

func (a *CLIAgent) printEventHelper(title string, event *pkg.ServerEvent) {
	if err := a.printer.Writeln(title, 0); err != nil {
		a.logger.Error("printing event title", err)
	}
	if event != nil {
		yamlBytes, err := event.MarshalYAML()
		if err != nil {
			a.logger.Error("marshaling event to yaml", err)
			return
		}
		if err := a.printer.Write(string(yamlBytes), 1); err != nil {
			a.logger.Error("printing event", err)
		}
	}
	if err := a.printer.Writeln("", 0); err != nil {
		a.logger.Error("printing newline after event", err)
	}
}

func (a *CLIAgent) printHelper(s string, indent int) {
	if err := a.printer.Write(s, indent); err != nil {
		a.logger.Error("printing", err)
	}
}
