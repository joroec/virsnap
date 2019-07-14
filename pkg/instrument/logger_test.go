package instrument

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewLogger(t *testing.T) {
	log := NewTestLogger(t)
	require.NotNil(t, log)
}
