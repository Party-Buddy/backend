package util

import "fmt"

// Must panics if err is not nil.
func Must[T any](value T, err error) T {
	if err != nil {
		panic(fmt.Sprintf("assertion failed: %s", err))
	}

	return value
}
