package kafka

import (
	"fmt"
	"os"

	"github.com/moov-io/base/log"
	"github.com/moov-io/base/strx"

	"github.com/IBM/sarama"
)

// SaramaLogger implements the sarama.StdLogger interface with a moov-io/base logger
//
// All messages are logged at a debug level.
type SaramaLogger struct {
	logger log.Logger
}

func NewSaramaLogger(logger log.Logger) *SaramaLogger {
	return &SaramaLogger{
		logger: logger,
	}
}

var _ sarama.StdLogger = (&SaramaLogger{})

func (l *SaramaLogger) Print(v ...interface{}) {
	l.logger.Debug().Log(fmt.Sprint(v...))
}

func (l *SaramaLogger) Printf(format string, v ...interface{}) {
	l.logger.Debug().Logf(format, v...)

}

func (l *SaramaLogger) Println(v ...interface{}) {
	l.Print(v...)
}

// EnableSaramaDebugLogging overrides the default sarama logger (which discards everything)
// with the provided moov-io/base logger.
//
// All messages are logged at a debug level.
func EnableSaramaDebugLogging(logger log.Logger) {
	if strx.Yes(os.Getenv("SARAMA_DEBUG_LOGGING")) {
		sarama.Logger = NewSaramaLogger(logger)
	}
}
