package middleware

import (
	"context"
	"time"
)

// RateLimitResult holds the result of a GCRA rate limit check.
type RateLimitResult struct {
	// Allowed is true if the request is within the rate limit.
	Allowed bool
	// Limit is the maximum number of requests allowed per period (burst capacity).
	Limit int
	// Remaining is the estimated number of requests remaining in the current window.
	Remaining int
	// RetryAfter is the duration the client should wait before retrying (zero if allowed).
	RetryAfter time.Duration
	// ResetAt is the time when the rate limit fully resets (TAT).
	ResetAt time.Time
}

// RateLimitStore abstracts the backend for distributed rate limiting.
// Implementations must be safe for concurrent use.
type RateLimitStore interface {
	// Allow checks whether a request identified by key is allowed under the given
	// rate limit parameters using the GCRA algorithm.
	//
	// Parameters:
	//   - key: unique identifier for the rate limit subject (e.g., "ip:1.2.3.4")
	//   - rate: number of allowed requests per period
	//   - burst: maximum burst capacity (additional requests above sustained rate)
	//   - period: the time period for the rate (e.g., 60s means `rate` requests per 60s)
	Allow(ctx context.Context, key string, rate int, burst int, period time.Duration) (*RateLimitResult, error)
}
