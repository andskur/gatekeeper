package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	nosql "github.com/andskur/sessions/persistance"
)

// New creates nosql.IStorage wrapper
func New(options *redis.Options) nosql.IStorage {
	return clientWrapper{
		client: redis.NewClient(options),
	}
}

// clientWrapper wraps redis universal client (e.g. for both single and cluster modes)
type clientWrapper struct {
	client *redis.Client
}

// Ping sent ping signal to current server and checks connection
func (c clientWrapper) Ping(ctx context.Context) (string, error) {
	pong, err := c.client.Ping(ctx).Result()
	if err != nil {
		return "", fmt.Errorf("redis ping: %w", err)
	}
	return pong, nil
}

// Get gets redis key using GET cmd, trying to unmarshal json into interface{}
func (c clientWrapper) Get(ctx context.Context, key string) (data interface{}, err error) {
	cmd := c.client.Get(ctx, key)
	if cmd.Err() != nil {
		if isNilErr(cmd.Err()) {
			return nil, nosql.NoKeyError(key)
		}
		err = fmt.Errorf("redis get: %w", cmd.Err())
		return
	}

	bytes, _ := cmd.Bytes()
	if bytes == nil {
		return nil, nosql.NoKeyError(key)
	}
	if len(bytes) == 0 {
		return
	}

	err = json.Unmarshal(bytes, &data)
	return
}

// Set sets redis key value marshaling it's value using json
func (c clientWrapper) Set(ctx context.Context, key string, data interface{}) error {
	return c.SetWithExpire(ctx, key, data, 0)
}

// SetWithExpire same as Set but with expiration
func (c clientWrapper) SetWithExpire(ctx context.Context, key string, data interface{}, ttl time.Duration) error {
	bytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("set with expire: %w", err)
	}
	res := c.client.Set(ctx, key, bytes, ttl)
	return res.Err()
}

// Delete deletes key from redis
func (c clientWrapper) Delete(ctx context.Context, key string) error {
	res := c.client.Del(ctx, key)
	if res, err := res.Result(); err != nil || res == 0 {
		if err != nil {
			return fmt.Errorf("key delete: %w", err)
		}
		return nosql.NoKeyError(key)
	}
	return nil
}

// CountKeys count keys that match given pattern
func (c clientWrapper) CountKeys(ctx context.Context, pattern string) (count int, err error) {
	var cursor uint64
	for {
		var keys []string
		keys, cursor, err = c.client.Scan(ctx, cursor, pattern, 10).Result()
		if err != nil {
			return 0, fmt.Errorf("count keys: %w", err)
		}

		count += len(keys)
		if cursor == 0 {
			break
		}
	}
	return
}

// SrtSet
func (c clientWrapper) StrSet(key string) nosql.IStrSet {
	return clientSetWrapper{clientWrapper: c, setKey: key}
}

// Close implements io.Closer interface
func (c clientWrapper) Close() error {
	return c.client.Close()
}
