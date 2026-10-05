package a

import (
	"errors"
	"fmt"
)

// Good: test files may use one-off errors.
func helperForTests() error {
	_ = fmt.Errorf("one-off in a test")
	return errors.New("one-off in a test")
}
