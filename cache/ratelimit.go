package cache

import (
	"context"
	"time"
)

// CheckRateLimit uses a Redis counter with expiry to implement a fixed-window rate limiter.
// Returns (allowed, remaining, error).
func (c *Client) CheckRateLimit(ctx context.Context, key string, max int, window time.Duration) (bool, int, error) {
	pipe := c.rdb.Pipeline()
	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, window)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return true, max, err
	}

	count := int(incr.Val())
	remaining := max - count
	if remaining < 0 {
		remaining = 0
	}

	return count <= max, remaining, nil
}
