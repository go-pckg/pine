# Pine

Opinionated logging library for Go.

## Installation

`go get -u github.com/go-pckg/pine`

## Getting Started

### Simple Logging Example

```go
package main

import (
    "github.com/go-pckg/pine"
)

func main() {
	logger := pine.New()

	logger.Info("hello world")
}

// Output: 2022-08-11T08:48:09+12:00 INF hello
```

### Error Logging

```go
package main

import (
	"errors"
	
    "github.com/go-pckg/pine"
)

func main() {
	logger := pine.New()

	err := errors.New("oops")
	logger.Error("we have a problem", pine.Err(err))
}

// Output: 2022-08-11T08:48:09+12:00 ERR we have a problem error=oops
```

### Extending Logger Fields

```go
package main

import (
    "github.com/go-pckg/pine"
)

func main() {
	logger := pine.New(pine.Fields(pine.Int("i", 1)))
	newLogger := logger.With(pine.String("s1", "A"))
	
	newLogger.Debug("hello", pine.String("s2", "B"))
}

// Output: 2022-08-11T08:48:09+12:00 DBG hello i=1 s1=A s2=B
```
