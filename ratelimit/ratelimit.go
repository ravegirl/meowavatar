package ratelimit

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"meowavatar/cache"
)

type Limiter struct {
	cache  *cache.Client
	max    int
	window time.Duration
}

func New(c *cache.Client) *Limiter {
	max := 60
	if v := os.Getenv("RATELIMIT_MAX"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			max = n
		}
	}

	window := 60 * time.Second
	if v := os.Getenv("RATELIMIT_WINDOW"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			window = time.Duration(n) * time.Second
		}
	}

	return &Limiter{cache: c, max: max, window: window}
}

func (l *Limiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := realIP(r)
		key := fmt.Sprintf("rl:%s", ip)

		allowed, remaining, err := l.cache.CheckRateLimit(context.Background(), key, l.max, l.window)
		if err != nil {
			// On Redis error, fail open (allow request)
			next.ServeHTTP(w, r)
			return
		}

		w.Header().Set("X-RateLimit-Limit", strconv.Itoa(l.max))
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))

		if !allowed {
			w.Header().Set("Retry-After", strconv.Itoa(int(l.window.Seconds())))
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// realIP extracts the real client IP, respecting common proxy headers.
func realIP(r *http.Request) string {
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	if ip := r.Header.Get("CF-Connecting-IP"); ip != "" {
		return ip
	}
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		// X-Forwarded-For can be a comma-separated list; first entry is the client
		if host, _, err := net.SplitHostPort(ip); err == nil {
			return host
		}
		return ip
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
