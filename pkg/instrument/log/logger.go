// Copyright (c) 2019 The virnsnap authors. See file "AUTHORS".
// Licensed under the MIT License. You have obtained a copy of the License at
// the "LICENSE" file in this repository.

// Package log provides logging directives.
package log

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// Configuration defines config parameters for building a Logger.
type Configuration struct {
	Level    string
	Fields   map[string]interface{}
	Encoding string
}

// NewTestLogger returns a new logger for testing purposes. This logger is
// configured with  the zap DevelopmentConfig and annotations disabled.
func NewTestLogger(t *testing.T) *zap.Logger {
	zc := zap.NewDevelopmentConfig()
	zc.DisableCaller = true
	zl, err := zc.Build()
	require.NoError(t, err)
	return zl
}

// NewLogger returns a new logger with a production-ready config.
func (cfg Configuration) NewLogger() (*zap.Logger, error) {
	zc := zap.Config{
		Level:             zap.NewAtomicLevelAt(zap.InfoLevel),
		Development:       false,
		DisableCaller:     true,
		DisableStacktrace: true,
		Sampling: &zap.SamplingConfig{
			Initial:    100,
			Thereafter: 100,
		},
		Encoding:         "json",
		EncoderConfig:    zap.NewProductionEncoderConfig(),
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stdout"},
		InitialFields:    cfg.Fields,
	}

	// Set log level
	if len(cfg.Level) != 0 {
		var parsedLevel zap.AtomicLevel
		if err := parsedLevel.UnmarshalText([]byte(cfg.Level)); err != nil {
			return nil, fmt.Errorf("unable to parse log level %s: %v", cfg.Level, err)
		}
		zc.Level = parsedLevel
	}

	// Set encoding
	if len(cfg.Encoding) != 0 {
		switch cfg.Encoding {
		case "console":
			zc.Encoding = "console"
		case "json":
			zc.Encoding = "json"
		default:
			return nil, fmt.Errorf("unable to set Encoding: invalid value '%s'", cfg.Encoding)
		}
	}

	return zc.Build()
}

// NewDefaultLogger returns an opinionated, sugared logger.
func NewDefaultLogger() (*zap.SugaredLogger, error) {
	cfg := Configuration{
		Level:    "info",
		Encoding: "console",
	}

	log, err := cfg.NewLogger()
	if err != nil {
		return nil, err
	}

	return log.Sugar(), nil
}
