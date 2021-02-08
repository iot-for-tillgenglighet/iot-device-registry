package logging

import (
	log "github.com/sirupsen/logrus"
)

//Logger interface that allows abstracting away the concrete logger implementation we are using
type Logger interface {
	//Fatal causes the application to terminate with the given error message
	Fatal(args ...interface{})
	//Fatalf causes the application to terminate with the given error message
	Fatalf(format string, args ...interface{})
	//Error logs a message at ERROR level on the default logger
	Error(args ...interface{})
	//Errorf logs a message at ERROR level on the default logger
	Errorf(format string, args ...interface{})
	//Infof logs a message at INFO level on the default logger
	Infof(format string, args ...interface{})
}

//NewLogger instantiates a new logger and returns it
func NewLogger() Logger {
	log.SetFormatter(&log.JSONFormatter{})
	return &logger{}
}

type logger struct {
}

func (l *logger) Error(args ...interface{}) {
	log.Error(args...)
}

func (l *logger) Errorf(format string, args ...interface{}) {
	log.Errorf(format, args...)
}

func (l *logger) Fatal(args ...interface{}) {
	log.Fatal(args...)
}

func (l *logger) Fatalf(format string, args ...interface{}) {
	log.Fatalf(format, args...)
}

func (l *logger) Infof(format string, args ...interface{}) {
	log.Infof(format, args...)
}
