package gemini

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"net"
	"strings"
	"time"
)

// RetryConfig controls the retry behaviour for API requests.
type RetryConfig struct {
	// MaxAttempts is the maximum number of attempts (including the first).
	MaxAttempts int
	// InitialDelay is the base delay before the first retry.
	InitialDelay time.Duration
	// MaxDelay is the maximum delay between retries.
	MaxDelay time.Duration
	// JitterFraction adds randomness to delay: +/- this fraction of the delay.
	JitterFraction float64
	// OnPersistent429 is called when 429 errors persist beyond Persistent429Threshold.
	// Return false to stop retrying immediately.
	OnPersistent429 func(attempt int) bool
	// Persistent429Threshold is the number of consecutive 429s before calling OnPersistent429.
	Persistent429Threshold int
}

// DefaultRetryConfig returns the default retry configuration.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:            10,
		InitialDelay:           2 * time.Second,
		MaxDelay:               30 * time.Second,
		JitterFraction:         0.2,
		OnPersistent429:        nil,
		Persistent429Threshold: 5,
	}
}

// RetryOption is a functional option for configuring retry behaviour.
type RetryOption func(*RetryConfig)

// WithMaxAttempts sets the maximum number of retry attempts.
func WithMaxAttempts(n int) RetryOption {
	return func(c *RetryConfig) { c.MaxAttempts = n }
}

// WithInitialDelay sets the initial backoff delay.
func WithInitialDelay(d time.Duration) RetryOption {
	return func(c *RetryConfig) { c.InitialDelay = d }
}

// WithMaxDelay sets the maximum backoff delay.
func WithMaxDelay(d time.Duration) RetryOption {
	return func(c *RetryConfig) { c.MaxDelay = d }
}

// WithJitter sets the jitter fraction for backoff delays.
func WithJitter(fraction float64) RetryOption {
	return func(c *RetryConfig) { c.JitterFraction = fraction }
}

// WithOnPersistent429 sets the callback for persistent 429 errors.
func WithOnPersistent429(fn func(attempt int) bool) RetryOption {
	return func(c *RetryConfig) { c.OnPersistent429 = fn }
}

// APIError represents an HTTP error from the Gemini API.
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("gemini API error %d: %s", e.StatusCode, e.Message)
}

// IsRetryable returns true if the error is transient and should be retried.
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	// Check for API errors with retryable status codes.
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		switch apiErr.StatusCode {
		case 429: // Rate limited
			return true
		case 500, 502, 503: // Server errors
			return true
		case 400, 401, 403, 404: // Client errors -- not retryable
			return false
		}
	}

	// Check for network errors.
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}

	// Check for known transient error strings.
	msg := err.Error()
	transientPatterns := []string{
		"ECONNRESET",
		"ETIMEDOUT",
		"EPIPE",
		"ENOTFOUND",
		"connection reset",
		"broken pipe",
		"i/o timeout",
		"TLS handshake timeout",
		"EOF",
	}
	for _, pattern := range transientPatterns {
		if strings.Contains(msg, pattern) {
			return true
		}
	}

	return false
}

// Is429 returns true if the error is a 429 rate limit error.
func Is429(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == 429
}

// exponentialBackoff calculates the backoff delay for the given attempt.
func exponentialBackoff(attempt int, cfg RetryConfig) time.Duration {
	// delay = min(initialDelay * 2^attempt, maxDelay)
	delay := float64(cfg.InitialDelay) * math.Pow(2, float64(attempt))
	if delay > float64(cfg.MaxDelay) {
		delay = float64(cfg.MaxDelay)
	}

	// Apply jitter: +/- jitterFraction * delay
	if cfg.JitterFraction > 0 {
		jitter := delay * cfg.JitterFraction
		delay = delay - jitter + (rand.Float64() * 2 * jitter)
	}

	return time.Duration(delay)
}

// ExecuteWithRetry runs the given function with retry logic.
// It retries on transient errors with exponential backoff and jitter.
// Returns the result of the last successful call or the last error.
func ExecuteWithRetry[T any](ctx context.Context, cfg RetryConfig, fn func(ctx context.Context) (T, error)) (T, error) {
	var lastErr error
	var zero T
	consecutive429 := 0

	for attempt := range cfg.MaxAttempts {
		// Check context before each attempt.
		if ctx.Err() != nil {
			return zero, fmt.Errorf("retry cancelled: %w", ctx.Err())
		}

		result, err := fn(ctx)
		if err == nil {
			return result, nil
		}

		lastErr = err

		if !IsRetryable(err) {
			return zero, fmt.Errorf("non-retryable error (attempt %d/%d): %w", attempt+1, cfg.MaxAttempts, err)
		}

		// Track consecutive 429s.
		if Is429(err) {
			consecutive429++
		} else {
			consecutive429 = 0
		}

		// Check persistent 429 threshold.
		if consecutive429 >= cfg.Persistent429Threshold && cfg.OnPersistent429 != nil {
			if !cfg.OnPersistent429(attempt + 1) {
				return zero, fmt.Errorf("persistent 429 after %d attempts, user stopped retry: %w", consecutive429, err)
			}
		}

		// Don't sleep after the last attempt.
		if attempt < cfg.MaxAttempts-1 {
			delay := exponentialBackoff(attempt, cfg)
			select {
			case <-time.After(delay):
				// Continue to next attempt.
			case <-ctx.Done():
				return zero, fmt.Errorf("retry cancelled during backoff: %w", ctx.Err())
			}
		}
	}

	return zero, fmt.Errorf("max retries (%d) exceeded: %w", cfg.MaxAttempts, lastErr)
}

// FallbackChain defines a sequence of models to try when the primary fails.
type FallbackChain struct {
	Models []string
}

// DefaultFallbackChains returns the standard fallback chains.
var DefaultFallbackChains = map[string][]string{
	"gemini-3.1-pro-preview":         {"gemini-2.5-pro"},
	"gemini-3.1-flash":               {"gemini-2.5-flash"},
	"gemini-3.1-flash-image-preview": {}, // No fallback for image gen
	"gemini-2.5-pro":                 {},  // Already fallback tier
	"gemini-2.5-flash":               {},  // Already fallback tier
}

// GetFallbackModels returns the fallback models for the given primary model.
func GetFallbackModels(modelID string) []string {
	chain, ok := DefaultFallbackChains[modelID]
	if !ok {
		return nil
	}
	return chain
}
