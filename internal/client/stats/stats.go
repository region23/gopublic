package stats

import (
	"sort"
	"sync"
	"time"
)

// Stats tracks connection and request statistics with thread-safe access.
type Stats struct {
	mu sync.RWMutex

	totalConns    int64
	openConns     int64
	totalRequests int64
	totalBytes    int64

	// Ring buffer for request times (for percentile calculations)
	requestTimes []time.Duration
	maxSamples   int

	// Server latency (measured during handshake)
	serverLatency time.Duration

	startTime time.Time
}

// Snapshot represents a point-in-time view of statistics.
type Snapshot struct {
	TotalConnections int64
	OpenConnections  int64
	TotalRequests    int64
	TotalBytes       int64

	// Request timing metrics
	RT1 time.Duration // Last request time
	RT5 time.Duration // Average of last 5 requests
	P50 time.Duration // 50th percentile
	P90 time.Duration // 90th percentile

	ServerLatency time.Duration
	Uptime        time.Duration
}

// New creates a new Stats tracker.
func New() *Stats {
	return &Stats{
		requestTimes: make([]time.Duration, 0, 100),
		maxSamples:   100,
		startTime:    time.Now(),
	}
}

// NewWithOptions creates a Stats tracker with custom options.
func NewWithOptions(maxSamples int) *Stats {
	if maxSamples <= 0 {
		maxSamples = 100
	}
	return &Stats{
		requestTimes: make([]time.Duration, 0, maxSamples),
		maxSamples:   maxSamples,
		startTime:    time.Now(),
	}
}

// IncrementConnections increments the connection counters.
func (s *Stats) IncrementConnections() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.totalConns++
	s.openConns++
}

// DecrementOpenConnections decrements the open connection counter.
func (s *Stats) DecrementOpenConnections() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.openConns > 0 {
		s.openConns--
	}
}

// RecordRequest records a completed request with its duration and size.
func (s *Stats) RecordRequest(duration time.Duration, bytes int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.totalRequests++
	s.totalBytes += bytes

	// Add to ring buffer
	if len(s.requestTimes) >= s.maxSamples {
		// Shift left, drop oldest
		copy(s.requestTimes, s.requestTimes[1:])
		s.requestTimes = s.requestTimes[:len(s.requestTimes)-1]
	}
	s.requestTimes = append(s.requestTimes, duration)
}

// SetServerLatency sets the measured server latency.
func (s *Stats) SetServerLatency(latency time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.serverLatency = latency
}

// Snapshot returns a point-in-time view of all statistics.
func (s *Stats) Snapshot() Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	snap := Snapshot{
		TotalConnections: s.totalConns,
		OpenConnections:  s.openConns,
		TotalRequests:    s.totalRequests,
		TotalBytes:       s.totalBytes,
		ServerLatency:    s.serverLatency,
		Uptime:           time.Since(s.startTime),
	}

	n := len(s.requestTimes)
	if n == 0 {
		return snap
	}

	// RT1: Last request time
	snap.RT1 = s.requestTimes[n-1]

	// RT5: Average of last 5 requests
	count := 5
	if n < count {
		count = n
	}
	var sum time.Duration
	for i := n - count; i < n; i++ {
		sum += s.requestTimes[i]
	}
	snap.RT5 = sum / time.Duration(count)

	// Percentiles require sorted copy
	sorted := make([]time.Duration, n)
	copy(sorted, s.requestTimes)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})

	// P50: 50th percentile (median)
	snap.P50 = sorted[n/2]

	// P90: 90th percentile
	p90Index := int(float64(n) * 0.9)
	if p90Index >= n {
		p90Index = n - 1
	}
	snap.P90 = sorted[p90Index]

	return snap
}

// Reset clears all statistics.
func (s *Stats) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.totalConns = 0
	s.openConns = 0
	s.totalRequests = 0
	s.totalBytes = 0
	s.requestTimes = s.requestTimes[:0]
	s.serverLatency = 0
	s.startTime = time.Now()
}
