package pine

import (
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
	Logger  *Logger
	Level   Level
	Time    time.Time
	Message string
	Caller  *Caller
	Stack   errors.StackTrace
	Fields  []Field
}

func (e *Entry) Debugf(msg string, args ...interface{}) {
	e.Logger.log(DebugLevel, msg, args, e.Fields)
}

func (e *Entry) Infof(msg string, args ...interface{}) {
	e.Logger.log(InfoLevel, msg, args, e.Fields)
}

func (e *Entry) Warnf(msg string, args ...interface{}) {
	e.Logger.log(WarnLevel, msg, args, e.Fields)
}

func (e *Entry) Errorf(msg string, args ...interface{}) {
	e.Logger.log(ErrorLevel, msg, args, e.Fields)
}

func (e *Entry) Panicf(msg string, args ...interface{}) {
	e.Logger.log(PanicLevel, msg, args, e.Fields)
}

func (e *Entry) Fatalf(msg string, args ...interface{}) {
	e.Logger.log(FatalLevel, msg, args, e.Fields)
	entryPool.Put(e)
}

func (e *Entry) logCaller(skipFrame int) {
	_, file, line, ok := getCaller(skipFrame)
	if !ok {
		e.Caller = nil
		return
	}
	e.Caller = &Caller{File: shortFile(file), Line: line}
}
