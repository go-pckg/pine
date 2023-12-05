package pine

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/go-pckg/pine/gelf"
)

const defaultFramesToSkip = 4

type consoleConfig struct {
	encoderConfig encoderConfig
	level         *LevelValue
	out           io.Writer
}

type gelfConfig struct {
	Enabled bool
	Addr    string
	Level   *LevelValue
}

type config struct {
	consoleConfig consoleConfig
	gelfConfig    gelfConfig

	stackTraceLevel *LevelValue
	errOut          io.Writer
	clock           Clock
	fields          map[string]Field
}

func New(options ...Option) *Logger {
	cfg := config{
		consoleConfig: consoleConfig{
			encoderConfig: encoderConfig{
				UseColors: readEnvOrDefaultUseColors(false),
			},
			level: NewLevelValue(readEnvOrDefaultLevel("PINE_LEVEL", DebugLevel)),
			out:   os.Stderr,
		},
		gelfConfig: gelfConfig{
			Enabled: readEnvOrDefaultBool("PINE_GRAYLOG_ENABLED", false),
			Level:   NewLevelValue(readEnvOrDefaultLevel("PINE_GRAYLOG_LEVEL", readEnvOrDefaultLevel("PINE_LEVEL", DebugLevel))),
			Addr:    readEnvOrDefaultString("PINE_GRAYLOG_ADDR", ""),
		},
		errOut:          os.Stderr,
		clock:           DefaultClock,
		stackTraceLevel: NewLevelValue(ErrorLevel),
		fields:          map[string]Field{},
	}

	for _, opt := range options {
		opt.apply(&cfg)
	}

	return create(cfg)
}

func create(cfg config) *Logger {
	handlers := []handler{
		&consoleHandler{
			level:   cfg.consoleConfig.level,
			encoder: newConsoleEncoder(cfg.consoleConfig.encoderConfig),
			out:     cfg.consoleConfig.out,
		},
	}
	if cfg.gelfConfig.Enabled {
		handlers = append(handlers, &gelfHandler{
			level:   cfg.gelfConfig.Level,
			encoder: newGelfEncoder(),
			out:     gelf.NewTCPWriter(cfg.gelfConfig.Addr),
			errOut:  cfg.errOut,
		})
	}

	lgr := &Logger{
		handlers:        handlers,
		errOut:          cfg.errOut,
		clock:           cfg.clock,
		lock:            &sync.Mutex{},
		fields:          cfg.fields,
		stackTraceLevel: cfg.stackTraceLevel,
	}

	return lgr
}

type Logger struct {
	stackTraceLevel *LevelValue

	handlers []handler

	errOut io.Writer
	lock   *sync.Mutex
	clock  Clock
	fields map[string]Field
}

func (l *Logger) clone() *Logger {
	lg := &Logger{
		errOut:          l.errOut,
		stackTraceLevel: l.stackTraceLevel,

		clock:  l.clock,
		lock:   l.lock,
		fields: map[string]Field{},
	}
	for k := range l.fields {
		lg.fields[k] = l.fields[k]
	}
	for i := range l.handlers {
		lg.handlers = append(lg.handlers, l.handlers[i].clone())
	}
	return lg
}

func (l *Logger) With(fields ...Field) *Logger {
	if len(fields) == 0 {
		return l
	}

	lg := l.clone()

	for i := range fields {
		lg.fields[fields[i].key] = fields[i]
	}

	return lg
}

func (l *Logger) Trace(msg string, fields ...Field) {
	l.log(TraceLevel, msg, nil, fields)
}

func (l *Logger) Tracef(msg string, a ...interface{}) {
	l.log(TraceLevel, msg, a, nil)
}

func (l *Logger) Debug(msg string, fields ...Field) {
	l.log(DebugLevel, msg, nil, fields)
}

func (l *Logger) Debugf(msg string, a ...interface{}) {
	l.log(DebugLevel, msg, a, nil)
}

func (l *Logger) Info(msg string, fields ...Field) {
	l.log(InfoLevel, msg, nil, fields)
}

func (l *Logger) Infof(msg string, a ...interface{}) {
	l.log(InfoLevel, msg, a, nil)
}

func (l *Logger) Warn(msg string, fields ...Field) {
	l.log(WarnLevel, msg, nil, fields)
}

func (l *Logger) Warnf(msg string, a ...interface{}) {
	l.log(WarnLevel, msg, a, nil)
}

func (l *Logger) Error(msg string, fields ...Field) {
	l.log(ErrorLevel, msg, nil, fields)
}

func (l *Logger) Errorf(msg string, a ...interface{}) {
	l.log(ErrorLevel, msg, a, nil)
}

func (l *Logger) Panic(msg string, fields ...Field) {
	l.log(PanicLevel, msg, nil, fields)
}

func (l *Logger) Panicf(msg string, a ...interface{}) {
	l.log(PanicLevel, msg, a, nil)
}

func (l *Logger) Fatal(msg string, fields ...Field) {
	l.log(FatalLevel, msg, nil, fields)
}

func (l *Logger) Fatalf(msg string, a ...interface{}) {
	l.log(FatalLevel, msg, a, nil)
}

func (l *Logger) WithFields(fields ...Field) *Entry {
	e := l.newEntry()
	e.fields = fields
	e.logger = l
	return e
}

func (l *Logger) Close() {
	l.lock.Lock()
	defer l.lock.Unlock()

	for i := range l.handlers {
		l.handlers[i].close()
	}
}

func (l *Logger) log(lvl Level, template string, fmtArgs []interface{}, fields []Field) {
	var e *Entry
	for i := range l.handlers {
		if l.handlers[i].isLevelEnabled(lvl) {
			if e == nil {
				e = l.newEntry()
				defer func() {
					entryPool.Put(e)
				}()

				e.level = lvl
				e.message = sprintf(template, fmtArgs)
				e.logger = l
				e.logCaller(defaultFramesToSkip)
				e.stack = nil

				for i := range fields {
					if fields[i].tp == errorType && fields[i].err != nil {
						if l.shouldPrintTrace(lvl) {
							stackTracer := getStackTracer(fields[i].err)
							if stackTracer != nil {
								e.stack = stackTracer.StackTrace()
								stackTrace := marshalStack(e.stack)
								fields = append(fields, Json("stack", stackTrace))
							}
						}
					}
				}

				for i := range l.fields {
					fields = append(fields, l.fields[i])
				}

				l.lock.Lock()
				defer l.lock.Unlock()
			}

			if err := l.handlers[i].write(e, fields); err != nil {
				if l.errOut != nil {
					fmt.Fprintf(l.errOut, "%v write error: %v\n", e.time, err)
				}
			}
		}
	}
}

func (l *Logger) shouldPrintTrace(lvl Level) bool {
	return l.stackTraceLevel.GetLevel() >= lvl
}

func (l *Logger) newEntry() *Entry {
	e := entryPool.Get().(*Entry)
	e.time = l.clock.Now()
	return e
}

func sprintf(format string, a []interface{}) string {
	if len(a) == 0 {
		return format
	}

	if format != "" {
		return fmt.Sprintf(format, a...)
	}

	return fmt.Sprint(a...)
}

func readEnvOrDefaultLevel(key string, defaultLevel Level) Level {
	level := os.Getenv(key)
	if level == "" {
		return defaultLevel
	}

	lvl, err := ParseLevel(level)
	if err != nil {
		return defaultLevel
	}
	return lvl
}

func readEnvOrDefaultUseColors(defaultUseColors bool) bool {
	useColors := os.Getenv("PINE_COLORS")
	if useColors == "" {
		return defaultUseColors
	}

	if strings.ToLower(useColors) == "true" {
		return true
	}
	return false
}

func readEnvOrDefaultBool(key string, defaultVal bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return defaultVal
	}
	return b
}

func readEnvOrDefaultString(key string, defaultVal string) string {
	v, ok := os.LookupEnv(key)
	if !ok {
		return defaultVal
	}
	return v
}

type handler interface {
	isLevelEnabled(lvl Level) bool
	write(ent *Entry, fields []Field) error
	clone() handler
	close()
}

type consoleHandler struct {
	level   *LevelValue
	encoder encoder
	out     io.Writer
}

func (h *consoleHandler) isLevelEnabled(lvl Level) bool {
	return h.level.GetLevel() >= lvl
}

func (h *consoleHandler) write(ent *Entry, fields []Field) error {
	buf, err := h.encoder.encodeEntry(ent, fields)
	if err != nil {
		return err
	}
	_, err = h.out.Write(buf)
	if err != nil {
		return err
	}
	return nil
}

func (h *consoleHandler) clone() handler {
	return &consoleHandler{
		level:   h.level,
		encoder: h.encoder.clone(),
		out:     h.out,
	}
}

func (h *consoleHandler) close() {
	//noop
}

type gelfHandler struct {
	level   *LevelValue
	encoder encoder
	out     io.WriteCloser
	errOut  io.Writer
	mu      sync.Mutex
}

func (h *gelfHandler) isLevelEnabled(lvl Level) bool {
	return h.level.GetLevel() >= lvl
}

func (h *gelfHandler) write(ent *Entry, fields []Field) error {
	buf, err := h.encoder.encodeEntry(ent, fields)
	if err != nil {
		return err
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	_, err = h.out.Write(buf)
	if err != nil {
		return err
	}
	return nil
}

func (h *gelfHandler) clone() handler {
	return &gelfHandler{
		level:   h.level,
		encoder: h.encoder.clone(),
		out:     h.out,
		errOut:  h.errOut,
	}
}

func (h *gelfHandler) close() {
	h.mu.Lock()
	defer h.mu.Unlock()
	err := h.out.Close()
	if err != nil {
		fmt.Fprintf(h.errOut, "gelf close error: %v\n", err)
	}
}
