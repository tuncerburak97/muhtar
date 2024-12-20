package logger

import (
	"os"

	"github.com/rs/zerolog"
)

var log zerolog.Logger

func Init(level string) {
	logLevel, err := zerolog.ParseLevel(level)
	if err != nil {
		logLevel = zerolog.InfoLevel
	}

	zerolog.SetGlobalLevel(logLevel)
	log = zerolog.New(os.Stdout).With().Timestamp().Logger()
}

func GetLogger() *zerolog.Logger {
	return &log
}
