package memory

import (
	"context"
	"strings"
	"sync"
	"time"

	nosql "github.com/andskur/gatekeeper/persistance"
)

// Storage is an in-memory implementation of IStorage for tests and lightweight use.
type Storage struct {
	mu   sync.RWMutex
	data map[string]interface{}
	sets map[string]*strSet
}

// New creates a new in-memory storage instance.
func New() *Storage {
	return &Storage{
		data: make(map[string]interface{}),
		sets: make(map[string]*strSet),
	}
}

func (s *Storage) Ping(ctx context.Context) (string, error) {
	return "PONG", nil
}

func (s *Storage) Get(ctx context.Context, key string) (data interface{}, err error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.data[key]
	if !ok {
		return nil, nosql.NoKeyError(key)
	}
	return v, nil
}

func (s *Storage) Set(ctx context.Context, key string, data interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = data
	return nil
}

func (s *Storage) SetWithExpire(ctx context.Context, key string, data interface{}, ttl time.Duration) error {
	s.mu.Lock()
	s.data[key] = data
	s.mu.Unlock()

	if ttl > 0 {
		go func() {
			select {
			case <-ctx.Done():
				return
			case <-time.After(ttl):
				_ = s.Delete(context.Background(), key)
			}
		}()
	}
	return nil
}

func (s *Storage) Delete(ctx context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data[key]; !ok {
		return nosql.NoKeyError(key)
	}
	delete(s.data, key)
	return nil
}

func (s *Storage) CountKeys(ctx context.Context, pattern string) (count int, err error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for k := range s.data {
		if strings.Contains(k, pattern) {
			count++
		}
	}
	return
}

func (s *Storage) StrSet(key string) nosql.IStrSet {
	s.mu.Lock()
	defer s.mu.Unlock()
	set, ok := s.sets[key]
	if !ok {
		set = &strSet{items: make(map[string]time.Time)}
		s.sets[key] = set
	}
	return set
}

func (s *Storage) Close() error { return nil }

// strSet implements IStrSet in memory.
type strSet struct {
	mu    sync.RWMutex
	items map[string]time.Time // zero time -> infinite
}

func (s *strSet) Add(ctx context.Context, val string) error {
	return s.AddExpire(ctx, val, -1)
}

func (s *strSet) AddExpire(ctx context.Context, val string, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	if ttl == -1 {
		s.items[val] = time.Time{}
	} else {
		s.items[val] = now.Add(ttl)
	}

	// cleanup expired
	for k, exp := range s.items {
		if !exp.IsZero() && exp.Before(now) {
			delete(s.items, k)
		}
	}
	return nil
}

func (s *strSet) Remove(ctx context.Context, val string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[val]; !ok {
		return nosql.ErrNoSuchKeyFound
	}
	delete(s.items, val)
	return nil
}

func (s *strSet) Check(ctx context.Context, val string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	exp, ok := s.items[val]
	if !ok {
		return false, nosql.ErrNoSuchKeyFound
	}
	if !exp.IsZero() && exp.Before(time.Now()) {
		return false, nosql.ErrNoSuchKeyFound
	}
	return true, nil
}

func (s *strSet) List(ctx context.Context) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	now := time.Now()
	res := make([]string, 0, len(s.items))
	for k, exp := range s.items {
		if exp.IsZero() || exp.After(now) {
			res = append(res, k)
		}
	}
	return res, nil
}
