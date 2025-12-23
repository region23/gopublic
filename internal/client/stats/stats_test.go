package stats

import (
	"sync"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	s := New()
	if s == nil {
		t.Fatal("New() returned nil")
	}

	snap := s.Snapshot()
	if snap.TotalConnections != 0 {
		t.Errorf("expected 0 total connections, got %d", snap.TotalConnections)
	}
	if snap.OpenConnections != 0 {
		t.Errorf("expected 0 open connections, got %d", snap.OpenConnections)
	}
}

func TestIncrementConnections(t *testing.T) {
	s := New()

	s.IncrementConnections()
	s.IncrementConnections()
	s.IncrementConnections()

	snap := s.Snapshot()
	if snap.TotalConnections != 3 {
		t.Errorf("expected 3 total connections, got %d", snap.TotalConnections)
	}
	if snap.OpenConnections != 3 {
		t.Errorf("expected 3 open connections, got %d", snap.OpenConnections)
	}
}

func TestDecrementOpenConnections(t *testing.T) {
	s := New()

	s.IncrementConnections()
	s.IncrementConnections()
	s.DecrementOpenConnections()

	snap := s.Snapshot()
	if snap.TotalConnections != 2 {
		t.Errorf("expected 2 total connections, got %d", snap.TotalConnections)
	}
	if snap.OpenConnections != 1 {
		t.Errorf("expected 1 open connection, got %d", snap.OpenConnections)
	}

	// Should not go below 0
	s.DecrementOpenConnections()
	s.DecrementOpenConnections()
	snap = s.Snapshot()
	if snap.OpenConnections != 0 {
		t.Errorf("expected 0 open connections (not negative), got %d", snap.OpenConnections)
	}
}

func TestRecordRequest(t *testing.T) {
	s := New()

	s.RecordRequest(100*time.Millisecond, 1024)
	s.RecordRequest(200*time.Millisecond, 2048)
	s.RecordRequest(150*time.Millisecond, 512)

	snap := s.Snapshot()
	if snap.TotalRequests != 3 {
		t.Errorf("expected 3 total requests, got %d", snap.TotalRequests)
	}
	if snap.TotalBytes != 3584 {
		t.Errorf("expected 3584 total bytes, got %d", snap.TotalBytes)
	}
	if snap.RT1 != 150*time.Millisecond {
		t.Errorf("expected RT1 150ms, got %v", snap.RT1)
	}
}

func TestRT5Average(t *testing.T) {
	s := New()

	// Record exactly 5 requests
	durations := []time.Duration{
		100 * time.Millisecond,
		200 * time.Millisecond,
		300 * time.Millisecond,
		400 * time.Millisecond,
		500 * time.Millisecond,
	}
	for _, d := range durations {
		s.RecordRequest(d, 0)
	}

	snap := s.Snapshot()
	expected := 300 * time.Millisecond // (100+200+300+400+500)/5
	if snap.RT5 != expected {
		t.Errorf("expected RT5 %v, got %v", expected, snap.RT5)
	}
}

func TestPercentiles(t *testing.T) {
	s := NewWithOptions(100)

	// Record 10 requests with known durations: 10, 20, 30, 40, 50, 60, 70, 80, 90, 100
	for i := 1; i <= 10; i++ {
		s.RecordRequest(time.Duration(i*10)*time.Millisecond, 0)
	}

	snap := s.Snapshot()

	// For n=10 elements [10,20,30,40,50,60,70,80,90,100]:
	// P50 at index n/2 = 5 → 60ms
	if snap.P50 != 60*time.Millisecond {
		t.Errorf("expected P50 60ms, got %v", snap.P50)
	}

	// P90 at index int(0.9*10) = 9 → 100ms
	if snap.P90 != 100*time.Millisecond {
		t.Errorf("expected P90 100ms, got %v", snap.P90)
	}
}

func TestRingBufferOverflow(t *testing.T) {
	s := NewWithOptions(5) // Small buffer

	// Record more than buffer size
	for i := 0; i < 10; i++ {
		s.RecordRequest(time.Duration(i)*time.Millisecond, 0)
	}

	snap := s.Snapshot()

	// RT1 should be the last recorded value (9ms)
	if snap.RT1 != 9*time.Millisecond {
		t.Errorf("expected RT1 9ms, got %v", snap.RT1)
	}

	// Should only have 5 samples (buffer size)
	// We can't directly check buffer size, but RT5 should average last 5: 5,6,7,8,9
	expected := (5 + 6 + 7 + 8 + 9) * time.Millisecond / 5
	if snap.RT5 != expected {
		t.Errorf("expected RT5 %v, got %v", expected, snap.RT5)
	}
}

func TestSetServerLatency(t *testing.T) {
	s := New()

	s.SetServerLatency(45 * time.Millisecond)

	snap := s.Snapshot()
	if snap.ServerLatency != 45*time.Millisecond {
		t.Errorf("expected latency 45ms, got %v", snap.ServerLatency)
	}
}

func TestUptime(t *testing.T) {
	s := New()

	time.Sleep(10 * time.Millisecond)

	snap := s.Snapshot()
	if snap.Uptime < 10*time.Millisecond {
		t.Errorf("expected uptime >= 10ms, got %v", snap.Uptime)
	}
}

func TestReset(t *testing.T) {
	s := New()

	s.IncrementConnections()
	s.RecordRequest(100*time.Millisecond, 1024)
	s.SetServerLatency(50 * time.Millisecond)

	s.Reset()

	snap := s.Snapshot()
	if snap.TotalConnections != 0 {
		t.Errorf("expected 0 connections after reset, got %d", snap.TotalConnections)
	}
	if snap.TotalRequests != 0 {
		t.Errorf("expected 0 requests after reset, got %d", snap.TotalRequests)
	}
	if snap.ServerLatency != 0 {
		t.Errorf("expected 0 latency after reset, got %v", snap.ServerLatency)
	}
}

func TestConcurrentAccess(t *testing.T) {
	s := New()
	var wg sync.WaitGroup

	// Simulate concurrent access
	for i := 0; i < 100; i++ {
		wg.Add(3)

		go func() {
			defer wg.Done()
			s.IncrementConnections()
		}()

		go func() {
			defer wg.Done()
			s.RecordRequest(10*time.Millisecond, 100)
		}()

		go func() {
			defer wg.Done()
			_ = s.Snapshot()
		}()
	}

	wg.Wait()

	snap := s.Snapshot()
	if snap.TotalConnections != 100 {
		t.Errorf("expected 100 connections, got %d", snap.TotalConnections)
	}
	if snap.TotalRequests != 100 {
		t.Errorf("expected 100 requests, got %d", snap.TotalRequests)
	}
}

func TestEmptySnapshot(t *testing.T) {
	s := New()
	snap := s.Snapshot()

	// All timing metrics should be zero for empty stats
	if snap.RT1 != 0 {
		t.Errorf("expected RT1 0, got %v", snap.RT1)
	}
	if snap.RT5 != 0 {
		t.Errorf("expected RT5 0, got %v", snap.RT5)
	}
	if snap.P50 != 0 {
		t.Errorf("expected P50 0, got %v", snap.P50)
	}
	if snap.P90 != 0 {
		t.Errorf("expected P90 0, got %v", snap.P90)
	}
}
