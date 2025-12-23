package tunnel

import (
	"context"
	"testing"
	"time"
)

func TestDefaultReconnectConfig(t *testing.T) {
	cfg := DefaultReconnectConfig()

	if cfg.InitialDelay != 1*time.Second {
		t.Errorf("InitialDelay = %v, want 1s", cfg.InitialDelay)
	}

	if cfg.MaxDelay != 60*time.Second {
		t.Errorf("MaxDelay = %v, want 60s", cfg.MaxDelay)
	}

	if cfg.Multiplier != 2.0 {
		t.Errorf("Multiplier = %v, want 2.0", cfg.Multiplier)
	}

	if cfg.MaxAttempts != 0 {
		t.Errorf("MaxAttempts = %d, want 0 (infinite)", cfg.MaxAttempts)
	}
}

func TestExponentialBackoff(t *testing.T) {
	cfg := &ReconnectConfig{
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     1 * time.Second,
		Multiplier:   2.0,
		MaxAttempts:  0,
	}

	delay := cfg.InitialDelay

	// First delay
	if delay != 100*time.Millisecond {
		t.Errorf("First delay = %v, want 100ms", delay)
	}

	// Second delay (doubled)
	delay = time.Duration(float64(delay) * cfg.Multiplier)
	if delay != 200*time.Millisecond {
		t.Errorf("Second delay = %v, want 200ms", delay)
	}

	// Third delay (doubled again)
	delay = time.Duration(float64(delay) * cfg.Multiplier)
	if delay != 400*time.Millisecond {
		t.Errorf("Third delay = %v, want 400ms", delay)
	}

	// Check max delay cap
	for i := 0; i < 10; i++ {
		delay = time.Duration(float64(delay) * cfg.Multiplier)
		if delay > cfg.MaxDelay {
			delay = cfg.MaxDelay
		}
	}

	if delay != cfg.MaxDelay {
		t.Errorf("Delay should be capped at MaxDelay, got %v", delay)
	}
}

func TestStartWithReconnect_ContextCancellation(t *testing.T) {
	tunnel := NewTunnel("invalid-server:9999", "test-token", "3000")

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	cfg := &ReconnectConfig{
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     50 * time.Millisecond,
		Multiplier:   2.0,
		MaxAttempts:  0,
	}

	err := tunnel.StartWithReconnect(ctx, cfg)

	// Should return context error (deadline exceeded or canceled)
	if err == nil {
		t.Error("StartWithReconnect() should return error when context is cancelled")
	}

	if err != context.DeadlineExceeded && err != context.Canceled {
		t.Errorf("Expected context error, got: %v", err)
	}
}

func TestStartWithReconnect_MaxAttempts(t *testing.T) {
	tunnel := NewTunnel("invalid-server:9999", "test-token", "3000")

	ctx := context.Background()

	cfg := &ReconnectConfig{
		InitialDelay: 1 * time.Millisecond,
		MaxDelay:     5 * time.Millisecond,
		Multiplier:   1.0, // No backoff increase
		MaxAttempts:  3,
	}

	start := time.Now()
	err := tunnel.StartWithReconnect(ctx, cfg)
	elapsed := time.Since(start)

	if err == nil {
		t.Error("StartWithReconnect() should return error after max attempts")
	}

	// Should finish quickly (3 attempts * ~1ms delay = ~3ms, plus connection attempts)
	if elapsed > 5*time.Second {
		t.Errorf("Took too long: %v", elapsed)
	}
}
