package pine

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/go-pckg/pine/gelf"
	"github.com/pkg/errors"
)

var defaultHostname = func() string {
	hostname, _ := os.Hostname()
	return hostname
}

// 1k bytes buffer by default
var bufPool = sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer(make([]byte, 0, 1024))
	},
}

func newBuffer() *bytes.Buffer {
	b := bufPool.Get().(*bytes.Buffer)
	if b != nil {
		b.Reset()
		return b
	}
	return bytes.NewBuffer(nil)
}

var gelfPool = &sync.Pool{
	New: func() interface{} {
		return &gelf.Message{}
	},
}

func newGelfMsg() *gelf.Message {
	m := gelfPool.Get().(*gelf.Message)
	if m != nil {
		return m
	}
	return &gelf.Message{}
}

type encoderConfig struct {
	UseColors        bool
	ForceQuote       bool
	QuoteEmptyFields bool
	DisableQuote     bool
	ReportCaller     bool
	DisableSorting   bool
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
	entities = append(entities, colorize(ent.time.Format("2006-01-02T15:04:05.000Z07:00"), colorDarkGray, l.UseColors))
	entities = append(entities, lvl)
	if l.ReportCaller && ent.caller != nil {
		entities = append(entities, fmt.Sprintf("%s:%v", ent.caller.File, ent.caller.Line))
	}
	entities = append(entities, ent.message)

	buf := newBuffer()
	defer func() {
		bufPool.Put(buf)
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

	if ent.stack != nil {
		stackTrace := marshalStack(ent.stack)
		fields = append(fields, Json("stack", stackTrace))
	}

	fieldsMap := map[string][]Field{}
	fieldKeys := []string{}
	for i := range fields {
		if _, ok := fieldsMap[fields[i].key]; !ok {
			fieldKeys = append(fieldKeys, fields[i].key)
		}
		fieldsMap[fields[i].key] = append(fieldsMap[fields[i].key], fields[i])
	}

	if !l.DisableSorting {
		sort.Strings(fieldKeys)
	}

	for _, key := range fieldKeys {
		for _, field := range fieldsMap[key] {
			if err := l.appendField(buf, field); err != nil {
				return nil, err
			}
		}
	}

	if _, err := buf.WriteRune('\n'); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (l consoleEncoder) appendField(b *bytes.Buffer, field Field) error {
	ok, value, err := getStringValue(field)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}

	var keyColour = colorCyan
	switch field.tp {
	case errorType:
		keyColour = colorRed
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

type gelfEncoder struct {
	extraFields map[string]Field
}

func newGelfEncoder(extraFields map[string]Field) gelfEncoder {
	return gelfEncoder{extraFields: extraFields}
}

func (l gelfEncoder) clone() encoder {
	return newGelfEncoder(l.extraFields)
}

func (l gelfEncoder) encodeEntry(ent *Entry, fields []Field) ([]byte, error) {
	hostname := defaultHostname()
	for k, f := range l.extraFields {
		if k == "host" {
			hostname = f.string
			break
		}
	}
	for i := range fields {
		if fields[i].key == "host" {
			hostname = fields[i].string
			break
		}
	}

	gelfMsg := newGelfMsg()
	defer gelfPool.Put(gelfMsg)
	gelfMsg.Version = "1.1"
	gelfMsg.Level = gelfLevel(ent.level)
	gelfMsg.TimeUnix = float64(ent.time.Unix())
	gelfMsg.Short = ent.message
	gelfMsg.Full = ""
	gelfMsg.Host = hostname
	gelfMsg.Extra = map[string]interface{}{}

	if ent.caller != nil {
		gelfMsg.Extra["_caller"] = fmt.Sprintf("%s:%v", ent.caller.File, ent.caller.Line)
		gelfMsg.Extra["_file"] = ent.caller.File
		gelfMsg.Extra["_line"] = ent.caller.Line
	}

	for i := range l.extraFields {
		if l.extraFields[i].key == "host" {
			continue
		}
		if err := l.appendField(gelfMsg.Extra, l.extraFields[i]); err != nil {
			return nil, err
		}
	}

	for i := range fields {
		if fields[i].key == "host" {
			continue
		}
		if err := l.appendField(gelfMsg.Extra, fields[i]); err != nil {
			return nil, err
		}
	}

	if ent.stack != nil {
		gelfMsg.Extra["_stack"] = fmt.Sprintf("%+v", ent.stack)
	}

	buf := newBuffer()
	defer bufPool.Put(buf)
	if err := gelfMsg.MarshalJSONBuf(buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (l gelfEncoder) appendField(m map[string]interface{}, field Field) error {
	ok, value, err := getStringValue(field)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}

	m["_"+field.key] = value

	return nil
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

func gelfLevel(lvl Level) int32 {
	switch lvl {
	case TraceLevel:
		return 7
	case DebugLevel:
		return 7
	case InfoLevel:
		return 6
	case WarnLevel:
		return 4
	case ErrorLevel:
		return 3
	case PanicLevel:
		return 2
	case FatalLevel:
		return 1
	default:
		return 7
	}
}

func getStringValue(field Field) (bool, string, error) {
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
	case boolType:
		bv := false
		if field.int64 == 1 {
			bv = true
		}
		value = strconv.FormatBool(bv)
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
			return false, "", err
		}
		value = string(bts)
	case interfaceType:
		value = fmt.Sprint(field.value)
	case errorType:
		if field.err == nil {
			return false, "", nil
		}
		value = field.err.Error()
	default:
		return false, "", errors.New("unknown field type")
	}

	return true, value, nil
}
