// Copyright 2014 Regents of the University of Minnesota. All rights reserved.
// The University of Minnesota is an equal opportunity educator and employer.
// Use of this file is governed by a license found in the license.md file.

package logger

import (
	"fmt"
	"sync"

	"github.com/foize/go.sgr"
)

const (
	Console = iota
)

type Logger struct {
	mutex   sync.RWMutex
	handler int
}

func NewLogger(name string, handler int) *Logger {

	logger := &Logger{}

	if handler == Console {
		logger.handler = Console
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
