package pine

import (
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
