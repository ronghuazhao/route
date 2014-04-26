package logger

import (
	"fmt"
	"github.com/foize/go.sgr"
	"sync"
)

const (
	Console = iota
)

type Logger struct {
	mutex    sync.RWMutex
	handler  int
}

func NewLogger(name string, handler int) *Logger {

	logger := &Logger{}

	if handler == Console {
		logger.handler = Console
		logger.Log("internal", fmt.Sprintf("%s.status", name), "using console output", "[fg-blue]")
	} else {
		panic("Invalid logging handler")
	}

	return logger
}

func (logger *Logger) Log(topic, key, value string, formatting interface{}) {
	logger.mutex.RLock()
	defer logger.mutex.RUnlock()

	if logger.handler == Console {
		var message string

		if formatting != nil {
			message = fmt.Sprintf("%s> %s: %s", formatting, topic, value)
		} else {
			message = fmt.Sprintf("> %s: %s", topic, value)
		}

		message = sgr.MustParseln(message)
		fmt.Print(message)
	}
}
