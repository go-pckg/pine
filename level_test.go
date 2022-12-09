package pine

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLevelParse(t *testing.T) {
	check := func(text string, want Level) {
		lvl, err := ParseLevel(text)
		assert.NoError(t, err)
		assert.Equal(t, want, lvl)
	}

	check("fatal", FatalLevel)
	check("panic", PanicLevel)
	check("error", ErrorLevel)
	check("warn", WarnLevel)
	check("info", InfoLevel)
	check("debug", DebugLevel)
	check("trace", TraceLevel)
	check("", DisabledLevel)
}

func TestEnvLevel(t *testing.T) {
	os.Setenv("PINE_LEVEL", "trace")
	lgr := New()
	assert.Equal(t, TraceLevel, *lgr.level.Value)
}
