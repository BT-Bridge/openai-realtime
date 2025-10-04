package main

import (
	"fmt"
	"os"

	"github.com/bt-bridge/openai-realtime/shared"
	"go.uber.org/zap"
)

const (
	logFileAddress string = "cli/cli.log"
	logFileMaxSize int    = 10 * 1 << 20 // 10 MB
	logFileMaxBackups int = 2            // keep 2 backups
	logFileMaxAge     int = 3            // max age 3 days
	logFileCompress   bool = false        // no compression
)

const (
	apiKeyEnvKey string = "OPENAI_API_KEY"
)

func loadApiKey(logger shared.LoggerAdapter) string {
	apiKey, err := shared.Getenv(shared.GetenvString, apiKeyEnvKey, false, "")
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

func main() {
	// Initialize logger
	logger := shared.NewFileLogger(
		logFileAddress, logFileMaxSize, logFileMaxBackups, logFileMaxAge, logFileCompress,
	).With(
		zap.String("component", "cli"), 
		zap.String("version", shared.Version),
	)
	apiKey := loadApiKey(logger)
	_ = apiKey
}
