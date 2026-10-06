package logger

import (
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Init configures the global zerolog logger: JSON at info level in prod,
// colored console output at debug level otherwise.
func Init(prod bool) {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.LevelColors[zerolog.DebugLevel] = 33 // colorYellow
	if prod {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
		return
	}
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	out := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.DateTime}
	out.FormatMessage = func(i any) string {
		return fmt.Sprintf("%+v", i)
	}
	log.Logger = log.Output(out)
}
