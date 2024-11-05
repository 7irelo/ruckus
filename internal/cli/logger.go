package cli

import (
	"os"
	"time"

	"github.com/rs/zerolog"
)

func buildLogger(human bool) zerolog.Logger {
	zerolog.TimeFieldFormat = time.RFC3339Nano

	if human {
		writer := zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}
		return zerolog.New(writer).With().Timestamp().Logger()
	}

	return zerolog.New(os.Stdout).With().Timestamp().Logger()
}
