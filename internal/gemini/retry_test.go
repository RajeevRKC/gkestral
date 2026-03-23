package gemini

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"
)

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "429", err: &APIError{StatusCode: 429, Message: "rate limited"}, want: true},
		{name: "500", err: &APIError{StatusCode: 500, Message: "server error"}, want: true},
		{name: "502", err: &APIError{StatusCode: 502, Message: "bad gateway"}, want: true},
		{name: "503", err: &APIError{StatusCode: 503, Message: "unavailable"}, want: true},
		{name: "400", err: &APIError{StatusCode: 400, Message: "bad request"}, want: false},
		{name: "401", err: &APIError{StatusCode: 401, Message: "unauthorized"}, want: false},
		{name: "403", err: &APIError{StatusCode: 403, Message: "forbidden"}, want: false},
		{name: "404", err: &APIError{StatusCode: 404, Message: "not found"}, want: false},
		{name: "connection reset", err: errors.New("connection reset by peer"), want: true},
		{name: "timeout", err: errors.New("i/o timeout"), want: true},
		{name: "EOF", err: errors.New("unexpected EOF"), want: true},
		{name: "broken pipe", err: errors.New("broken pipe"), want: true},
		{name: "generic error", err: errors.New("something went wrong"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsRetryable(tt.err)
			if got != tt.want {
				t.Errorf("IsRetryable(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestIs429(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "429 error", err: &APIError{StatusCode: 429}, want: true},
		{name: "500 error", err: &APIError{StatusCode: 500}, want: false},
		{name: "generic error", err: errors.New("not an API error"), want: false},
		{name: "nil", err: nil, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Is429(tt.err)
			if got != tt.want {
				t.Errorf("Is429(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestExponentialBackoff(t *testing.T) {
	cfg := RetryConfig{
		InitialDelay:   1 * time.Second,
		MaxDelay:       10 * time.Second,
		JitterFraction: 0, // No jitter for deterministic tests
	}

	tests := []struct {
		attempt  int
		wantMin  time.Duration
		wantMax  time.Duration
	}{
		{attempt: 0, wantMin: 1 * time.Second, wantMax: 1 * time.Second},
		{attempt: 1, wantMin: 2 * time.Second, wantMax: 2 * time.Second},
		{attempt: 2, wantMin: 4 * time.Second, wantMax: 4 * time.Second},
		{attempt: 3, wantMin: 8 * time.Second, wantMax: 8 * time.Second},
		{attempt: 4, wantMin: 10 * time.Second, wantMax: 10 * time.Second}, // Capped at maxDelay
		{attempt: 10, wantMin: 10 * time.Second, wantMax: 10 * time.Second},
	}

	for _, tt := range tests {
		delay := exponentialBackoff(tt.attempt, cfg)
		if delay < tt.wantMin || delay > tt.wantMax {
			t.Errorf("exponentialBackoff(%d) = %v, want [%v, %v]", tt.attempt, delay, tt.wantMin, tt.wantMax)
		}
	}
}

func TestExponentialBackoff_WithJitter(t *testing.T) {
	cfg := RetryConfig{
		InitialDelay:   1 * time.Second,
		MaxDelay:       30 * time.Second,
		JitterFraction: 0.2,
	}

	// With 20% jitter on 1s delay, expect 0.8s - 1.2s
	seen := make(map[time.Duration]bool)
	for range 20 {
		delay := exponentialBackoff(0, cfg)
		seen[delay] = true
		if delay < 800*time.Millisecond || delay > 1200*time.Millisecond {
			t.Errorf("exponentialBackoff with jitter = %v, want [0.8s, 1.2s]", delay)
		}
	}
	// Should see variance (not all identical)
	if len(seen) < 2 {
		t.Error("Jitter produced no variance across 20 samples")
	}
}

func TestExecuteWithRetry_Success(t *testing.T) {
	cfg := DefaultRetryConfig()
	cfg.InitialDelay = 1 * time.Millisecond // Fast for tests
	cfg.MaxAttempts = 3

	calls := 0
	result, err := ExecuteWithRetry(context.Background(), cfg, func(_ context.Context) (string, error) {
		calls++
		return "ok", nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "ok" {
		t.Errorf("result = %q, want %q", result, "ok")
	}
	if calls != 1 {
		t.Errorf("calls = %d, want 1", calls)
	}
}

func TestExecuteWithRetry_SuccessAfterRetries(t *testing.T) {
	cfg := DefaultRetryConfig()
	cfg.InitialDelay = 1 * time.Millisecond
	cfg.MaxAttempts = 5

	calls := 0
	result, err := ExecuteWithRetry(context.Background(), cfg, func(_ context.Context) (string, error) {
		calls++
		if calls < 3 {
			return "", &APIError{StatusCode: 503, Message: "unavailable"}
		}
		return "recovered", nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "recovered" {
		t.Errorf("result = %q, want %q", result, "recovered")
	}
	if calls != 3 {
		t.Errorf("calls = %d, want 3", calls)
	}
}

func TestExecuteWithRetry_NonRetryableError(t *testing.T) {
	cfg := DefaultRetryConfig()
	cfg.InitialDelay = 1 * time.Millisecond
	cfg.MaxAttempts = 5

	calls := 0
	_, err := ExecuteWithRetry(context.Background(), cfg, func(_ context.Context) (string, error) {
		calls++
		return "", &APIError{StatusCode: 401, Message: "unauthorized"}
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if calls != 1 {
		t.Errorf("calls = %d, want 1 (should not retry non-retryable)", calls)
	}
}

func TestExecuteWithRetry_MaxAttemptsExhausted(t *testing.T) {
	cfg := DefaultRetryConfig()
	cfg.InitialDelay = 1 * time.Millisecond
	cfg.MaxAttempts = 3

	calls := 0
	_, err := ExecuteWithRetry(context.Background(), cfg, func(_ context.Context) (string, error) {
		calls++
		return "", &APIError{StatusCode: 503, Message: "unavailable"}
	})

	if err == nil {
		t.Fatal("expected error after max attempts, got nil")
	}
	if calls != 3 {
		t.Errorf("calls = %d, want 3", calls)
	}
}

func TestExecuteWithRetry_ContextCancellation(t *testing.T) {
	cfg := DefaultRetryConfig()
	cfg.InitialDelay = 1 * time.Second // Long delay to test cancellation
	cfg.MaxAttempts = 10

	ctx, cancel := context.WithCancel(context.Background())

	calls := 0
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	_, err := ExecuteWithRetry(ctx, cfg, func(_ context.Context) (string, error) {
		calls++
		return "", &APIError{StatusCode: 503, Message: "unavailable"}
	})

	if err == nil {
		t.Fatal("expected cancellation error, got nil")
	}
	if calls > 3 {
		t.Errorf("calls = %d, expected few attempts before cancellation", calls)
	}
}

func TestExecuteWithRetry_Persistent429Callback(t *testing.T) {
	cfg := DefaultRetryConfig()
	cfg.InitialDelay = 1 * time.Millisecond
	cfg.MaxAttempts = 10
	cfg.Persistent429Threshold = 3

	callbackCalled := false
	callbackAttempt := 0
	cfg.OnPersistent429 = func(attempt int) bool {
		callbackCalled = true
		callbackAttempt = attempt
		return false // Stop retrying
	}

	calls := 0
	_, err := ExecuteWithRetry(context.Background(), cfg, func(_ context.Context) (string, error) {
		calls++
		return "", &APIError{StatusCode: 429, Message: "rate limited"}
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !callbackCalled {
		t.Error("OnPersistent429 callback was not called")
	}
	if callbackAttempt != 3 {
		t.Errorf("callback called at attempt %d, want 3", callbackAttempt)
	}
}

func TestExecuteWithRetry_Persistent429ContinueRetrying(t *testing.T) {
	cfg := DefaultRetryConfig()
	cfg.InitialDelay = 1 * time.Millisecond
	cfg.MaxAttempts = 10
	cfg.Persistent429Threshold = 3

	cfg.OnPersistent429 = func(_ int) bool {
		return true // Continue retrying
	}

	calls := 0
	_, err := ExecuteWithRetry(context.Background(), cfg, func(_ context.Context) (string, error) {
		calls++
		if calls >= 7 {
			return "finally", nil
		}
		return "", &APIError{StatusCode: 429, Message: "rate limited"}
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 7 {
		t.Errorf("calls = %d, want 7", calls)
	}
}

func TestGetFallbackModels(t *testing.T) {
	tests := []struct {
		modelID string
		want    int
	}{
		{"gemini-3.1-pro-preview", 1},
		{"gemini-3.1-flash", 1},
		{"gemini-3.1-flash-image-preview", 0},
		{"gemini-2.5-pro", 0},
		{"gemini-2.5-flash", 0},
		{"unknown", 0},
	}

	for _, tt := range tests {
		t.Run(tt.modelID, func(t *testing.T) {
			fallbacks := GetFallbackModels(tt.modelID)
			if len(fallbacks) != tt.want {
				t.Errorf("GetFallbackModels(%q) = %d models, want %d", tt.modelID, len(fallbacks), tt.want)
			}
		})
	}
}

// Ensure APIError implements the error interface properly.
func TestAPIError_Error(t *testing.T) {
	err := &APIError{StatusCode: 429, Message: "rate limited"}
	got := err.Error()
	want := "gemini API error 429: rate limited"
	if got != want {
		t.Errorf("APIError.Error() = %q, want %q", got, want)
	}
}

// Compile-time interface check.
var _ net.Error = (*net.OpError)(nil)
