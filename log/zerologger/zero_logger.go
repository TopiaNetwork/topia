package zerologger

import (
	"io"

	"github.com/rs/zerolog"

	logcomm "github.com/TopiaNetwork/topia/log/common"
)

type ZeroLogger struct {
   log *zerolog.Logger
}

func NewLogger(level zerolog.Level, w io.Writer) *ZeroLogger {
	zl := zerolog.New(w).Level(level).With().Timestamp().Logger()

	return &ZeroLogger {
		&zl,
	}
}

func (zl *ZeroLogger) Trace(msg string) {
	zl.log.Trace().Msg(msg)
}

func (zl *ZeroLogger) Tracef(format string, args ...interface{}) {
	zl.log.Trace().Msgf(format, args...)
}

func (zl *ZeroLogger) Debug(msg string) {
	zl.log.Debug().Msg(msg)
}

func (zl *ZeroLogger) Debugf(format string, args ...interface{}) {
	zl.log.Debug().Msgf(format, args...)
}

func (zl *ZeroLogger) Info(msg string) {
	zl.log.Info().Msg(msg)
}

func (zl *ZeroLogger) Infof(format string, args ...interface{}) {
	zl.log.Info().Msgf(format, args...)
}

func (zl *ZeroLogger) Warn(msg string) {
	zl.log.Warn().Msg(msg)
}

func (zl *ZeroLogger) Warnf(format string, args ...interface{}) {
	zl.log.Warn().Msgf(format, args...)
}

func (zl *ZeroLogger) Error(msg string) {
	zl.log.Error().Msg(msg)
}

func (zl *ZeroLogger) Errorf(format string, args ...interface{}) {
	zl.log.Error().Msgf(format, args...)
}

func (zl *ZeroLogger) Fatal(msg string) {
	zl.log.Fatal().Msg(msg)
}

func (zl *ZeroLogger) Fatalf(format string, args ...interface{}) {
	zl.log.Fatal().Msgf(format, args...)
}

func (zl *ZeroLogger) Panic(msg string) {
	zl.log.Panic().Msg(msg)
}

func (zl *ZeroLogger) Panicf(format string, args ...interface{}) {
	zl.log.Panic().Msgf(format, args...)
}

func (zl *ZeroLogger) UpdateLoggerLevel(level logcomm.LogLevel) {
	zxNew := zl.log.Level(logcomm.ToZerologLevel(level))
	zl.log = &zxNew
}

func (zl *ZeroLogger) CreateModuleLogger(level zerolog.Level, module string) *ZeroLogger {
	mLog := zl.log.With().Str("module",module).Logger().Level(level)

	return &ZeroLogger{
		&mLog,
	}
}



