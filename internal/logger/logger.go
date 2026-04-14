package logger

import (
	"fmt"
	"log"
	"os"
)

// Logger defines the logging interface.
type Logger interface {
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	Debug(msg string, args ...interface{})
}

// StdLogger implements Logger using standard library logging.
type StdLogger struct {
	infoLog  *log.Logger
	warnLog  *log.Logger
	errorLog *log.Logger
	debugLog *log.Logger
	debug    bool
}

// NewStdLogger creates a new standard logger.
func NewStdLogger() *StdLogger {
	return &StdLogger{
		infoLog:  log.New(os.Stdout, "[INFO] ", log.Ltime),
		warnLog:  log.New(os.Stdout, "[WARN] ", log.Ltime),
		errorLog: log.New(os.Stderr, "[ERROR] ", log.Ltime),
		debugLog: log.New(os.Stdout, "[DEBUG] ", log.Ltime),
		debug:    false,
	}
}

// NewStdLoggerWithDebug creates a new standard logger with debug enabled.
func NewStdLoggerWithDebug(debug bool) *StdLogger {
	return &StdLogger{
		infoLog:  log.New(os.Stdout, "[INFO] ", log.Ltime),
		warnLog:  log.New(os.Stdout, "[WARN] ", log.Ltime),
		errorLog: log.New(os.Stderr, "[ERROR] ", log.Ltime),
		debugLog: log.New(os.Stdout, "[DEBUG] ", log.Ltime),
		debug:    debug,
	}
}

// Info logs an info level message.
func (l *StdLogger) Info(msg string, args ...interface{}) {
	formatted := fmt.Sprintf(msg, args...)
	l.infoLog.Println(formatted)
}

// Warn logs a warn level message.
func (l *StdLogger) Warn(msg string, args ...interface{}) {
	formatted := fmt.Sprintf(msg, args...)
	l.warnLog.Println(formatted)
}

// Error logs an error level message.
func (l *StdLogger) Error(msg string, args ...interface{}) {
	formatted := fmt.Sprintf(msg, args...)
	l.errorLog.Println(formatted)
}

// Debug logs a debug level message (only if debug is enabled).
func (l *StdLogger) Debug(msg string, args ...interface{}) {
	if !l.debug {
		return
	}
	formatted := fmt.Sprintf(msg, args...)
	l.debugLog.Println(formatted)
}
