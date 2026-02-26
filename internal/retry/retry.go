package retry

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"
)

// Config holds retry configuration
type Config struct {
	MaxRetries     int
	InitialDelay   time.Duration
	MaxDelay       time.Duration
	Multiplier     float64
	JitterFraction float64
}

// DefaultConfig returns the default retry configuration
// Initial delay: 100ms, Max delay: 10s, Max retries: 3
func DefaultConfig() Config {
	return Config{
		MaxRetries:     3,
		InitialDelay:   100 * time.Millisecond,
		MaxDelay:       10 * time.Second,
		Multiplier:     2.0,
		JitterFraction: 0.1,
	}
}

// Retryer handles retry logic with exponential backoff
type Retryer struct {
	config Config
}

// NewRetryer creates a new retryer with the given configuration
func NewRetryer(config Config) *Retryer {
	return &Retryer{config: config}
}

// NewDefaultRetryer creates a new retryer with default configuration
func NewDefaultRetryer() *Retryer {
	return NewRetryer(DefaultConfig())
}

// Do executes the given function with retry logic
// Returns the result of the function or the last error encountered
func (r *Retryer) Do(ctx context.Context, fn func() error) error {
	var lastErr error

	for attempt := 0; attempt <= r.config.MaxRetries; attempt++ {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			return fmt.Errorf("retry cancelled: %w", ctx.Err())
		default:
		}

		// Execute the function
		err := fn()
		if err == nil {
			return nil // Success
		}

		lastErr = err

		// Check if error is retryable
		if !IsRetryable(err) {
			return fmt.Errorf("non-retryable error: %w", err)
		}

		// Don't sleep after the last attempt
		if attempt == r.config.MaxRetries {
			break
		}

		// Calculate delay with exponential backoff and jitter
		delay := r.calculateDelay(attempt)

		// Sleep before next retry
		select {
		case <-ctx.Done():
			return fmt.Errorf("retry cancelled during backoff: %w", ctx.Err())
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	return fmt.Errorf("max retries (%d) exceeded: %w", r.config.MaxRetries, lastErr)
}

// DoWithResult executes the given function with retry logic and returns a result
func (r *Retryer) DoWithResult(ctx context.Context, fn func() (interface{}, error)) (interface{}, error) {
	var lastErr error
	var result interface{}

	for attempt := 0; attempt <= r.config.MaxRetries; attempt++ {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("retry cancelled: %w", ctx.Err())
		default:
		}

		// Execute the function
		res, err := fn()
		if err == nil {
			return res, nil // Success
		}

		lastErr = err

		// Check if error is retryable
		if !IsRetryable(err) {
			return nil, fmt.Errorf("non-retryable error: %w", err)
		}

		// Don't sleep after the last attempt
		if attempt == r.config.MaxRetries {
			break
		}

		// Calculate delay with exponential backoff and jitter
		delay := r.calculateDelay(attempt)

		// Sleep before next retry
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("retry cancelled during backoff: %w", ctx.Err())
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	return result, fmt.Errorf("max retries (%d) exceeded: %w", r.config.MaxRetries, lastErr)
}

// calculateDelay calculates the delay for the given attempt with exponential backoff and jitter
func (r *Retryer) calculateDelay(attempt int) time.Duration {
	// Calculate exponential backoff: initialDelay * multiplier^attempt
	delay := float64(r.config.InitialDelay) * math.Pow(r.config.Multiplier, float64(attempt))

	// Cap at max delay
	if delay > float64(r.config.MaxDelay) {
		delay = float64(r.config.MaxDelay)
	}

	// Add jitter to prevent thundering herd
	jitter := delay * r.config.JitterFraction * (rand.Float64()*2 - 1) // Random value between -jitterFraction and +jitterFraction
	delay += jitter

	// Ensure delay is positive
	if delay < 0 {
		delay = float64(r.config.InitialDelay)
	}

	return time.Duration(delay)
}

// Retry is a convenience function that uses the default retryer
func Retry(ctx context.Context, fn func() error) error {
	return NewDefaultRetryer().Do(ctx, fn)
}

// RetryWithResult is a convenience function that uses the default retryer and returns a result
func RetryWithResult(ctx context.Context, fn func() (interface{}, error)) (interface{}, error) {
	return NewDefaultRetryer().DoWithResult(ctx, fn)
}
