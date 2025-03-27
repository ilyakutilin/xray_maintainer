package main

import (
	"io"
	"log"
	"os"
)

type Logger struct {
	Debug *log.Logger
	Info  *log.Logger
	Error *log.Logger
}

func GetLogger(debug bool) *Logger {
	var debugWriter io.Writer

	if debug {
		debugWriter = os.Stdout
	} else {
		debugWriter = io.Discard
	}

	return &Logger{
		Debug: log.New(debugWriter, "DEBUG\t", log.Ldate|log.Ltime),
		Info:  log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime),
		Error: log.New(os.Stderr, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile),
	}
}
