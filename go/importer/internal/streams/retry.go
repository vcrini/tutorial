package streams

import (
	"context"
	"math"
	"math/rand"
	"time"
)

type RetryPolicy struct {
	Attempts     int
	MinBackoff   time.Duration
	MaxBackoff   time.Duration
	RandomFactor float64
}

func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		Attempts:     5,
		MinBackoff:   100 * time.Millisecond,
		MaxBackoff:   2 * time.Second,
		RandomFactor: 0.5,
	}
}

func Retry(ctx context.Context, policy RetryPolicy, fn func() error) error {
	attempts := policy.Attempts
	if attempts <= 0 {
		attempts = 1
	}

	minBackoff := policy.MinBackoff
	if minBackoff <= 0 {
		minBackoff = 100 * time.Millisecond
	}
	maxBackoff := policy.MaxBackoff
	if maxBackoff <= 0 {
		maxBackoff = 2 * time.Second
	}
	randomFactor := policy.RandomFactor

	var lastErr error
	for i := 0; i < attempts; i++ {
		if err := fn(); err != nil {
			lastErr = err
			if i == attempts-1 {
				break
			}

			backoff := time.Duration(float64(minBackoff) * math.Pow(2, float64(i)))
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			if randomFactor > 0 {
				jitter := 1 + randomFactor*(rand.Float64()*2-1)
				backoff = time.Duration(float64(backoff) * jitter)
				if backoff < 0 {
					backoff = minBackoff
				}
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
			continue
		}
		return nil
	}

	return lastErr
}
