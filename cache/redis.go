package cache

import (
	"context"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type Client struct {
	rdb *redis.Client
	ttl time.Duration
}

func NewRedis() *Client {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}

	password := os.Getenv("REDIS_PASSWORD")

	db := 0
	if v := os.Getenv("REDIS_DB"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			db = n
		}
	}

	ttl := 3600 * time.Second
	if v := os.Getenv("CACHE_TTL"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			ttl = time.Duration(n) * time.Second
		}
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	return &Client{rdb: rdb, ttl: ttl}
}

// ServeImage checks cache, calls fetch on miss, writes image to response.
func (c *Client) ServeImage(
	w http.ResponseWriter,
	r *http.Request,
	key string,
	fetch func(string) ([]byte, string, error),
	identifier string,
) {
	ctx := context.Background()

	// Try cache hit: key -> content-type stored alongside as key:ct
	data, err := c.rdb.Get(ctx, key).Bytes()
	if err == nil {
		ct, _ := c.rdb.Get(ctx, key+":ct").Result()
		if ct == "" {
			ct = "image/jpeg"
		}
		w.Header().Set("Content-Type", ct)
		w.Header().Set("X-Cache", "HIT")
		w.WriteHeader(http.StatusOK)
		w.Write(data) //nolint:errcheck
		return
	}

	// Cache miss - fetch from source
	imgData, contentType, err := fetch(identifier)
	if err != nil {
		http.Error(w, "failed to fetch avatar: "+err.Error(), http.StatusBadGateway)
		return
	}

	// Store in cache
	pipe := c.rdb.Pipeline()
	pipe.Set(ctx, key, imgData, c.ttl)
	pipe.Set(ctx, key+":ct", contentType, c.ttl)
	pipe.Exec(ctx) //nolint:errcheck

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("X-Cache", "MISS")
	w.WriteHeader(http.StatusOK)
	w.Write(imgData) //nolint:errcheck
}
