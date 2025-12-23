package events

import (
	"errors"
	"sync"
	"testing"
	"time"
)

func TestNewBus(t *testing.T) {
	bus := NewBus()
	if bus == nil {
		t.Fatal("NewBus() returned nil")
	}
	if bus.SubscriberCount() != 0 {
		t.Errorf("expected 0 subscribers, got %d", bus.SubscriberCount())
	}
}

func TestSubscribe(t *testing.T) {
	bus := NewBus()

	ch1 := bus.Subscribe()
	ch2 := bus.Subscribe()

	if bus.SubscriberCount() != 2 {
		t.Errorf("expected 2 subscribers, got %d", bus.SubscriberCount())
	}

	if ch1 == nil || ch2 == nil {
		t.Error("Subscribe() returned nil channel")
	}
}

func TestPublish(t *testing.T) {
	bus := NewBus()
	ch := bus.Subscribe()

	event := Event{
		Type: EventConnected,
		Data: "test data",
	}
	bus.Publish(event)

	select {
	case received := <-ch:
		if received.Type != EventConnected {
			t.Errorf("expected EventConnected, got %v", received.Type)
		}
		if received.Data != "test data" {
			t.Errorf("expected 'test data', got %v", received.Data)
		}
		if received.Timestamp.IsZero() {
			t.Error("timestamp should be set automatically")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("timeout waiting for event")
	}
}

func TestPublishFanOut(t *testing.T) {
	bus := NewBus()

	ch1 := bus.Subscribe()
	ch2 := bus.Subscribe()
	ch3 := bus.Subscribe()

	bus.Publish(Event{Type: EventConnecting})

	// All subscribers should receive the event
	for i, ch := range []<-chan Event{ch1, ch2, ch3} {
		select {
		case event := <-ch:
			if event.Type != EventConnecting {
				t.Errorf("subscriber %d: expected EventConnecting, got %v", i, event.Type)
			}
		case <-time.After(100 * time.Millisecond):
			t.Errorf("subscriber %d: timeout waiting for event", i)
		}
	}
}

func TestPublishType(t *testing.T) {
	bus := NewBus()
	ch := bus.Subscribe()

	bus.PublishType(EventDisconnected)

	select {
	case event := <-ch:
		if event.Type != EventDisconnected {
			t.Errorf("expected EventDisconnected, got %v", event.Type)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("timeout waiting for event")
	}
}

func TestPublishError(t *testing.T) {
	bus := NewBus()
	ch := bus.Subscribe()

	testErr := errors.New("test error")
	bus.PublishError(testErr, "test context")

	select {
	case event := <-ch:
		if event.Type != EventError {
			t.Errorf("expected EventError, got %v", event.Type)
		}
		data, ok := event.Data.(ErrorData)
		if !ok {
			t.Fatal("expected ErrorData type")
		}
		if data.Error != testErr {
			t.Errorf("expected test error, got %v", data.Error)
		}
		if data.Context != "test context" {
			t.Errorf("expected 'test context', got %s", data.Context)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("timeout waiting for event")
	}
}

func TestUnsubscribe(t *testing.T) {
	bus := NewBus()

	ch := bus.Subscribe()
	if bus.SubscriberCount() != 1 {
		t.Errorf("expected 1 subscriber, got %d", bus.SubscriberCount())
	}

	bus.Unsubscribe(ch)
	if bus.SubscriberCount() != 0 {
		t.Errorf("expected 0 subscribers after unsubscribe, got %d", bus.SubscriberCount())
	}

	// Channel should be closed
	select {
	case _, ok := <-ch:
		if ok {
			t.Error("channel should be closed after unsubscribe")
		}
	default:
		// Channel might be closed but not yet readable
	}
}

func TestClose(t *testing.T) {
	bus := NewBus()

	ch1 := bus.Subscribe()
	ch2 := bus.Subscribe()

	bus.Close()

	// All channels should be closed
	for i, ch := range []<-chan Event{ch1, ch2} {
		select {
		case _, ok := <-ch:
			if ok {
				t.Errorf("channel %d should be closed", i)
			}
		case <-time.After(100 * time.Millisecond):
			t.Errorf("channel %d: timeout waiting for close", i)
		}
	}

	// Publishing after close should not panic
	bus.Publish(Event{Type: EventConnecting})

	// Subscribing after close should return closed channel
	ch := bus.Subscribe()
	select {
	case _, ok := <-ch:
		if ok {
			t.Error("channel should be closed after bus is closed")
		}
	default:
		// Expected
	}
}

func TestNonBlockingPublish(t *testing.T) {
	bus := NewBusWithBuffer(1) // Tiny buffer

	ch := bus.Subscribe()

	// Fill the buffer
	bus.Publish(Event{Type: EventConnecting})

	// This should not block even if buffer is full
	done := make(chan bool)
	go func() {
		bus.Publish(Event{Type: EventConnected})
		bus.Publish(Event{Type: EventDisconnected})
		done <- true
	}()

	select {
	case <-done:
		// Success - publish did not block
	case <-time.After(100 * time.Millisecond):
		t.Error("Publish blocked when buffer was full")
	}

	// Drain the channel
	<-ch
}

func TestConcurrentPublish(t *testing.T) {
	bus := NewBus()
	ch := bus.Subscribe()

	var wg sync.WaitGroup
	eventCount := 100

	// Start consumer
	received := make([]Event, 0, eventCount)
	var mu sync.Mutex
	go func() {
		for event := range ch {
			mu.Lock()
			received = append(received, event)
			mu.Unlock()
		}
	}()

	// Concurrent publishers
	for i := 0; i < eventCount; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			bus.Publish(Event{Type: EventRequestComplete, Data: idx})
		}(i)
	}

	wg.Wait()
	time.Sleep(50 * time.Millisecond) // Allow consumer to process

	bus.Close()

	mu.Lock()
	defer mu.Unlock()
	if len(received) != eventCount {
		t.Errorf("expected %d events, got %d", eventCount, len(received))
	}
}

func TestEventTypeString(t *testing.T) {
	tests := []struct {
		eventType EventType
		expected  string
	}{
		{EventConnecting, "connecting"},
		{EventConnected, "connected"},
		{EventDisconnected, "disconnected"},
		{EventReconnecting, "reconnecting"},
		{EventRequestStart, "request_start"},
		{EventRequestComplete, "request_complete"},
		{EventError, "error"},
		{EventTunnelReady, "tunnel_ready"},
		{EventType(999), "unknown"},
	}

	for _, tt := range tests {
		if got := tt.eventType.String(); got != tt.expected {
			t.Errorf("EventType(%d).String() = %s, want %s", tt.eventType, got, tt.expected)
		}
	}
}

func TestConnectedData(t *testing.T) {
	bus := NewBus()
	ch := bus.Subscribe()

	data := ConnectedData{
		ServerAddr:   "localhost:4443",
		BoundDomains: []string{"test.example.com"},
		Latency:      45 * time.Millisecond,
	}
	bus.Publish(Event{Type: EventConnected, Data: data})

	event := <-ch
	received, ok := event.Data.(ConnectedData)
	if !ok {
		t.Fatal("expected ConnectedData")
	}
	if received.ServerAddr != "localhost:4443" {
		t.Errorf("expected localhost:4443, got %s", received.ServerAddr)
	}
	if len(received.BoundDomains) != 1 || received.BoundDomains[0] != "test.example.com" {
		t.Errorf("unexpected BoundDomains: %v", received.BoundDomains)
	}
	if received.Latency != 45*time.Millisecond {
		t.Errorf("expected 45ms latency, got %v", received.Latency)
	}
}
