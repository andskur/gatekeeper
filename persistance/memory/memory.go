package memory

import (
	"time"

	nosql "github.com/andskur/gatekeeper/persistance"
)

func New() nosql.IStorage {
	return &inMemory{
		storage: make(map[string]any),
	}
}

type inMemory struct {
	storage map[string]any
}

func (i *inMemory) Ping() (string, error) {
	return "pong", nil
}

func (i *inMemory) Get(key string) (data any, err error) {
	data, ok := i.storage[key]
	if !ok {
		return nil, nosql.NoKeyError(key)
	}

	return data, nil
}

func (i *inMemory) Set(key string, data any) error {
	i.storage[key] = data

	return nil
}

func (i *inMemory) SetWithExpire(key string, data interface{}, ttl time.Duration) error {
	//TODO implement me
	panic("implement me")
}

func (i *inMemory) Delete(key string) error {
	if _, ok := i.storage[key]; !ok {
		return nosql.NoKeyError(key)
	}

	delete(i.storage, key)
	return nil
}

func (i *inMemory) CountKeys(pattern string) (count int, err error) {
	//TODO implement me
	panic("implement me")
}

func (i *inMemory) StrSet(key string) nosql.IStrSet {
	//TODO implement me
	panic("implement me")
}

func (i *inMemory) Close() error {
	return nil
}
