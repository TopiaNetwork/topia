package log

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog"

	logcomm "github.com/TopiaNetwork/topia/log/common"
	"github.com/TopiaNetwork/topia/log/zerologger"
)

type LogFormat uint8

const (
	TextFormat LogFormat = iota
	JSONFormat
)
const DefaultLogFormat = TextFormat

type LogOutput uint8

const (
	StdErrOutput LogOutput = iota
	FileLogOutput
	SysLogOutput
)
const DefaultLogOutput = StdErrOutput

const DefaultOutputParallelLimit = 5

type Logger interface {
	//log a message at a trace level
	Trace(msg string)
	//log a formatted message at a trace level
	Tracef(string, ...interface{})
	//log a message at an info level
	Info(msg string)
	//log a formatted message at an info level
	Infof(string, ...interface{})
	//log a message at a debug level
	Debug(msg string)
	//log a formatted message at a debug level
	Debugf(string, ...interface{})
	//log a message at a warn level
	Warn(msg string)
	//log a formatted message at a warn level
	Warnf(string, ...interface{})
	//log a message at an error level
	Error(msg string)
	//log a formatted message at an error level
	Errorf(string, ...interface{})
	//log a message at a fatal level
	Fatal(msg string)
	//log a formatted message at a info level
	Fatalf(string, ...interface{})
	//log a message at a panic level
	Panic(msg string)
	//log a formatted message at a panic level
	Panicf(string, ...interface{})

	//update the logger level
	UpdateLoggerLevel(level logcomm.LogLevel)
}

const TimestampFormat = "2006-01-02T15:04:05.000000000Z07:00"

func (l LogFormat) String() string {
	switch l {
	case TextFormat:
		return "text"
	case JSONFormat:
		return "json"
	}
	return string(l)
}

func (o LogOutput) String() string {
	switch o {
	case StdErrOutput:
		return "stderr"
	case FileLogOutput:
		return "filelog"
	case SysLogOutput:
		return "syslog"
	}
	return string(o)
}

var cwd string

func init() {
	var err error
	cwd, err = os.Getwd()
	if err != nil {
		cwd = ""
		fmt.Println("couldn't get current working directory: ", err.Error())
	}
}

func defaultPartsOrder() []string {
	return []string{
		zerolog.TimestampFieldName,
		zerolog.LevelFieldName,
		zerolog.MessageFieldName,
		zerolog.CallerFieldName,
	}
}

func formatCaller() zerolog.Formatter {
	return func(i interface{}) string {
		var c string
		if cc, ok := i.(string); ok {
			c = cc
		}
		if len(c) > 0 {
			if len(cwd) > 0 {
				c = strings.TrimPrefix(c, cwd)
				c = strings.TrimPrefix(c, "/")
			}
			c = "file=" + c
		}
		return c
	}
}

func newDefaultTextOutput(out io.Writer) io.Writer {
	return &zerolog.ConsoleWriter{
		Out:          out,
		NoColor:      true,
		TimeFormat:   TimestampFormat,
		PartsOrder:   defaultPartsOrder(),
		FormatCaller: formatCaller(),
	}
}

func selectFormatOutput(format LogFormat, output io.Writer) (io.Writer, error) {
	switch format {
	case TextFormat:
		return newDefaultTextOutput(output), nil
	case JSONFormat:
		return output, nil
	default:
		return nil, errors.New("unknown formatter " + format.String())
	}
}

func generateOutput(output LogOutput, param string) (io.Writer, error) {
	switch output {
	case StdErrOutput:
		return os.Stderr, nil
	case FileLogOutput:
		if param == "" {
			return nil, errors.New("generateOutput err: fileFullPath blank")
		}

		fileFullPath := param

		if _, err := os.Stat(fileFullPath); os.IsNotExist(err) {
			return os.OpenFile(fileFullPath, os.O_APPEND, 0666)
		} else {
			return os.Create(fileFullPath)
		}
	case SysLogOutput:
		binName := filepath.Base(os.Args[0])
		return  zerologger.ConnectSyslogByParam(param, binName)
	default:
		return nil, errors.New("unknown output type " + LogOutput.String(output))
	}
}

func CreateMainLogger(level logcomm.LogLevel, format LogFormat, output LogOutput, param string) (Logger, error) {
	outputW, err := generateOutput(output, param)
	if err != nil {
		return nil, err
	}

	wr, err := selectFormatOutput(format, outputW)
	if err != nil {
		return nil, err
	}

	return zerologger.NewLogger(logcomm.ToZerologLevel(level), wr), nil
}

func SetGlobalLevel(level logcomm.LogLevel) {
	zerolog.SetGlobalLevel(logcomm.ToZerologLevel(level))
}

func CreateModuleLogger(level logcomm.LogLevel, module string, l Logger) Logger {
	if zl, ok := l.(*zerologger.ZeroLogger); ok {
		return zl.CreateModuleLogger(logcomm.ToZerologLevel(level), module)
	}

	return nil
}




