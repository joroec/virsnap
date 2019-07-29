// Copyright (c) 2019 The virnsnap authors. See file "AUTHORS".
// Licensed under the MIT License. You have obtained a copy of the License at
// the "LICENSE" file in this repository.

// Package log provides logging directives.
package log

// Logger interface provides an abstraction over different loggers
// that can be used with the application.
type Logger interface {
	Debug(...interface{})
	Debugf(string, ...interface{})
	Info(...interface{})
	Infof(string, ...interface{})
	Warn(...interface{})
	Warnf(string, ...interface{})
	Error(...interface{})
	Errorf(string, ...interface{})
}
