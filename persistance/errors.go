package persistance

import (
	"errors"
	"fmt"
)

// NoSql storage error
var (
	ErrNoSuchKeyFound = errors.New("no such key found")
	ErrNotStrSet      = errors.New("not a strings set")
)

// NoKeyError format 'no such key; error
func NoKeyError(key string) error {
	return fmt.Errorf("%s - %s", ErrNoSuchKeyFound, key)
}
