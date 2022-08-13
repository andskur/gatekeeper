package persistance

import (
	"time"
)

// IStorage exposes simple key-value interface which wraps some NoSql DB.
// Also this interface expects that backend DB knows how the data will be serialized.
type IStorage interface {
	// Ping check connection to NoSql storage
	Ping() (string, error)

	// Get returns data associated with given key or return ErrNoSuchKeyFound
	Get(key string) (data interface{}, err error)

	// Set associates given key with given data
	Set(key string, data interface{}) error

	// SetWithExpire associates given key with given data for specified time
	SetWithExpire(key string, data interface{}, ttl time.Duration) error

	// Delete delete value associated with given key from storage
	// should return ErrNoSuchKey if nothing deleted
	Delete(key string) error

	// CountKeys count keys that match given pattern
	CountKeys(pattern string) (count int, err error)

	// StrSet returns strings set associated with given key.
	// This method doesn't checks either key presence nor
	// that value associated with key is set
	StrSet(key string) IStrSet

	// Close close current connection to NoSql storage
	Close() error
}

// IStrSet storage which operates with sets of strings providing
// most common the set data structure interface
type IStrSet interface {
	// Add elements to the set
	Add(val string) error

	// AddExpire same as Add but also set element expire time
	AddExpire(val string, ttl time.Duration) error

	// Remove element from the set
	Remove(val string) error

	// Check element presence
	Check(val string) (bool, error)

	// List returns all non-expired elements from the set
	List() ([]string, error)
}
