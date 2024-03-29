package pine

import "runtime"

var getCaller = func(skip int) (pc uintptr, file string, line int, ok bool) {
	return runtime.Caller(skip)
}

type Caller struct {
	File string
	Line int
}

func shortFile(file string) string {
	short := file
	for i := len(file) - 1; i > 0; i-- {
		if file[i] == '/' {
			short = file[i+1:]
			break
		}
	}
	return short
}
