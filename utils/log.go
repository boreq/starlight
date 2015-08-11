package utils

import "log"

var loggers map[string]log.Logger

func init() {
	loggers := make(map[string]log.Logger)
}

func Logger(name string) log.Logger {
}
