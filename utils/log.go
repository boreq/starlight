package utils

import (
	"log"
	"os"
)

var loggers map[string]*log.Logger

func init() {
	loggers = make(map[string]*log.Logger)
}

func Logger(name string) *log.Logger {
	if _, ok := loggers[name]; !ok {
		loggers[name] = log.New(os.Stdout, name+": ", 0)
	}
	return loggers[name]
}
