package redis

import (
	"errors"
	"strings"

	"github.com/redis/go-redis/v9"

	nosql "github.com/andskur/gatekeeper/persistance"
)

// coerceRedisErr process redis error to return
func coerceRedisErr(err error, key string) error {
	switch {
	case err == nil:
		return nil
	case isNilErr(err):
		return nosql.NoKeyError(key)
	case isWrongOpErr(err):
		return nosql.ErrNotStrSet
	default:
		return err
	}
}

// isNilErr checks if error is nil
func isNilErr(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, redis.Nil)
}

// isWrongOpErr checks if type is wrong
func isWrongOpErr(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "wrong type")
}
