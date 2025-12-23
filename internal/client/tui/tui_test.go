package tui

import (
	"strings"
	"testing"
	"time"

	"gopublic/internal/client/events"
	"gopublic/internal/client/stats"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewModel(t *testing.T) {
	model := NewModel(nil, nil)

	if model.status != "connecting" {
		t.Errorf("expected status 'connecting', got '%s'", model.status)
	}
	if len(model.tunnels) != 0 {
		t.Errorf("expected 0 tunnels, got %d", len(model.tunnels))
	}
	if len(model.requests) != 0 {
		t.Errorf("expected 0 requests, got %d", len(model.requests))
	}
}

func TestNewModel_WithDependencies(t *testing.T) {
	bus := events.NewBus()
	statsTracker := stats.New()

	model := NewModel(bus, statsTracker)

	if model.eventBus != bus {
		t.Error("eventBus should be set")
	}
	if model.stats != statsTracker {
		t.Error("stats should be set")
	}
	if model.eventSub == nil {
		t.Error("eventSub should be subscribed")
	}
}

func TestModel_Init(t *testing.T) {
	model := NewModel(nil, nil)
	cmd := model.Init()

	if cmd == nil {
		t.Error("Init should return a command")
	}
}

func TestModel_Update_Quit(t *testing.T) {
	model := NewModel(nil, nil)

	// Test 'q' key
	newModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Error("expected quit command")
	}
	_ = newModel

	// Test 'ctrl+c'
	model2 := NewModel(nil, nil)
	_, cmd2 := model2.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd2 == nil {
		t.Error("expected quit command for ctrl+c")
	}
}

func TestModel_Update_WindowSize(t *testing.T) {
	model := NewModel(nil, nil)

	newModel, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m := newModel.(Model)

	if m.width != 120 {
		t.Errorf("expected width 120, got %d", m.width)
	}
	if m.height != 40 {
		t.Errorf("expected height 40, got %d", m.height)
	}
}

func TestModel_HandleEvent_Connecting(t *testing.T) {
	model := NewModel(nil, nil)
	model.status = "offline"

	model = model.handleEvent(events.Event{Type: events.EventConnecting})

	if model.status != "connecting" {
		t.Errorf("expected status 'connecting', got '%s'", model.status)
	}
}

func TestModel_HandleEvent_Connected(t *testing.T) {
	model := NewModel(nil, nil)

	model = model.handleEvent(events.Event{
		Type: events.EventConnected,
		Data: events.ConnectedData{
			ServerAddr:   "localhost:4443",
			BoundDomains: []string{"test.example.com"},
			Latency:      50 * time.Millisecond,
		},
	})

	if model.status != "online" {
		t.Errorf("expected status 'online', got '%s'", model.status)
	}
	if model.serverAddr != "localhost:4443" {
		t.Errorf("expected serverAddr 'localhost:4443', got '%s'", model.serverAddr)
	}
	if model.serverLatency != 50*time.Millisecond {
		t.Errorf("expected latency 50ms, got %v", model.serverLatency)
	}
}

func TestModel_HandleEvent_Disconnected(t *testing.T) {
	model := NewModel(nil, nil)
	model.status = "online"

	model = model.handleEvent(events.Event{Type: events.EventDisconnected})

	if model.status != "offline" {
		t.Errorf("expected status 'offline', got '%s'", model.status)
	}
}

func TestModel_HandleEvent_Reconnecting(t *testing.T) {
	model := NewModel(nil, nil)
	model.status = "offline"

	model = model.handleEvent(events.Event{Type: events.EventReconnecting})

	if model.status != "reconnecting" {
		t.Errorf("expected status 'reconnecting', got '%s'", model.status)
	}
}

func TestModel_HandleEvent_TunnelReady(t *testing.T) {
	model := NewModel(nil, nil)

	model = model.handleEvent(events.Event{
		Type: events.EventTunnelReady,
		Data: events.TunnelReadyData{
			LocalPort:    "3000",
			BoundDomains: []string{"test.example.com"},
			Scheme:       "https",
		},
	})

	if len(model.tunnels) != 1 {
		t.Fatalf("expected 1 tunnel, got %d", len(model.tunnels))
	}
	if model.tunnels[0].LocalPort != "3000" {
		t.Errorf("expected LocalPort '3000', got '%s'", model.tunnels[0].LocalPort)
	}
	if model.tunnels[0].Scheme != "https" {
		t.Errorf("expected Scheme 'https', got '%s'", model.tunnels[0].Scheme)
	}
}

func TestModel_HandleEvent_TunnelReady_AddDomain(t *testing.T) {
	model := NewModel(nil, nil)

	// Add first domain
	model = model.handleEvent(events.Event{
		Type: events.EventTunnelReady,
		Data: events.TunnelReadyData{
			LocalPort:    "3000",
			BoundDomains: []string{"test1.example.com"},
			Scheme:       "https",
		},
	})

	// Add second domain to same tunnel
	model = model.handleEvent(events.Event{
		Type: events.EventTunnelReady,
		Data: events.TunnelReadyData{
			LocalPort:    "3000",
			BoundDomains: []string{"test2.example.com"},
			Scheme:       "https",
		},
	})

	if len(model.tunnels) != 1 {
		t.Fatalf("expected 1 tunnel, got %d", len(model.tunnels))
	}
	if len(model.tunnels[0].BoundDomains) != 2 {
		t.Errorf("expected 2 domains, got %d", len(model.tunnels[0].BoundDomains))
	}
}

func TestModel_HandleEvent_RequestComplete(t *testing.T) {
	model := NewModel(nil, nil)
	model.maxRequests = 5

	model = model.handleEvent(events.Event{
		Type: events.EventRequestComplete,
		Data: events.RequestData{
			Method:   "GET",
			Path:     "/api/test",
			Status:   200,
			Duration: 50 * time.Millisecond,
			Bytes:    1024,
		},
	})

	if len(model.requests) != 1 {
		t.Fatalf("expected 1 request, got %d", len(model.requests))
	}
	if model.requests[0].Method != "GET" {
		t.Errorf("expected Method 'GET', got '%s'", model.requests[0].Method)
	}
	if model.requests[0].Status != 200 {
		t.Errorf("expected Status 200, got %d", model.requests[0].Status)
	}
}

func TestModel_HandleEvent_RequestComplete_MaxLimit(t *testing.T) {
	model := NewModel(nil, nil)
	model.maxRequests = 3

	// Add 5 requests
	for i := 0; i < 5; i++ {
		model = model.handleEvent(events.Event{
			Type: events.EventRequestComplete,
			Data: events.RequestData{
				Method: "GET",
				Path:   "/test",
				Status: 200,
			},
		})
	}

	// Should only keep maxRequests
	if len(model.requests) != 3 {
		t.Errorf("expected 3 requests (max), got %d", len(model.requests))
	}
}

func TestModel_HandleEvent_Error(t *testing.T) {
	model := NewModel(nil, nil)

	model = model.handleEvent(events.Event{
		Type: events.EventError,
		Data: events.ErrorData{
			Error:   nil,
			Context: "test_context",
		},
	})

	if !strings.Contains(model.lastError, "test_context") {
		t.Errorf("expected lastError to contain 'test_context', got '%s'", model.lastError)
	}
}

func TestModel_View_ContainsHeader(t *testing.T) {
	model := NewModel(nil, nil)
	model.width = 80

	view := model.View()

	if !strings.Contains(view, "gopublic") {
		t.Error("view should contain 'gopublic' header")
	}
	if !strings.Contains(view, "Ctrl+C") {
		t.Error("view should contain quit hint")
	}
}

func TestModel_View_ContainsStatus(t *testing.T) {
	model := NewModel(nil, nil)
	model.status = "online"

	view := model.View()

	if !strings.Contains(view, "Session Status") {
		t.Error("view should contain 'Session Status' label")
	}
	if !strings.Contains(view, "online") {
		t.Error("view should contain 'online' status")
	}
}

func TestModel_View_ContainsWebInterface(t *testing.T) {
	model := NewModel(nil, nil)

	view := model.View()

	if !strings.Contains(view, "Web Interface") {
		t.Error("view should contain 'Web Interface' label")
	}
	if !strings.Contains(view, "4040") {
		t.Error("view should contain inspector port")
	}
}

func TestModel_View_ContainsForwarding(t *testing.T) {
	model := NewModel(nil, nil)
	model.tunnels = []TunnelInfo{
		{
			LocalPort:    "3000",
			BoundDomains: []string{"test.example.com"},
			Scheme:       "https",
		},
	}

	view := model.View()

	if !strings.Contains(view, "Forwarding") {
		t.Error("view should contain 'Forwarding' label")
	}
	if !strings.Contains(view, "test.example.com") {
		t.Error("view should contain domain")
	}
	if !strings.Contains(view, "localhost:3000") {
		t.Error("view should contain local port")
	}
}

func TestModel_View_ContainsStats(t *testing.T) {
	model := NewModel(nil, nil)

	view := model.View()

	if !strings.Contains(view, "Connections") {
		t.Error("view should contain 'Connections' label")
	}
	// Check for stats headers
	for _, header := range []string{"ttl", "opn", "rt1", "rt5", "p50", "p90"} {
		if !strings.Contains(view, header) {
			t.Errorf("view should contain stats header '%s'", header)
		}
	}
}

func TestModel_View_ContainsRequests(t *testing.T) {
	model := NewModel(nil, nil)
	model.requests = []RequestEntry{
		{
			Method:   "GET",
			Path:     "/api/users",
			Status:   200,
			Duration: 45 * time.Millisecond,
		},
	}

	view := model.View()

	if !strings.Contains(view, "HTTP Requests") {
		t.Error("view should contain 'HTTP Requests' label")
	}
	if !strings.Contains(view, "GET") {
		t.Error("view should contain request method")
	}
	if !strings.Contains(view, "/api/users") {
		t.Error("view should contain request path")
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		expected string
	}{
		{0, "0.00"},
		{50 * time.Millisecond, "0.05"},
		{500 * time.Millisecond, "0.50"},
		{1500 * time.Millisecond, "1.5"},
	}

	for _, tt := range tests {
		result := formatDuration(tt.duration)
		if result != tt.expected {
			t.Errorf("formatDuration(%v) = %s, expected %s", tt.duration, result, tt.expected)
		}
	}
}

func TestTruncatePath(t *testing.T) {
	tests := []struct {
		path     string
		maxLen   int
		expected string
	}{
		{"/short", 20, "/short"},
		{"/very/long/path/that/exceeds/limit", 20, "/very/long/path/t..."},
		{"", 10, ""},
	}

	for _, tt := range tests {
		result := truncatePath(tt.path, tt.maxLen)
		if result != tt.expected {
			t.Errorf("truncatePath(%s, %d) = %s, expected %s", tt.path, tt.maxLen, result, tt.expected)
		}
	}
}

func TestStatusText(t *testing.T) {
	// Just ensure it doesn't panic and returns something
	statuses := []string{"online", "connecting", "reconnecting", "offline", "unknown"}
	for _, s := range statuses {
		result := StatusText(s)
		if result == "" {
			t.Errorf("StatusText(%s) returned empty string", s)
		}
	}
}

func TestMethodText(t *testing.T) {
	methods := []string{"GET", "POST", "PUT", "DELETE"}
	for _, m := range methods {
		result := MethodText(m)
		if result == "" {
			t.Errorf("MethodText(%s) returned empty string", m)
		}
	}
}

func TestStatusCodeText(t *testing.T) {
	codes := []int{200, 201, 301, 400, 404, 500}
	for _, c := range codes {
		result := StatusCodeText(c)
		if result == "" {
			t.Errorf("StatusCodeText(%d) returned empty string", c)
		}
	}
}
