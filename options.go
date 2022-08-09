package pine

import "io"

type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (f optionFunc) apply(log *config) {
	f(log)
}

func WithLevel(lvl Level) Option {
	return optionFunc(func(log *config) {
		log.level = NewLevelValue(lvl)
	})
}

func WithLevelValue(lvl *LevelValue) Option {
	return optionFunc(func(log *config) {
		log.level = lvl
	})
}

func Colored(useColors bool) Option {
	return optionFunc(func(log *config) {
		log.UseColors = useColors
	})
}

func WithColors() Option {
	return optionFunc(func(log *config) {
		log.UseColors = true
	})
}

func NoColors() Option {
	return optionFunc(func(log *config) {
		log.UseColors = false
	})
}

func WithStackTraceLevel(lvl Level) Option {
	return optionFunc(func(log *config) {
		log.stackTraceLevel = NewLevelValue(lvl)
	})
}

func ForceQuote() Option {
	return optionFunc(func(log *config) {
		log.ForceQuote = true
	})
}

func QuoteEmptyFields() Option {
	return optionFunc(func(log *config) {
		log.QuoteEmptyFields = true
	})
}

func NoQuotes() Option {
	return optionFunc(func(log *config) {
		log.DisableQuote = true
	})
}

func Fields(fields ...Field) Option {
	return optionFunc(func(log *config) {
		for i := range fields {
			log.fields[fields[i].key] = fields[i]
		}
	})
}

func Development(enabled bool) Option {
	return optionFunc(func(log *config) {
		log.development = enabled
	})
}

func AddCaller() Option {
	return WithCaller(true)
}

func WithCaller(enabled bool) Option {
	return optionFunc(func(log *config) {
		log.ReportCaller = enabled
	})
}

func NoSorting() Option {
	return optionFunc(func(log *config) {
		log.DisableSorting = true
	})
}

func Sorting(disabled bool) Option {
	return optionFunc(func(log *config) {
		log.DisableSorting = disabled
	})
}

func WithClock(clock Clock) Option {
	return optionFunc(func(log *config) {
		log.clock = clock
	})
}

func Output(out io.Writer) Option {
	return optionFunc(func(log *config) {
		log.out = out
	})
}

func ErrOutput(out io.Writer) Option {
	return optionFunc(func(log *config) {
		log.errOut = out
	})
}
