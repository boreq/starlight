package utils

import (
	"log"
	"os"
)

// Logger defines methods used for logging in a normal mode and a debug mode.
// Debug mode log messages are displayed only if a proper environment variable
// with the name stored in DebugEnvVar is set.
type Logger interface {
	Print(...interface{})
	Printf(string, ...interface{})
	Debug(...interface{})
	Debugf(string, ...interface{})
}

type logger struct {
	logger *log.Logger
}

func (l *logger) Print(v ...interface{}) {
	l.logger.Print(v...)
}

func (l *logger) Printf(format string, v ...interface{}) {
	l.logger.Printf(format, v...)
}

func (l *logger) Debug(v ...interface{}) {
	if debug {
		l.logger.Print(v...)
	}
}

func (l *logger) Debugf(format string, v ...interface{}) {
	if debug {
		l.logger.Printf(format, v...)
	}
}

// The name of the environment variable which enables displaying debug level log
// messages. To do that this environment variable can be set to any value but
// an empty string.
const DebugEnvVar = "STARLIGHTDEBUG"

var debug bool
var loggers map[string]Logger

func init() {
	loggers = make(map[string]Logger)
	debug = (os.Getenv(DebugEnvVar) != "")
}

// GetLogger creates a new logger or returns an already existing logger created
// with the given name using this method.
func GetLogger(name string) Logger {
	if _, ok := loggers[name]; !ok {
		loggers[name] = &logger{log.New(os.Stdout, name+": ", 0)}
	}
	return loggers[name]
}
