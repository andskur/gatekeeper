package redis

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/redis/go-redis/v9"
)

// clientSetWrapper implements ISetStr interface
type clientSetWrapper struct {
	clientWrapper
	setKey string
}

// Add string element to set without expire value
func (c clientSetWrapper) Add(ctx context.Context, val string) error {
	return c.AddExpire(ctx, val, -1)
}

// AddExpire add string element to set with expire value
func (c clientSetWrapper) AddExpire(ctx context.Context, val string, ttl time.Duration) error {
	// cleanup expired tokens
	cmd := c.client.ZRemRangeByScore(ctx, c.setKey, "-inf", fmt.Sprintf("%d", time.Now().UTC().Unix()-1))
	if cmd.Err() != nil {
		return fmt.Errorf("add expire: %w", coerceRedisErr(cmd.Err(), c.setKey))
	}

	var score float64
	if ttl == -1 {
		score = math.Inf(1)
	} else {
		score = float64(time.Now().Add(ttl).UTC().Unix())
	}

	return coerceRedisErr(c.client.ZAdd(ctx, c.setKey, redis.Z{
		Score:  score,
		Member: val,
	}).Err(), c.setKey)
}

// Remove remove element from string set
func (c clientSetWrapper) Remove(ctx context.Context, val string) error {
	cmd := c.client.ZRem(ctx, c.setKey, val)
	return coerceRedisErr(cmd.Err(), c.setKey)
}

// Check checks element int string set
func (c clientSetWrapper) Check(ctx context.Context, val string) (bool, error) {
	cmd := c.client.ZScore(ctx, c.setKey, val)
	if cmd.Err() != nil {
		return false, fmt.Errorf("check storage key: %w", coerceRedisErr(cmd.Err(), c.setKey))
	}

	return cmd.Val() >= float64(time.Now().UTC().Unix()), nil
}

// List all elements in string set
func (c clientSetWrapper) List(ctx context.Context) ([]string, error) {
	cmd := c.client.ZRangeByScore(ctx, c.setKey, &redis.ZRangeBy{
		Min:   fmt.Sprintf("%d", time.Now().UTC().Unix()),
		Max:   "+inf",
		Count: 100,
	})
	if cmd.Err() != nil {
		return nil, fmt.Errorf("storage element list: %w", coerceRedisErr(cmd.Err(), c.setKey))
	}

	return cmd.Val(), nil
}
