package chart

import (
	"fmt"
	"io"
	"os"
	"time"
)

var (
	_ Logger = (*StdoutLogger)(nil)
)

// NewLogger returns a new logger.
func NewLogger(options ...LoggerOption) Logger {
	stl := &StdoutLogger{
		TimeFormat: time.RFC3339Nano,
		Stdout:     os.Stdout,
		Stderr:     os.Stderr,
	}
	for _, option := range options {
		option(stl)
	}
	return stl
}

// Logger is a type that implements the logging interface.
type Logger interface {
	Info(...interface{})
	Infof(string, ...interface{})
	Debug(...interface{})
	Debugf(string, ...interface{})
	Err(error)
	FatalErr(error)
	Error(...interface{})
	Errorf(string, ...interface{})
}

// Info logs an info message if the logger is set.
func Info(log Logger, arguments ...interface{}) {
	if log == nil {
		return
	}
	log.Info(arguments...)
}

// Infof logs an info message if the logger is set.
func Infof(log Logger, format string, arguments ...interface{}) {
	if log == nil {
		return
	}
	log.Infof(format, arguments...)
}

// Debug logs an debug message if the logger is set.
func Debug(log Logger, arguments ...interface{}) {
	if log == nil {
		return
	}
	log.Debug(arguments...)
}

// Debugf logs an debug message if the logger is set.
func Debugf(log Logger, format string, arguments ...interface{}) {
	if log == nil {
		return
	}
	log.Debugf(format, arguments...)
}

// LoggerOption mutates a stdout logger.
type LoggerOption = func(*StdoutLogger)

//OptLoggerStdout sets the Stdout writer.
func OptLoggerStdout(wr io.Writer) LoggerOption {
	return func(stl *StdoutLogger) {
		stl.Stdout = wr
	}
}

// OptLoggerStderr sets the Stdout writer.
func OptLoggerStderr(wr io.Writer) LoggerOption {
	return func(stl *StdoutLogger) {
		stl.Stderr = wr
	}
}

// StdoutLogger is a basic logger.
type StdoutLogger struct {
	TimeFormat string
	Stdout     io.Writer
	Stderr     io.Writer
}

// Info writes an info message.
func (l *StdoutLogger) Info(arguments ...interface{}) {
	l.Println(append([]interface{}{"[INFO]"}, arguments...)...)
}

// Infof writes an info message.
func (l *StdoutLogger) Infof(format string, arguments ...interface{}) {
	l.Println(append([]interface{}{"[INFO]"}, fmt.Sprintf(format, arguments...))...)
}

// Debug writes an debug message.
func (l *StdoutLogger) Debug(arguments ...interface{}) {
	l.Println(append([]interface{}{"[DEBUG]"}, arguments...)...)
}

// Debugf writes an debug message.
func (l *StdoutLogger) Debugf(format string, arguments ...interface{}) {
	l.Println(append([]interface{}{"[DEBUG]"}, fmt.Sprintf(format, arguments...))...)
}

// Error writes an error message.
func (l *StdoutLogger) Error(arguments ...interface{}) {
	l.Println(append([]interface{}{"[ERROR]"}, arguments...)...)
}

// Errorf writes an error message.
func (l *StdoutLogger) Errorf(format string, arguments ...interface{}) {
	l.Println(append([]interface{}{"[ERROR]"}, fmt.Sprintf(format, arguments...))...)
}

// Err writes an error message.
func (l *StdoutLogger) Err(err error) {
	if err != nil {
		l.Println(append([]interface{}{"[ERROR]"}, err.Error())...)
	}
}

// FatalErr writes an error message and exits.
func (l *StdoutLogger) FatalErr(err error) {
	if err != nil {
		l.Println(append([]interface{}{"[FATAL]"}, err.Error())...)
		os.Exit(1)
	}
}

// Println prints a new message.
func (l *StdoutLogger) Println(arguments ...interface{}) {
	fmt.Fprintln(l.Stdout, append([]interface{}{time.Now().UTC().Format(l.TimeFormat)}, arguments...)...)
}

// Errorln prints a new message.
func (l *StdoutLogger) Errorln(arguments ...interface{}) {
	fmt.Fprintln(l.Stderr, append([]interface{}{time.Now().UTC().Format(l.TimeFormat)}, arguments...)...)
}
