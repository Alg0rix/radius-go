package runtime

import (
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/your-org/radius-go/internal/config"
)

func NewLogger(cfg config.Config) zerolog.Logger {
	zerolog.TimeFieldFormat = time.RFC3339

	if cfg.LogFormat == "json" {
		return zerolog.New(os.Stdout).With().Timestamp().Logger()
	}

	return zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}).With().Timestamp().Logger()
}