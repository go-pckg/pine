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

var testDate = time.Date(2022, time.August, 10, 21, 29, 59, 123456789, time.UTC)

type testClock struct {
	date time.Time
}

func newTestClock() *testClock {
	return &testClock{date: testDate}
}

func newTestClockWithDate(date time.Time) *testClock {
	return &testClock{date: date}
}

func (c testClock) Now() time.Time {
	return c.date
}

func TestLogger(t *testing.T) {
	buf := &bytes.Buffer{}
	lgr := New(NoColors(), Output(buf), WithClock(newTestClock()), WithLevel(TraceLevel), WithStackTraceLevel(DisabledLevel))

	t.Run("console", func(tt *testing.T) {
		lgr.Info("hello")
		assert.Equal(tt, "2022-08-10T21:29:59.123Z INF hello\n", buf.String())
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

		assert.Equal(tt, `2022-08-10T21:29:59.123Z TRC hello error="test error" float32=6.1E+00 float64=7.2E+00 int=1 int16=3 int32=4 int64=5 int8=2 json="{\"A\":\"B\"}" obj="{B}" string=s time="2022-08-10T21:29:59.123456789Z"
`, buf.String())
		buf.Reset()
	})

	t.Run("with", func(tt *testing.T) {
		lgr2 := lgr.With(Int("i", 1))
		lgr2.Info("hello")
		assert.Equal(tt, "2022-08-10T21:29:59.123Z INF hello i=1\n", buf.String())
		buf.Reset()
	})

	t.Run("nil error", func(tt *testing.T) {
		lgr.Info("hello", Err(nil))
		assert.Equal(tt, "2022-08-10T21:29:59.123Z INF hello\n", buf.String())
		buf.Reset()
	})
}

func TestLogger_DefaultFields(t *testing.T) {
	buf := &bytes.Buffer{}
	lgr := New(NoColors(), Output(buf), WithClock(newTestClock()), Fields(String("A", "B")))
	lgr.Info("hello")
	assert.Equal(t, "2022-08-10T21:29:59.123Z INF hello A=B\n", buf.String())
	buf.Reset()

	lgr2 := lgr.With(Int("i", 1))
	lgr2.Info("hello")
	assert.Equal(t, "2022-08-10T21:29:59.123Z INF hello A=B i=1\n", buf.String())
}

func TestLogger_Order(t *testing.T) {
	buf := &bytes.Buffer{}
	lgr := New(NoColors(), Output(buf), WithClock(newTestClock()), NoSorting())
	lgr.Info("hello", String("C", "3"), String("B", "2"), String("A", "1"))
	assert.Equal(t, "2022-08-10T21:29:59.123Z INF hello C=3 B=2 A=1\n", buf.String())
	buf.Reset()

	lgr = New(NoColors(), Output(buf), WithClock(newTestClock()))
	lgr.Info("hello", String("C", "3"), String("B", "2"), String("A", "1"))
	assert.Equal(t, "2022-08-10T21:29:59.123Z INF hello A=1 B=2 C=3\n", buf.String())
}

func TestLogger_LevelChange(t *testing.T) {
	lvl := NewLevelValue(DebugLevel)

	buf := &bytes.Buffer{}
	lgr := New(NoColors(), Output(buf), WithClock(newTestClock()), WithLevelValue(lvl))
	lgr.Info("hello")
	assert.Equal(t, "2022-08-10T21:29:59.123Z INF hello\n", buf.String())
	buf.Reset()

	lvl.SetLevel(PanicLevel)

	lgr.Info("hello")
	assert.Equal(t, "", buf.String())
}

func TestLogger_Colored(t *testing.T) {
	buf := &bytes.Buffer{}
	lgr := New(WithColors(), Output(buf), WithClock(newTestClock()), WithLevel(TraceLevel))

	lgr.Trace("hello", Int("i", 1), Err(fmt.Errorf("err")))
	assert.Equal(t, "\x1b[90m2022-08-10T21:29:59.123Z\x1b[0m \x1b[35mTRC\x1b[0m hello \x1b[31merror\x1b[0m=err \x1b[36mi\x1b[0m=1\n", buf.String())
	buf.Reset()

	lgr.Debug("hello", Int("i", 1), Err(fmt.Errorf("err")))
	assert.Equal(t, "\x1b[90m2022-08-10T21:29:59.123Z\x1b[0m \x1b[33mDBG\x1b[0m hello \x1b[31merror\x1b[0m=err \x1b[36mi\x1b[0m=1\n", buf.String())
	buf.Reset()

	lgr.Info("hello", Int("i", 1), Err(fmt.Errorf("err")))
	assert.Equal(t, "\x1b[90m2022-08-10T21:29:59.123Z\x1b[0m \x1b[32mINF\x1b[0m hello \x1b[31merror\x1b[0m=err \x1b[36mi\x1b[0m=1\n", buf.String())
	buf.Reset()

	lgr.Warn("hello", Int("i", 1), Err(fmt.Errorf("err")))
	assert.Equal(t, "\x1b[90m2022-08-10T21:29:59.123Z\x1b[0m \x1b[31mWRN\x1b[0m hello \x1b[31merror\x1b[0m=err \x1b[36mi\x1b[0m=1\n", buf.String())
	buf.Reset()

	lgr.Error("hello", Int("i", 1), Err(fmt.Errorf("err")))
	assert.Equal(t, "\x1b[90m2022-08-10T21:29:59.123Z\x1b[0m \x1b[1m\x1b[31mERR\x1b[0m\x1b[0m hello \x1b[31merror\x1b[0m=err \x1b[36mi\x1b[0m=1\n", buf.String())
	buf.Reset()
}

func TestLogger_Caller(t *testing.T) {
	buf := &bytes.Buffer{}
	lgr := New(AddCaller(), Output(buf), WithClock(newTestClock()))
	lgr.Info("testcaller")
	_, _, line, _ := runtime.Caller(0) //nolint:dogsled
	assert.Equal(t, fmt.Sprintf("2022-08-10T21:29:59.123Z INF logger_test.go:%v testcaller\n", line-1), buf.String())
}

func TestLogger_WithFields(t *testing.T) {
	buf := &bytes.Buffer{}
	lgr := New(Output(buf), WithClock(newTestClock()))
	lgr.WithFields(Int("i", 1)).Warnf("hello %v", "world")
	assert.Equal(t, "2022-08-10T21:29:59.123Z WRN hello world i=1\n", buf.String())
}

func TestLogger_Stacktrace(t *testing.T) {
	buf := &bytes.Buffer{}
	lgr := New(Output(buf), WithClock(newTestClock()))
	lgr.Error("hello", Err(outer()))
	assert.Regexp(t, "2022-08-10T21:29:59.123Z ERR hello error=test stack=\"\\[{\\\\\"func\\\\\":\\\\\"inner\\\\\",\\\\\"line\\\\\":\\\\\"10\\\\\",\\\\\"source\\\\\":\\\\\"stacktrace_test.go\\\\\"},{\\\\\"func\\\\\":\\\\\"outer\\\\\",\\\\\"line\\\\\":\\\\\"6\\\\\",\\\\\"source\\\\\":\\\\\"stacktrace_test.go\\\\\"}(.*)\n", buf.String())
}

func TestLogger_Error(t *testing.T) {
	buf := &bytes.Buffer{}
	lgr := New(Output(buf), WithClock(newTestClock()))
	lgr.Error("hello", Err(fmt.Errorf("test")))
	assert.Equal(t, "2022-08-10T21:29:59.123Z ERR hello error=test\n", buf.String())
}

func TestLogger_Format(t *testing.T) {
	buf := &bytes.Buffer{}
	lgr := New(Output(buf), WithClock(newTestClock()))

	lgr.Debugf("hello %v", 1)
	assert.Equal(t, "2022-08-10T21:29:59.123Z DBG hello 1\n", buf.String())
	buf.Reset()
}

func TestLogger_Time(t *testing.T) {
	lg := func(d time.Time) string {
		buf := &bytes.Buffer{}
		lgr2 := New(Output(buf), WithClock(newTestClockWithDate(d)))
		lgr2.Debugf("hello %v", 1)
		return buf.String()
	}

	t.Run("utc with milliseconds", func(t *testing.T) {
		d := time.Date(2022, time.August, 10, 21, 29, 59, 123456789, time.UTC)
		got := lg(d)
		assert.Equal(t, "2022-08-10T21:29:59.123Z DBG hello 1\n", got)
	})

	t.Run("utc no milliseconds", func(t *testing.T) {
		d := time.Date(2022, time.August, 10, 21, 29, 59, 789, time.UTC)
		got := lg(d)
		assert.Equal(t, "2022-08-10T21:29:59.000Z DBG hello 1\n", got)
	})

	t.Run("auckland", func(t *testing.T) {
		loc, err := time.LoadLocation("Pacific/Auckland")
		require.NoError(t, err)
		d := time.Date(2022, time.August, 10, 21, 29, 59, 789, loc)
		got := lg(d)
		assert.Equal(t, "2022-08-10T21:29:59.000+12:00 DBG hello 1\n", got)
	})
}

func TestLogger_Race(t *testing.T) {
	buf := &bytes.Buffer{}
	lgr := New(Output(buf), ErrOutput(os.Stderr), WithClock(newTestClock()))

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
		require.Equal(t, "2022-08-10T21:29:59.123Z INF hello\n", line)
	}

	assert.Equal(t, workers*cycles, counter)
}

func TestLogger_RaceMultiple(t *testing.T) {
	buf := &bytes.Buffer{}
	lgr1 := New(Output(buf), ErrOutput(os.Stderr), WithClock(newTestClock()), Fields(Int("i", 1)))
	lgr2 := lgr1.With(Int("i", 1))
	lgr3 := lgr2.With(Int("i", 1))

	loggers := []*Logger{lgr1, lgr2, lgr3}

	workers := 3
	cycles := 1000

	wg := sync.WaitGroup{}
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func(i int) {
			lgr := loggers[i]
			for j := 0; j < cycles; j++ {
				lgr.Info("hello")
			}
			wg.Done()
		}(i)
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
		require.Equal(t, "2022-08-10T21:29:59.123Z INF hello i=1\n", line)
	}

	assert.Equal(t, workers*cycles, counter)
}
