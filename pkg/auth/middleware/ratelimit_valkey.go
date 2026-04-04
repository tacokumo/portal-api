package middleware

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/tacokumo/portal-api/pkg/auth/session"
	"github.com/valkey-io/valkey-go"
)

// gcraScript is a Lua script that implements the GCRA (Generic Cell Rate Algorithm)
// atomically on the Valkey server.
//
// == Why a Lua script? ==
// GCRA requires a read-modify-write (RMW) sequence: read the current TAT (Theoretical
// Arrival Time), compute the new TAT, and conditionally write it back. If this were done
// as separate GET + SET commands from Go, concurrent requests could read the same TAT
// and both be allowed, violating the rate limit. Valkey executes Lua scripts atomically
// (single-threaded), so the entire RMW operation is serialized without race conditions.
//
// == Algorithm overview ==
// GCRA models a "leaky bucket" that drains at a constant rate. Each request advances
// the TAT (the time the bucket will next be empty) by one emission_interval. A request
// is allowed if the new TAT does not exceed the current time plus the burst_offset
// (which represents the maximum amount of "debt" the bucket can accumulate).
//
// == Parameters ==
//
//	KEYS[1]: the rate limit key (e.g., "portal:rate_limit:ip:1.2.3.4")
//	ARGV[1]: now           - current time in microseconds (int64)
//	ARGV[2]: emission_interval - microseconds between allowed requests (period / rate)
//	ARGV[3]: burst_offset  - maximum burst window in microseconds (emission_interval * burst)
//	ARGV[4]: ttl_ms        - key TTL in milliseconds for auto-expiry
//
// == Return values ==
//
//	[1]: allowed    - 1 if request is allowed, 0 if rejected
//	[2]: remaining  - estimated remaining requests before the limit is hit
//	[3]: retry_after_us - microseconds until the request would be allowed ("0" if allowed)
//	[4]: reset_at_us    - TAT in microseconds (time when limit fully resets)
//
// Note: Large microsecond values are formatted with string.format("%.0f", ...) to avoid
// Lua's default scientific notation (e.g., "1.77e+15") which cannot be parsed by Go's
// strconv.ParseInt.
const gcraScript = `
local tat_raw = redis.call("GET", KEYS[1])
local tat = 0
if tat_raw then
  tat = tonumber(tat_raw)
end

local now = tonumber(ARGV[1])
local emission_interval = tonumber(ARGV[2])
local burst_offset = tonumber(ARGV[3])
local ttl_ms = tonumber(ARGV[4])

local new_tat = math.max(tat, now) + emission_interval
local allow_at = new_tat - burst_offset

if allow_at > now then
  local retry_after_us = allow_at - now
  return {0, 0, string.format("%.0f", retry_after_us), string.format("%.0f", new_tat)}
end

redis.call("SET", KEYS[1], string.format("%.0f", new_tat), "PX", ttl_ms)
local diff = burst_offset - (new_tat - now)
local remaining = 0
if diff > 0 and emission_interval > 0 then
  remaining = math.floor(diff / emission_interval)
end
return {1, remaining, "0", string.format("%.0f", new_tat)}
`

// ValkeyRateLimitStore implements RateLimitStore using Valkey with the GCRA algorithm.
type ValkeyRateLimitStore struct {
	client valkey.Client
}

// NewValkeyRateLimitStore creates a new GCRA-based rate limit store.
// The provided valkey.Client is safe for concurrent use and can be shared
// with other components (e.g., session manager).
func NewValkeyRateLimitStore(client valkey.Client) *ValkeyRateLimitStore {
	return &ValkeyRateLimitStore{client: client}
}

// Allow checks whether a request is allowed under the GCRA rate limit.
// See the gcraScript constant for the full algorithm documentation.
func (s *ValkeyRateLimitStore) Allow(ctx context.Context, key string, rate int, burst int, period time.Duration) (*RateLimitResult, error) {
	now := time.Now()
	nowUs := now.UnixMicro()

	// emission_interval: time between each allowed request
	emissionIntervalUs := period.Microseconds() / int64(rate)
	// burst_offset: how far ahead TAT can be from now (burst window)
	burstOffsetUs := emissionIntervalUs * int64(burst)
	// TTL: enough to cover burst_offset + one emission_interval, in milliseconds
	ttlMs := (burstOffsetUs + emissionIntervalUs) / 1000
	if ttlMs < 1000 {
		ttlMs = 1000 // minimum 1 second TTL
	}

	fullKey := session.KeyPrefixRateLimit + key

	cmd := s.client.B().Eval().
		Script(gcraScript).
		Numkeys(1).
		Key(fullKey).
		Arg(
			strconv.FormatInt(nowUs, 10),
			strconv.FormatInt(emissionIntervalUs, 10),
			strconv.FormatInt(burstOffsetUs, 10),
			strconv.FormatInt(ttlMs, 10),
		).Build()

	resp := s.client.Do(ctx, cmd)
	if err := resp.Error(); err != nil {
		return nil, fmt.Errorf("GCRA eval failed: %w", err)
	}

	arr, err := resp.ToArray()
	if err != nil {
		return nil, fmt.Errorf("GCRA response parse failed: %w", err)
	}
	if len(arr) != 4 {
		return nil, fmt.Errorf("GCRA unexpected response length: %d", len(arr))
	}

	allowed, err := arr[0].ToInt64()
	if err != nil {
		return nil, fmt.Errorf("GCRA parse allowed: %w", err)
	}

	remaining, err := arr[1].ToInt64()
	if err != nil {
		return nil, fmt.Errorf("GCRA parse remaining: %w", err)
	}

	retryAfterStr, err := arr[2].ToString()
	if err != nil {
		return nil, fmt.Errorf("GCRA parse retry_after: %w", err)
	}
	retryAfterUs, _ := strconv.ParseInt(retryAfterStr, 10, 64)

	resetAtStr, err := arr[3].ToString()
	if err != nil {
		return nil, fmt.Errorf("GCRA parse reset_at: %w", err)
	}
	resetAtUs, _ := strconv.ParseInt(resetAtStr, 10, 64)

	return &RateLimitResult{
		Allowed:    allowed == 1,
		Limit:      burst,
		Remaining:  int(remaining),
		RetryAfter: time.Duration(retryAfterUs) * time.Microsecond,
		ResetAt:    time.UnixMicro(resetAtUs),
	}, nil
}
