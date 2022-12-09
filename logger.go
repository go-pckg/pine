package pine

import (
	"fmt"
	"io"
	"os"
	"sync"
)

const defaultFramesToSkip = 3

type config struct {
	encoderConfig

	level           *LevelValue
	stackTraceLevel *LevelValue
	out             io.Writer
	errOut          io.Writer
	clock           Clock
	fields          map[string]Field
}

func NewDevelopment(options ...Option) *Logger {
	cfg := config{
		encoderConfig: encoderConfig{
			development: true,
			UseColors:   true,
		},
		out:             os.Stderr,
		errOut:          os.Stderr,
		clock:           DefaultClock,
		level:           NewLevelValue(DebugLevel),
		stackTraceLevel: NewLevelValue(ErrorLevel),
		fields:          map[string]Field{},
	}

	for _, opt := range options {
		opt.apply(&cfg)
	}

	return create(cfg)
}

func New(options ...Option) *Logger {
	cfg := config{
		encoderConfig:   encoderConfig{},
		out:             os.Stderr,
		errOut:          os.Stderr,
		clock:           DefaultClock,
		level:           NewLevelValue(readEnvOrDefaultLevel(DebugLevel)),
		stackTraceLevel: NewLevelValue(ErrorLevel),
		fields:          map[string]Field{},
	}

	for _, opt := range options {
		opt.apply(&cfg)
	}

	return create(cfg)
}

func create(cfg config) *Logger {
	enc := newConsoleEncoder(cfg.encoderConfig)

	lgr := &Logger{
		out:             cfg.out,
		errOut:          cfg.errOut,
		encoder:         enc,
		clock:           cfg.clock,
		lock:            &sync.Mutex{},
		fields:          cfg.fields,
		level:           cfg.level,
		stackTraceLevel: cfg.stackTraceLevel,
	}

	return lgr
}

type Logger struct {
	level           *LevelValue
	stackTraceLevel *LevelValue

	encoder encoder

	out    io.Writer
	errOut io.Writer
	lock   *sync.Mutex
	clock  Clock
	fields map[string]Field
}

func (l *Logger) clone() *Logger {
	lg := &Logger{
		out:             l.out,
		errOut:          l.errOut,
		level:           l.level,
		stackTraceLevel: l.stackTraceLevel,

		encoder: l.encoder.clone(),

		clock:  l.clock,
		lock:   l.lock,
		fields: map[string]Field{},
	}
	for k := range l.fields {
		lg.fields[k] = l.fields[k]
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

func (l *Logger) log(lvl Level, template string, fmtArgs []interface{}, fields []Field) {
	if l.isLevelEnabled(lvl) {
		msg := sprintf(template, fmtArgs)
		e := l.newEntry()
		defer func() {
			entryPool.Put(e)
		}()

		e.level = lvl
		e.message = msg
		e.logger = l
		e.logCaller(defaultFramesToSkip)

		for i := range fields {
			if fields[i].tp == errorType && fields[i].err != nil && l.shouldPrintTrace(lvl) {
				stackTracer := getStackTracer(fields[i].err)
				if stackTracer != nil {
					e.stack = stackTracer.StackTrace()
					stackTrace := marshalStack(e.stack)
					fields = append(fields, Json("stack", stackTrace))
				}
			}
		}

		for i := range l.fields {
			fields = append(fields, l.fields[i])
		}

		l.lock.Lock()
		defer l.lock.Unlock()

		if err := l.write(e, fields); err != nil {
			if l.errOut != nil {
				fmt.Fprintf(l.errOut, "%v write error: %v\n", e.time, err)
			}
		}
	}
}

func (l *Logger) isLevelEnabled(lvl Level) bool {
	return l.level.GetLevel() >= lvl
}

func (l *Logger) shouldPrintTrace(lvl Level) bool {
	return l.stackTraceLevel.GetLevel() >= lvl
}

func (l *Logger) newEntry() *Entry {
	e := entryPool.Get().(*Entry)
	e.time = l.clock.Now()
	return e
}

func (l *Logger) write(ent *Entry, fields []Field) error {
	buf, err := l.encoder.encodeEntry(ent, fields)
	if err != nil {
		return err
	}
	_, err = l.out.Write(buf)
	if err != nil {
		return err
	}
	return nil
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

func readEnvOrDefaultLevel(defaultLevel Level) Level {
	level := os.Getenv("PINE_LEVEL")
	if level == "" {
		return defaultLevel
	}

	lvl, err := ParseLevel(level)
	if err != nil {
		return defaultLevel
	}
	return lvl
}
