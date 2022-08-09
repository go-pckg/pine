package pine

import (
	"runtime"
	"sync"
	"time"

	"github.com/pkg/errors"
)

var entryPool = &sync.Pool{
	New: func() interface{} {
		return &Entry{}
	},
}

type Entry struct {
	logger  *Logger
	level   Level
	time    time.Time
	message string
	caller  *Caller
	stack   errors.StackTrace
	fields  []Field
}

func (e *Entry) Debugf(msg string, args ...interface{}) {
	e.logger.log(DebugLevel, msg, args, e.fields)
}

func (e *Entry) Infof(msg string, args ...interface{}) {
	e.logger.log(InfoLevel, msg, args, e.fields)
}

func (e *Entry) Warnf(msg string, args ...interface{}) {
	e.logger.log(WarnLevel, msg, args, e.fields)
}

func (e *Entry) Errorf(msg string, args ...interface{}) {
	e.logger.log(ErrorLevel, msg, args, e.fields)
}

func (e *Entry) Panicf(msg string, args ...interface{}) {
	e.logger.log(PanicLevel, msg, args, e.fields)
}

func (e *Entry) Fatalf(msg string, args ...interface{}) {
	e.logger.log(FatalLevel, msg, args, e.fields)
	entryPool.Put(e)
}

func (e *Entry) logCaller(skipFrame int) {
	_, file, line, ok := runtime.Caller(skipFrame)
	if !ok {
		e.caller = nil
		return
	}
	e.caller = &Caller{File: shortFile(file), Line: line}
}
