package retry

import (
	"context"
	"math"
	"math/rand"
	"time"
)

type ExponentialBackoff struct {
	InitialInterval     time.Duration
	MaxInterval         time.Duration
	MaxElapsedTime      time.Duration
	Multiplier          float64
	RandomizationFactor float64
	MaxRetries          int
}

type Retryer interface {
	Do(ctx context.Context, fn func() error) error
}

func NewExponentialBackoff(config ExponentialBackoffConfig) *ExponentialBackoff {
	return &ExponentialBackoff{
		InitialInterval:     config.InitialInterval,
		MaxInterval:         config.MaxInterval,
		MaxElapsedTime:      config.MaxElapsedTime,
		Multiplier:          config.Multiplier,
		RandomizationFactor: config.RandomizationFactor,
		MaxRetries:          config.MaxRetries,
	}
}

type ExponentialBackoffConfig struct {
	InitialInterval     time.Duration
	MaxInterval         time.Duration
	MaxElapsedTime      time.Duration
	Multiplier          float64
	RandomizationFactor float64
	MaxRetries          int
}

func DefaultExponentialBackoffConfig() ExponentialBackoffConfig {
	return ExponentialBackoffConfig{
		InitialInterval:     1 * time.Second,
		MaxInterval:         30 * time.Second,
		MaxElapsedTime:      5 * time.Minute,
		Multiplier:          2.0,
		RandomizationFactor: 0.1,
		MaxRetries:          3,
	}
}

func (eb *ExponentialBackoff) Do(ctx context.Context, fn func() error) error {
	var lastErr error
	currentInterval := eb.InitialInterval
	startTime := time.Now()

	for attempt := 0; attempt <= eb.MaxRetries; attempt++ {
		if err := fn(); err == nil {
			return nil
		} else {
			lastErr = err
		}

		if attempt == eb.MaxRetries {
			break
		}

		if time.Since(startTime) >= eb.MaxElapsedTime {
			break
		}

		if currentInterval > eb.MaxInterval {
			currentInterval = eb.MaxInterval
		}

		jitter := eb.getJitter(currentInterval)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(jitter):
		}

		currentInterval = time.Duration(float64(currentInterval) * eb.Multiplier)
	}

	return lastErr
}

func (eb *ExponentialBackoff) getJitter(interval time.Duration) time.Duration {
	if eb.RandomizationFactor == 0 {
		return interval
	}

	delta := eb.RandomizationFactor * float64(interval)
	minInterval := float64(interval) - delta
	maxInterval := float64(interval) + delta

	jitter := minInterval + (rand.Float64() * (maxInterval - minInterval))

	return time.Duration(math.Max(0, jitter))
}
