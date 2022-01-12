package common

import (
	"github.com/rs/zerolog"
)

type LogLevel uint8

// NoLevel means it should be ignored
const (
	NoLevel LogLevel = iota
	TraceLevel
	DebugLevel
	InfoLevel
	WarnLevel
	ErrorLevel
	FatalLevel
	PanicLevel
	maxLogLevel
)

const LogLevelCount = int(maxLogLevel)

var levelMapping = []zerolog.Level{
	NoLevel:    zerolog.NoLevel,
	TraceLevel: zerolog.TraceLevel,
	InfoLevel:  zerolog.InfoLevel,
	DebugLevel: zerolog.DebugLevel,
	WarnLevel:  zerolog.WarnLevel,
	ErrorLevel: zerolog.ErrorLevel,
	FatalLevel: zerolog.FatalLevel,
	PanicLevel: zerolog.PanicLevel,
}

func ToZerologLevel(level LogLevel) zerolog.Level {
	return levelMapping[level]
}
