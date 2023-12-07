package pine

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

type state struct {
	b []byte
}

func (s *state) Write(b []byte) (n int, err error) {
	s.b = b
	return len(b), nil
}

func (s *state) Width() (wid int, ok bool) {
	return 0, false
}

func (s *state) Precision() (prec int, ok bool) {
	return 0, false
}

func (s *state) Flag(c int) bool {
	return false
}

type stackTracer interface {
	StackTrace() errors.StackTrace
}

func getStackTracer(err error) stackTracer {
	var sterr stackTracer
	var ok bool
	for {
		sterr, ok = err.(stackTracer)
		if ok {
			break
		}
		err = errors.Unwrap(err)
		if err == nil {
			break
		}
	}
	if !ok {
		return nil
	}
	return sterr
}

func marshalStack(st errors.StackTrace) interface{} {
	s := &state{}
	out := make([]map[string]string, 0, len(st))
	for _, fr := range st {
		out = append(out, map[string]string{
			"source": formatFrame(fr, s, 's'),
			"line":   formatFrame(fr, s, 'd'),
			"func":   formatFrame(fr, s, 'n'),
		})
	}
	return out
}

func flattenStack(st errors.StackTrace) string {
	sb := strings.Builder{}
	for _, fr := range st {
		sb.WriteString(fmt.Sprintf("%n() at %s:%d <- ", fr, fr, fr))
	}
	return strings.TrimRight(sb.String(), " <- ")
}

type frameFormatter interface {
	Format(s fmt.State, verb rune)
}

func formatFrame(f frameFormatter, s *state, verb rune) string {
	f.Format(s, verb)
	return string(s.b)
}
