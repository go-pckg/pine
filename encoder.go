package pine

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/pkg/errors"
)

var bufferPool = &sync.Pool{
	New: func() interface{} {
		return &bytes.Buffer{}
	},
}

type encoderConfig struct {
	UseColors        bool
	ForceQuote       bool
	QuoteEmptyFields bool
	DisableQuote     bool
	ReportCaller     bool
	DisableSorting   bool
	development      bool
}

type encoder interface {
	encodeEntry(ent *Entry, fields []Field) ([]byte, error)
	clone() encoder
}

type consoleEncoder struct {
	*encoderConfig
}

func newConsoleEncoder(config encoderConfig) consoleEncoder {
	return consoleEncoder{encoderConfig: &config}
}

func (l consoleEncoder) clone() encoder {
	return newConsoleEncoder(*l.encoderConfig)
}

func (l consoleEncoder) encodeEntry(ent *Entry, fields []Field) ([]byte, error) {
	lvl := consoleLevel(ent.level)
	switch ent.level {
	case TraceLevel:
		lvl = colorize(lvl, colorMagenta, l.UseColors)
	case DebugLevel:
		lvl = colorize(lvl, colorYellow, l.UseColors)
	case InfoLevel:
		lvl = colorize(lvl, colorGreen, l.UseColors)
	case WarnLevel:
		lvl = colorize(lvl, colorRed, l.UseColors)
	case ErrorLevel:
		lvl = colorize(colorize(lvl, colorRed, l.UseColors), colorBold, l.UseColors)
	case FatalLevel:
		lvl = colorize(colorize(lvl, colorRed, l.UseColors), colorBold, l.UseColors)
	case PanicLevel:
		lvl = colorize(colorize(lvl, colorRed, l.UseColors), colorBold, l.UseColors)
	default:
		lvl = colorize(lvl, colorBold, l.UseColors)
	}

	var entities []interface{}
	entities = append(entities, colorize(ent.time.Format(time.RFC3339), colorDarkGray, l.UseColors))
	entities = append(entities, lvl)
	if l.ReportCaller && ent.caller != nil {
		entities = append(entities, fmt.Sprintf("%s:%v", ent.caller.File, ent.caller.Line))
	}
	entities = append(entities, ent.message)

	buf := bufferPool.Get().(*bytes.Buffer)

	defer func() {
		buf.Reset()
		bufferPool.Put(buf)
	}()

	for i := range entities {
		if i > 0 {
			if _, err := buf.WriteRune(' '); err != nil {
				return nil, err
			}
		}
		if _, err := fmt.Fprint(buf, entities[i]); err != nil {
			return nil, err
		}
	}

	fieldsMap := map[string]Field{}
	fieldKeys := []string{}
	for i := range fields {
		fieldsMap[fields[i].key] = fields[i]
		fieldKeys = append(fieldKeys, fields[i].key)
	}

	if !l.DisableSorting {
		sort.Strings(fieldKeys)
	}

	for _, key := range fieldKeys {
		if err := l.appendField(buf, fieldsMap[key]); err != nil {
			return nil, err
		}
	}

	if l.development && ent.stack != nil {
		if _, err := buf.WriteString(fmt.Sprintf("%+v", ent.stack)); err != nil {
			return nil, err
		}
	}

	if _, err := buf.WriteRune('\n'); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (l consoleEncoder) appendField(b *bytes.Buffer, field Field) error {
	var keyColour = colorCyan
	var value string

	switch field.tp {
	case stringType:
		value = field.string
	case intType:
		value = strconv.FormatInt(field.int64, 10)
	case int8Type:
		value = strconv.FormatInt(field.int64, 10)
	case int16Type:
		value = strconv.FormatInt(field.int64, 10)
	case int32Type:
		value = strconv.FormatInt(field.int64, 10)
	case int64Type:
		value = strconv.FormatInt(field.int64, 10)
	case float32Type:
		value = strconv.FormatFloat(field.float64, 'E', -1, 32)
	case float64Type:
		value = strconv.FormatFloat(field.float64, 'E', -1, 64)
	case timeType:
		tm := field.value.(time.Time)
		value = tm.Format(time.RFC3339Nano)
	case jsonType:
		bts, err := json.Marshal(field.value)
		if err != nil {
			return err
		}
		value = string(bts)
	case interfaceType:
		value = fmt.Sprint(field.value)
	case errorType:
		keyColour = colorRed
		value = field.err.Error()
	default:
		return errors.New("unknown field type")
	}

	if b.Len() > 0 {
		b.WriteByte(' ')
	}
	b.WriteString(colorize(field.key, keyColour, l.UseColors))
	b.WriteByte('=')
	l.appendValue(b, value)

	return nil
}

func (l consoleEncoder) appendValue(b *bytes.Buffer, value string) {
	if !l.needsQuoting(value) {
		b.WriteString(value)
	} else {
		b.WriteString(fmt.Sprintf("%q", value))
	}
}

func (l consoleEncoder) needsQuoting(text string) bool {
	if l.ForceQuote {
		return true
	}
	if l.QuoteEmptyFields && len(text) == 0 {
		return true
	}
	if l.DisableQuote {
		return false
	}
	for _, ch := range text {
		if !((ch >= 'a' && ch <= 'z') ||
			(ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') ||
			ch == '-' || ch == '.' || ch == '_' || ch == '/' || ch == '@' || ch == '^' || ch == '+') {
			return true
		}
	}
	return false
}

func consoleLevel(lvl Level) string {
	switch lvl {
	case TraceLevel:
		return "TRC"
	case DebugLevel:
		return "DBG"
	case InfoLevel:
		return "INF"
	case WarnLevel:
		return "WRN"
	case ErrorLevel:
		return "ERR"
	case PanicLevel:
		return "PNC"
	case FatalLevel:
		return "FTL"
	default:
		return "???"
	}
}
