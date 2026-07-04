package logger

import (
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func Init() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.LevelColors[zerolog.DebugLevel] = 33 // colorYellow
	if os.Getenv("ENVIRONMENT") == "prod" {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		out := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.DateTime}
		out.FormatMessage = func(i any) string {
			return fmt.Sprintf("%+v", i)
		}
		log.Logger = log.Output(out)
	}
}
