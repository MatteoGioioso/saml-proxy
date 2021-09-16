package sharedKernel

import (
	"log"
	"os"
)

type Logger interface {
	Info(message string)
	Failure(err error)
}

type DefaultLogger struct {
	logger *log.Logger
}

func NewDefaultLogger() *DefaultLogger {
	return &DefaultLogger{logger: log.New(os.Stdout, "[Saml Proxy] ", log.Ltime)}
}

func (d DefaultLogger) Info(message string) {
	d.logger.Println("INFO", message)
}

func (d DefaultLogger) Failure(err error) {
	d.logger.Println("ERROR", err.Error())
}

