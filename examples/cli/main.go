package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/bt-bridge/openai-realtime/agents"
	"github.com/bt-bridge/openai-realtime/shared"
	"github.com/openai/openai-go/v3/packages/param"
	"github.com/openai/openai-go/v3/realtime"
	"go.uber.org/zap"
)

// Environment variable keys
const (
	envKeyApiKey string = "OPENAI_API_KEY"
)

// Log file configuration
const (
	logFileAddress    string = "cli/cli.log"
	logFileMaxSize    int    = 10 * 1 << 20 // 10 MB
	logFileMaxBackups int    = 2            // keep 2 backups
	logFileMaxAge     int    = 3            // max age 3 days
	logFileCompress   bool   = false        // no compression
)

// Greeting message
const greeting string = `
Introduce yourself as user's business coach (بیزینس کوچ).
Say you are glad to assist in today's session.
Speak in Persian.
`

// Agent configuration
const (
	agentPrinterIndentString string = "│  "
)

// Session Config (4 October 2025)
const (
	sessionInstructions string = "You are a natural, native AI assistant that speaks clearly and conversationally."
	// gpt-realtime
	// / gpt-realtime-2025-08-28
	// / gpt-4o-realtime-preview
	// / gpt-4o-realtime-preview-2024-10-01
	// / gpt-4o-realtime-preview-2024-12-17
	// / gpt-4o-realtime-preview-2025-06-03
	// / gpt-4o-mini-realtime-preview
	// / gpt-4o-mini-realtime-preview-2024-12-17
	sessionModel          string = "gpt-realtime"
	sessionVADEagerness   string = "low"        // low, medium, high
	sessionNoiseReduction string = "near_field" // near_field, far_field
	// The language of the input audio. Supplying the input language in
	// [ISO-639-1](https://en.wikipedia.org/wiki/List_of_ISO_639-1_codes) (e.g. `en`)
	sessionInputLanguage            string = "fa"
	sessionInputTranscriptionPrompt string = "expect words related to web technologies"
	// whisper-1
	// / gpt-4o-transcribe-latest
	// / gpt-4o-mini-transcribe
	// / gpt-4o-transcribe
	sessionInputTranscriptionModel string  = "whisper-1"
	sessionOutputSpeed             float64 = 1.1
	// alloy
	// / ash
	// / ballad
	// / coral
	// / echo
	// / sage
	// / shimmer
	// / verse
	// / marin
	// / cedar
	sessionOutputVoice     string = "ash"
	sessionMaxOutputTokens int64  = 1024
)

func main() {
	// Initialize logger
	logger := shared.NewFileLogger(
		logFileAddress, logFileMaxSize, logFileMaxBackups, logFileMaxAge, logFileCompress,
	).With(
		zap.String("component", "cli"),
		zap.String("version", shared.Version),
	)

	// Loading API Key
	apiKey, err := shared.Getenv(shared.GetenvString, envKeyApiKey, false, "")
	if err != nil {
		logger.Error("OPENAI_API_KEY environment variable", err)
		os.Exit(1)
	}
	if apiKey == "" {
		fmt.Print("Enter your OpenAI API key: ")
		var input string
		_, err := fmt.Scanln(&input)
		if err != nil {
			if err.Error() == "unexpected newline" {
				logger.Error("no API key provided", nil)
				os.Exit(1)
			}
			logger.Error("failed to read API key from stdin", err)
			os.Exit(1)
		}
		apiKey = input
	}
	logger.Info(
		"using OpenAI API key",
		zap.String("apiKey", apiKey[:10]+"..."),
	)

	// Making Session Config
	session := &realtime.RealtimeSessionCreateRequestParam{
		Instructions: param.NewOpt(sessionInstructions),
		Model:        sessionModel,
		Audio: realtime.RealtimeAudioConfigParam{
			Input: realtime.RealtimeAudioConfigInputParam{
				TurnDetection: realtime.RealtimeAudioInputTurnDetectionUnionParam{
					OfSemanticVad: &realtime.RealtimeAudioInputTurnDetectionSemanticVadParam{
						CreateResponse:    param.NewOpt(true),
						InterruptResponse: param.NewOpt(true),
						Eagerness:         sessionVADEagerness,
					},
				},
				Format: realtime.RealtimeAudioFormatsUnionParam{
					OfAudioPCM: &realtime.RealtimeAudioFormatsAudioPCMParam{
						Rate: 24000,
						Type: "audio/pcm",
					},
				},
				NoiseReduction: realtime.RealtimeAudioConfigInputNoiseReductionParam{
					Type: realtime.NoiseReductionType(sessionNoiseReduction),
				},
				Transcription: realtime.AudioTranscriptionParam{
					Language: param.NewOpt(sessionInputLanguage),
					Prompt:   param.NewOpt(sessionInputTranscriptionPrompt),
					Model:    realtime.AudioTranscriptionModel(sessionInputTranscriptionModel),
				},
			},
			Output: realtime.RealtimeAudioConfigOutputParam{
				Speed: param.NewOpt(sessionOutputSpeed),
				Format: realtime.RealtimeAudioFormatsUnionParam{
					OfAudioPCM: &realtime.RealtimeAudioFormatsAudioPCMParam{
						Rate: 24000,
						Type: "audio/pcm",
					},
				},
				Voice: realtime.RealtimeAudioConfigOutputVoice(sessionOutputVoice),
			},
		},
		MaxOutputTokens: realtime.RealtimeSessionCreateRequestMaxOutputTokensUnionParam{
			OfInt: param.NewOpt(sessionMaxOutputTokens),
		},
	}

	// Loading Base URL
	baseUrl := shared.MustGetenv(
		shared.GetenvString,
		"OPENAI_BASE_URL",
		false,
		"https://api.openai.com/v1",
	)

	// Making Printer Hooks
	stdoutHook := shared.NewWriteCloser(os.Stdout)
	if stdoutHook == nil {
		logger.Error("creating stdout hook", nil)
		os.Exit(1)
	}
	printer, err := shared.NewPrinter(agentPrinterIndentString, stdoutHook)
	if err != nil {
		logger.Error("creating printer", err)
		os.Exit(1)
	}

	// Spawning CLI Agent
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	agent := new(agents.CLIAgent)
	err = agent.Spawn(ctx, logger, apiKey, session, greeting, printer, baseUrl)
	if err != nil {
		logger.Error("spawning CLI agent", err)
		os.Exit(1)
	}

	// Waiting for graceful shutdown or session end
	sig := make(chan os.Signal, 1)
	defer close(sig)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-agent.Done():
		logger.Info("session ended")
		return
	case <-sig:
		logger.Info("shutting down...")
		if err = agent.Close(); err != nil {
			logger.Error("closing CLI agent", err)
			os.Exit(1)
		}
		select {
		case <-agent.Done():
			logger.Info("graceful shutdown complete")
			return
		case <-sig:
			logger.Info("forcing shutdown")
			return
		}
	}
}
