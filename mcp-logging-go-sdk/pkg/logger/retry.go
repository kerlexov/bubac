package logger

import (
	"context"
	"math"
	"math/rand"
	"time"
)

type retryer struct {
	config RetryConfig
}

func newRetryer(config RetryConfig) *retryer {
	return &retryer{config: config}
}

func (r *retryer) Do(ctx context.Context, fn func() error) error {
	var lastErr error
	currentInterval := r.config.InitialInterval
	startTime := time.Now()
	maxRetries := 3

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if err := fn(); err == nil {
			return nil
		} else {
			lastErr = err
		}

		if attempt == maxRetries {
			break
		}

		if time.Since(startTime) >= r.config.MaxElapsedTime {
			break
		}

		if currentInterval > r.config.MaxInterval {
			currentInterval = r.config.MaxInterval
		}

		jitter := r.getJitter(currentInterval)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(jitter):
		}

		currentInterval = time.Duration(float64(currentInterval) * r.config.Multiplier)
	}

	return lastErr
}

func (r *retryer) getJitter(interval time.Duration) time.Duration {
	if r.config.RandomizationFactor == 0 {
		return interval
	}

	delta := r.config.RandomizationFactor * float64(interval)
	minInterval := float64(interval) - delta
	maxInterval := float64(interval) + delta

	jitter := minInterval + (rand.Float64() * (maxInterval - minInterval))

	return time.Duration(math.Max(0, jitter))
}
