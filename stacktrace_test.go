package pine

import "github.com/pkg/errors"

func outer() error {
	return inner()
}

func inner() error {
	return errors.New("test")
}
