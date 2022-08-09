package pine

import "time"

var DefaultClock = systemClock{}

type Clock interface {
	Now() time.Time
}

type systemClock struct{}

func (systemClock) Now() time.Time {
	return time.Now()
}
