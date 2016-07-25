package main

import (
	"fmt"
	"io"
	"os"
	"time"
)

var verbose bool
var log io.Writer

func openLog() {
	cdWorkDir()
	logFile, err := os.OpenFile("log", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		errorExit("Unable to open logfile: %s", err)
	}

	restoreCwd()

	if verbose {
		log = io.MultiWriter(logFile, os.Stderr)
	} else {
		log = logFile
	}
}

func logf(format string, args ...interface{}) {
	fmt.Fprintf(log, "\n%s ", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(log, format+"\n", args...)
}
