// Copyright (c) 2019 The virnsnap authors. See file "AUTHORS".
// Licensed under the MIT License. You have obtained a copy of the License at
// the "LICENSE" file in this repository.

// Package instrument provides functions to enable monitoring and logging the
// application.
package instrument

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// NewTestLogger returns a new logger for testing purposes. This logger is
// configured with  the zap DevelopmentConfig and annotations disabled.
func NewTestLogger(t *testing.T) *zap.Logger {
	zc := zap.NewDevelopmentConfig()
	zc.DisableCaller = true
	zl, err := zc.Build()
	require.NoError(t, err)
	return zl
}
