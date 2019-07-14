package log

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

var testConfigs = []struct {
	cfg Configuration
	ok  bool
}{
	{
		cfg: Configuration{
			Level:    "info",
			Encoding: "console",
		},
		ok: true,
	},
}

func TestNewTestLogger(t *testing.T) {
	log := NewTestLogger(t)
	require.NotNil(t, log)
}

func TestConfigurations(t *testing.T) {
	for _, tt := range testConfigs {
		log, err := tt.cfg.NewLogger()
		if tt.ok {
			require.NoError(t, err, fmt.Sprintf("Configuration %#v should not throw error", tt.cfg))
			require.NotNil(t, log)
		} else {
			require.Error(t, err, fmt.Sprintf("Configuration %#v should throw error", tt.cfg))
		}
	}
}
