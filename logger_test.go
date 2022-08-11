package pine

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testDate = time.Date(2022, time.August, 10, 21, 29, 59, 4, time.UTC)

type testClock struct{}

func (testClock) Now() time.Time {
	return testDate
}

func TestLogger(t *testing.T) {
	buf := &bytes.Buffer{}
	lgr := New(NoColors(), Output(buf), WithClock(testClock{}), WithLevel(TraceLevel), WithStackTraceLevel(DisabledLevel))

	t.Run("console", func(tt *testing.T) {
		lgr.Info("hello")
		assert.Equal(tt, "2022-08-10T21:29:59Z INF hello\n", buf.String())
		buf.Reset()
	})

	t.Run("fields", func(tt *testing.T) {
		obj := struct {
			A string `json:"A"`
		}{A: "B"}

		lgr.Trace("hello",
			Int("int", 1),
			Int8("int8", 2),
			Int16("int16", 3),
			Int32("int32", 4),
			Int64("int64", 5),
			Float32("float32", 6.1),
			Float64("float64", 7.2),
			String("string", "s"),
			Time("time", testDate),
			Err(fmt.Errorf("test error")),
			Json("json", obj),
			Interface("obj", obj),
		)

		assert.Equal(tt, `2022-08-10T21:29:59Z TRC hello error="test error" float32=6.1E+00 float64=7.2E+00 int=1 int16=3 int32=4 int64=5 int8=2 json="{\"A\":\"B\"}" obj="{B}" string=s time="2022-08-10T21:29:59.000000004Z"
`, buf.String())
		buf.Reset()
	})

	t.Run("with", func(tt *testing.T) {
		lgr2 := lgr.With(Int("i", 1))
		lgr2.Info("hello")
		assert.Equal(tt, "2022-08-10T21:29:59Z INF hello i=1\n", buf.String())
		buf.Reset()
	})
}

func TestLogger_DefaultFields(t *testing.T) {
	buf := &bytes.Buffer{}
	lgr := New(NoColors(), Output(buf), WithClock(testClock{}), Fields(String("A", "B")))
	lgr.Info("hello")
	assert.Equal(t, "2022-08-10T21:29:59Z INF hello A=B\n", buf.String())
	buf.Reset()

	lgr2 := lgr.With(Int("i", 1))
	lgr2.Info("hello")
	assert.Equal(t, "2022-08-10T21:29:59Z INF hello A=B i=1\n", buf.String())
}

func TestLogger_Order(t *testing.T) {
	buf := &bytes.Buffer{}
	lgr := New(NoColors(), Output(buf), WithClock(testClock{}), NoSorting())
	lgr.Info("hello", String("C", "3"), String("B", "2"), String("A", "1"))
	assert.Equal(t, "2022-08-10T21:29:59Z INF hello C=3 B=2 A=1\n", buf.String())
	buf.Reset()

	lgr = New(NoColors(), Output(buf), WithClock(testClock{}))
	lgr.Info("hello", String("C", "3"), String("B", "2"), String("A", "1"))
	assert.Equal(t, "2022-08-10T21:29:59Z INF hello A=1 B=2 C=3\n", buf.String())
}

func TestLogger_LevelChange(t *testing.T) {
	lvl := NewLevelValue(DebugLevel)

	buf := &bytes.Buffer{}
	lgr := New(NoColors(), Output(buf), WithClock(testClock{}), WithLevelValue(lvl))
	lgr.Info("hello")
	assert.Equal(t, "2022-08-10T21:29:59Z INF hello\n", buf.String())
	buf.Reset()

	lvl.SetLevel(PanicLevel)

	lgr.Info("hello")
	assert.Equal(t, "", buf.String())
}

func TestLogger_Colored(t *testing.T) {
	buf := &bytes.Buffer{}
	lgr := New(WithColors(), Output(buf), WithClock(testClock{}), WithLevel(TraceLevel))

	lgr.Trace("hello", Int("i", 1), Err(fmt.Errorf("err")))
	assert.Equal(t, "\x1b[90m2022-08-10T21:29:59Z\x1b[0m \x1b[35mTRC\x1b[0m hello \x1b[31merror\x1b[0m=err \x1b[36mi\x1b[0m=1\n", buf.String())
	buf.Reset()

	lgr.Debug("hello", Int("i", 1), Err(fmt.Errorf("err")))
	assert.Equal(t, "\x1b[90m2022-08-10T21:29:59Z\x1b[0m \x1b[33mDBG\x1b[0m hello \x1b[31merror\x1b[0m=err \x1b[36mi\x1b[0m=1\n", buf.String())
	buf.Reset()

	lgr.Info("hello", Int("i", 1), Err(fmt.Errorf("err")))
	assert.Equal(t, "\x1b[90m2022-08-10T21:29:59Z\x1b[0m \x1b[32mINF\x1b[0m hello \x1b[31merror\x1b[0m=err \x1b[36mi\x1b[0m=1\n", buf.String())
	buf.Reset()

	lgr.Warn("hello", Int("i", 1), Err(fmt.Errorf("err")))
	assert.Equal(t, "\x1b[90m2022-08-10T21:29:59Z\x1b[0m \x1b[31mWRN\x1b[0m hello \x1b[31merror\x1b[0m=err \x1b[36mi\x1b[0m=1\n", buf.String())
	buf.Reset()

	lgr.Error("hello", Int("i", 1), Err(fmt.Errorf("err")))
	assert.Equal(t, "\x1b[90m2022-08-10T21:29:59Z\x1b[0m \x1b[1m\x1b[31mERR\x1b[0m\x1b[0m hello \x1b[31merror\x1b[0m=err \x1b[36mi\x1b[0m=1\n", buf.String())
	buf.Reset()
}

func TestLogger_Caller(t *testing.T) {
	buf := &bytes.Buffer{}
	lgr := New(AddCaller(), Output(buf), WithClock(testClock{}))
	lgr.Info("testcaller")
	_, _, line, _ := runtime.Caller(0) //nolint:dogsled
	assert.Equal(t, fmt.Sprintf("2022-08-10T21:29:59Z INF logger_test.go:%v testcaller\n", line-1), buf.String())
}

func TestLogger_WithFields(t *testing.T) {
	buf := &bytes.Buffer{}
	lgr := New(Output(buf), WithClock(testClock{}))
	lgr.WithFields(Int("i", 1)).Warnf("hello %v", "world")
	assert.Equal(t, "2022-08-10T21:29:59Z WRN hello world i=1\n", buf.String())
}

func TestLogger_Stacktrace(t *testing.T) {
	buf := &bytes.Buffer{}
	lgr := New(Output(buf), WithClock(testClock{}))
	lgr.Error("hello", Err(outer()))
	assert.Regexp(t, "2022-08-10T21:29:59Z ERR hello error=test stack=\"\\[{\\\\\"func\\\\\":\\\\\"inner\\\\\",\\\\\"line\\\\\":\\\\\"10\\\\\",\\\\\"source\\\\\":\\\\\"stacktrace_test.go\\\\\"},{\\\\\"func\\\\\":\\\\\"outer\\\\\",\\\\\"line\\\\\":\\\\\"6\\\\\",\\\\\"source\\\\\":\\\\\"stacktrace_test.go\\\\\"}(.*)\n", buf.String())
}

func TestLogger_Error(t *testing.T) {
	buf := &bytes.Buffer{}
	lgr := New(Output(buf), WithClock(testClock{}))
	lgr.Error("hello", Err(fmt.Errorf("test")))
	assert.Equal(t, "2022-08-10T21:29:59Z ERR hello error=test\n", buf.String())
}

func TestLogger_Format(t *testing.T) {
	buf := &bytes.Buffer{}
	lgr := New(Output(buf), WithClock(testClock{}))

	lgr.Debugf("hello %v", 1)
	assert.Equal(t, "2022-08-10T21:29:59Z DBG hello 1\n", buf.String())
	buf.Reset()
}

func TestLogger_Race(t *testing.T) {
	buf := &bytes.Buffer{}
	lgr := New(Output(buf), ErrOutput(os.Stderr), WithClock(testClock{}))

	workers := 3
	cycles := 1000

	wg := sync.WaitGroup{}
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			for j := 0; j < cycles; j++ {
				lgr.Info("hello")
			}
			wg.Done()
		}()
	}
	wg.Wait()

	counter := 0
	for {
		line, err := buf.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			} else {
				t.Fatal(err)
			}
		}
		counter++
		require.Equal(t, "2022-08-10T21:29:59Z INF hello\n", line)
	}

	assert.Equal(t, workers*cycles, counter)
}
