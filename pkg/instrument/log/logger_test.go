package log

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	validEncodings = []string{"json", "console"}
	validLevels    = []string{"debug", "info", "warn", "error"}

	invalidStrings = []string{"test", "asdo1293", "😍", "👾 🙇 💁 🙅 🙆 🙋 🙎 🙍", "﷽"}
)

func TestNewTestLogger(t *testing.T) {
	log := NewTestLogger(t)
	require.NotNil(t, log)
}

func TestConfigurations(t *testing.T) {
	t.Run("TestValid", func(t *testing.T) {
		for _, encoding := range validEncodings {
			for _, level := range validLevels {
				cfg := Configuration{
					Level:    level,
					Encoding: encoding,
				}
				log, err := cfg.NewLogger()
				require.NoError(t, err, fmt.Sprintf("Configuration %#v should not throw error", cfg))
				require.NotNil(t, log)
			}
		}
	})

	t.Run("TestInvalid", func(t *testing.T) {
		for _, encoding := range validEncodings {
			for _, level := range invalidStrings {
				cfg := Configuration{
					Level:    level,
					Encoding: encoding,
				}
				log, err := cfg.NewLogger()
				require.Error(t, err, fmt.Sprintf("Configuration %#v should throw error", cfg))
				require.Nil(t, log)
			}
		}

		for _, encoding := range invalidStrings {
			for _, level := range validLevels {
				cfg := Configuration{
					Level:    level,
					Encoding: encoding,
				}
				log, err := cfg.NewLogger()
				require.Error(t, err, fmt.Sprintf("Configuration %#v should throw error", cfg))
				require.Nil(t, log)
			}
		}
	})
}
