package utils

import (
	"log"
	"os"
)

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

var debug bool
var loggers map[string]Logger

func init() {
	loggers = make(map[string]Logger)
	debug = (os.Getenv("NETBLOGDEBUG") != "")
}

func GetLogger(name string) Logger {
	if _, ok := loggers[name]; !ok {
		loggers[name] = &logger{log.New(os.Stdout, name+": ", 0)}
	}
	return loggers[name]
}
