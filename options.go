package pine

import (
	"io"
)

type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (f optionFunc) apply(log *config) {
	f(log)
}

func WithLevel(lvl Level) Option {
	return optionFunc(func(log *config) {
		log.consoleConfig.level = NewLevelValue(lvl)
	})
}

func WithLevelValue(lvl *LevelValue) Option {
	return optionFunc(func(log *config) {
		log.consoleConfig.level = lvl
	})
}

func Colored(useColors bool) Option {
	return optionFunc(func(log *config) {
		log.consoleConfig.encoderConfig.UseColors = useColors
	})
}

func WithColors() Option {
	return optionFunc(func(log *config) {
		log.consoleConfig.encoderConfig.UseColors = true
	})
}

func NoColors() Option {
	return optionFunc(func(c *config) {
		c.consoleConfig.encoderConfig.UseColors = false
	})
}

func WithStackTraceLevel(lvl Level) Option {
	return optionFunc(func(c *config) {
		c.stackTraceLevel = NewLevelValue(lvl)
	})
}

func ForceQuote() Option {
	return optionFunc(func(c *config) {
		c.consoleConfig.encoderConfig.ForceQuote = true
	})
}

func QuoteEmptyFields() Option {
	return optionFunc(func(c *config) {
		c.consoleConfig.encoderConfig.QuoteEmptyFields = true
	})
}

func NoQuotes() Option {
	return optionFunc(func(c *config) {
		c.consoleConfig.encoderConfig.DisableQuote = true
	})
}

func Fields(fields ...Field) Option {
	return optionFunc(func(c *config) {
		for i := range fields {
			c.fields[fields[i].key] = fields[i]
		}
	})
}

func AddCaller() Option {
	return WithCaller(true)
}

func WithCaller(enabled bool) Option {
	return optionFunc(func(c *config) {
		c.consoleConfig.encoderConfig.ReportCaller = enabled
	})
}

func NoSorting() Option {
	return optionFunc(func(c *config) {
		c.consoleConfig.encoderConfig.DisableSorting = true
	})
}

func Sorting(disabled bool) Option {
	return optionFunc(func(c *config) {
		c.consoleConfig.encoderConfig.DisableSorting = disabled
	})
}

func WithClock(clock Clock) Option {
	return optionFunc(func(c *config) {
		c.clock = clock
	})
}

func Output(out io.Writer) Option {
	return optionFunc(func(c *config) {
		c.consoleConfig.out = out
	})
}

func ErrOutput(out io.Writer) Option {
	return optionFunc(func(c *config) {
		c.errOut = out
	})
}

func Graylog(addr string) Option {
	return optionFunc(func(c *config) {
		c.gelfConfig.Enabled = true
		c.gelfConfig.Addr = addr
	})
}

func GraylogLevel(lvl Level) Option {
	return optionFunc(func(c *config) {
		c.gelfConfig.Level = NewLevelValue(lvl)
	})
}
