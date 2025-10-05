package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/bt-bridge/openai-realtime/shared"
	"github.com/bytedance/sonic"
	"github.com/goccy/go-yaml"
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
	sessionOutputSpeed             float64 = 0.9
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
	sessionOutputVoice     string = "cedar"
	sessionMaxOutputTokens int64  = 1024
)

// Realtime Client Config
const (
	// Opus Encoded Audio Input
	realtimeInputChannels int = 1     // mono
	realtimeInputRate     int = 48000 // 48 kHz
	realtimeInputFrameMs  int = 20    // 20 ms
)

func loadApiKey(logger shared.LoggerAdapter) string {
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
	return apiKey
}

func loadSessionConfig(logger shared.LoggerAdapter) []byte {
	cfg := realtime.RealtimeSessionCreateRequestParam{
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
	bytes, err := cfg.MarshalJSON()
	if err != nil {
		logger.Error("failed to marshal session config", err)
		os.Exit(1)
	}
	var cfgMap map[string]any
	if err = sonic.Unmarshal(bytes, &cfgMap); err != nil {
		logger.Error("failed to re-unmarshal session config", err)
		os.Exit(1)
	}
	yamlBytes, err := yaml.Marshal(cfgMap)
	if err != nil {
		logger.Error("failed to marshal session config to yaml", err)
		os.Exit(1)
	}
	boldPrintln("📋 Session Config")
	fmt.Println("\n├─  " + strings.ReplaceAll(string(yamlBytes), "\n", "\n│   ") + "\n")
	return bytes
}

func boldPrintln(a ...any) {
	fmt.Print("\033[1m")
	fmt.Println(a...)
	fmt.Print("\033[0m")
}

func main() {
	// Initialize logger
	logger := shared.NewFileLogger(
		logFileAddress, logFileMaxSize, logFileMaxBackups, logFileMaxAge, logFileCompress,
	).With(
		zap.String("component", "cli"),
		zap.String("version", shared.Version),
	)
	apiKey := loadApiKey(logger)
	session := loadSessionConfig(logger)
	_ = apiKey
	_ = session

}
