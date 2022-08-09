package pine

import (
	"fmt"
	"strings"
)

type Level int8

const (
	DisabledLevel Level = iota
	PanicLevel
	FatalLevel
	ErrorLevel
	WarnLevel
	InfoLevel
	DebugLevel
	TraceLevel
)

func (l Level) String() string {
	switch l {
	case TraceLevel:
		return "trace"
	case DebugLevel:
		return "debug"
	case InfoLevel:
		return "info"
	case WarnLevel:
		return "warn"
	case ErrorLevel:
		return "error"
	case PanicLevel:
		return "panic"
	case FatalLevel:
		return "fatal"
	default:
		return ""
	}
}

func (l Level) Value() Level {
	return l
}

func (l Level) MarshalText() ([]byte, error) {
	return []byte(l.String()), nil
}

func (l *Level) UnmarshalText(text []byte) error {
	if !l.unmarshalText(text) {
		return fmt.Errorf("invalid level: %q", text)
	}
	return nil
}

func (l *Level) unmarshalText(text []byte) bool {
	switch strings.ToLower(string(text)) {
	case "trace":
		*l = TraceLevel
	case "debug":
		*l = DebugLevel
	case "info":
		*l = InfoLevel
	case "warn":
		*l = WarnLevel
	case "error":
		*l = ErrorLevel
	case "panic":
		*l = PanicLevel
	case "fatal":
		*l = FatalLevel
	case "":
		*l = DisabledLevel
	default:
		return false
	}
	return true
}

func ParseLevel(text string) (Level, error) {
	var level Level
	err := level.UnmarshalText([]byte(text))
	return level, err
}
