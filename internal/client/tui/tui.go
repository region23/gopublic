package tui

import (
	"fmt"
	"strings"
	"time"

	"gopublic/internal/client/events"
	"gopublic/internal/client/stats"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Version can be set at build time
var Version = "dev"

// TunnelInfo represents info about a single tunnel
type TunnelInfo struct {
	Name         string
	LocalPort    string
	BoundDomains []string
	Scheme       string
}

// RequestEntry represents a recent request for display
type RequestEntry struct {
	Method   string
	Path     string
	Status   int
	Duration time.Duration
	Time     time.Time
}

// Model is the main Bubble Tea model
type Model struct {
	// Connection state
	status string // "connecting", "online", "reconnecting", "offline"

	// Tunnel information
	tunnels []TunnelInfo

	// Dependencies
	stats    *stats.Stats
	eventBus *events.Bus
	eventSub <-chan events.Event

	// Display state
	width     int
	height    int
	startTime time.Time

	// Server info
	serverAddr    string
	serverLatency time.Duration

	// Recent requests for display
	requests   []RequestEntry
	maxRequests int

	// Error message (if any)
	lastError string
}

// NewModel creates a new TUI model
func NewModel(eventBus *events.Bus, statsTracker *stats.Stats) Model {
	var eventSub <-chan events.Event
	if eventBus != nil {
		eventSub = eventBus.Subscribe()
	}

	return Model{
		status:      "connecting",
		tunnels:     make([]TunnelInfo, 0),
		stats:       statsTracker,
		eventBus:    eventBus,
		eventSub:    eventSub,
		startTime:   time.Now(),
		requests:    make([]RequestEntry, 0),
		maxRequests: 10,
	}
}

// Messages
type tickMsg time.Time
type eventMsg events.Event

// Commands
func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func waitForEvent(sub <-chan events.Event) tea.Cmd {
	return func() tea.Msg {
		if sub == nil {
			return nil
		}
		event, ok := <-sub
		if !ok {
			return nil
		}
		return eventMsg(event)
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{tickCmd()}
	if m.eventSub != nil {
		cmds = append(cmds, waitForEvent(m.eventSub))
	}
	return tea.Batch(cmds...)
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tickMsg:
		// Refresh stats display
		return m, tickCmd()

	case eventMsg:
		m = m.handleEvent(events.Event(msg))
		return m, waitForEvent(m.eventSub)
	}

	return m, nil
}

func (m Model) handleEvent(event events.Event) Model {
	switch event.Type {
	case events.EventConnecting:
		m.status = "connecting"

	case events.EventConnected:
		m.status = "online"
		if data, ok := event.Data.(events.ConnectedData); ok {
			m.serverAddr = data.ServerAddr
			m.serverLatency = data.Latency
		}

	case events.EventDisconnected:
		m.status = "offline"

	case events.EventReconnecting:
		m.status = "reconnecting"

	case events.EventTunnelReady:
		if data, ok := event.Data.(events.TunnelReadyData); ok {
			// Add or update tunnel info
			found := false
			for i, t := range m.tunnels {
				if t.LocalPort == data.LocalPort {
					m.tunnels[i].BoundDomains = append(m.tunnels[i].BoundDomains, data.BoundDomains...)
					found = true
					break
				}
			}
			if !found {
				m.tunnels = append(m.tunnels, TunnelInfo{
					Name:         data.Name,
					LocalPort:    data.LocalPort,
					BoundDomains: data.BoundDomains,
					Scheme:       data.Scheme,
				})
			}
		}

	case events.EventRequestComplete:
		if data, ok := event.Data.(events.RequestData); ok {
			entry := RequestEntry{
				Method:   data.Method,
				Path:     data.Path,
				Status:   data.Status,
				Duration: data.Duration,
				Time:     time.Now(),
			}
			// Prepend (newest first)
			m.requests = append([]RequestEntry{entry}, m.requests...)
			if len(m.requests) > m.maxRequests {
				m.requests = m.requests[:m.maxRequests]
			}
		}

	case events.EventError:
		if data, ok := event.Data.(events.ErrorData); ok {
			m.lastError = fmt.Sprintf("%s: %v", data.Context, data.Error)
		}
	}

	return m
}

// View renders the model
func (m Model) View() string {
	var b strings.Builder

	// Header
	b.WriteString(m.renderHeader())
	b.WriteString("\n\n")

	// Status section
	b.WriteString(m.renderStatus())
	b.WriteString("\n")

	// Forwarding section
	if len(m.tunnels) > 0 {
		b.WriteString(m.renderForwarding())
		b.WriteString("\n")
	}

	// Stats section
	b.WriteString(m.renderStats())
	b.WriteString("\n")

	// Recent requests
	if len(m.requests) > 0 {
		b.WriteString(m.renderRequests())
	}

	return b.String()
}

func (m Model) renderHeader() string {
	title := titleStyle.Render("gopublic")
	hint := hintStyle.Render("(Ctrl+C to quit)")

	// Calculate spacing
	spacing := ""
	if m.width > 0 {
		titleLen := lipgloss.Width(title)
		hintLen := lipgloss.Width(hint)
		spaces := m.width - titleLen - hintLen
		if spaces > 0 {
			spacing = strings.Repeat(" ", spaces)
		}
	} else {
		spacing = strings.Repeat(" ", 40)
	}

	return title + spacing + hint
}

func (m Model) renderStatus() string {
	var lines []string

	// Session Status
	lines = append(lines, m.renderField("Session Status", StatusText(m.status)))

	// Version
	lines = append(lines, m.renderField("Version", Version))

	// Latency
	latencyStr := "-"
	if m.serverLatency > 0 {
		latencyStr = fmt.Sprintf("%dms", m.serverLatency.Milliseconds())
	}
	lines = append(lines, m.renderField("Latency", latencyStr))

	// Web Interface
	lines = append(lines, m.renderField("Web Interface", urlStyle.Render("http://127.0.0.1:4040")))

	return strings.Join(lines, "\n")
}

func (m Model) renderField(label, value string) string {
	return labelStyle.Render(label) + valueStyle.Render(value)
}

func (m Model) renderForwarding() string {
	var lines []string
	lines = append(lines, "") // Empty line before

	for i, t := range m.tunnels {
		for j, domain := range t.BoundDomains {
			label := ""
			if i == 0 && j == 0 {
				label = "Forwarding"
			}

			url := fmt.Sprintf("%s://%s", t.Scheme, domain)
			local := fmt.Sprintf("http://localhost:%s", t.LocalPort)

			value := urlStyle.Render(url) + arrowStyle.Render(" -> ") + valueStyle.Render(local)
			lines = append(lines, labelStyle.Render(label)+value)
		}
	}

	return strings.Join(lines, "\n")
}

func (m Model) renderStats() string {
	var lines []string
	lines = append(lines, "") // Empty line before

	// Get stats snapshot
	var snap stats.Snapshot
	if m.stats != nil {
		snap = m.stats.Snapshot()
	}

	// Header row
	headers := []string{"ttl", "opn", "rt1", "rt5", "p50", "p90"}
	headerRow := labelStyle.Render("Connections")
	for _, h := range headers {
		headerRow += statsHeaderStyle.Render(h)
	}
	lines = append(lines, headerRow)

	// Values row
	valueRow := labelStyle.Render("")
	valueRow += statsValueStyle.Render(fmt.Sprintf("%d", snap.TotalConnections))
	valueRow += statsValueStyle.Render(fmt.Sprintf("%d", snap.OpenConnections))
	valueRow += statsValueStyle.Render(formatDuration(snap.RT1))
	valueRow += statsValueStyle.Render(formatDuration(snap.RT5))
	valueRow += statsValueStyle.Render(formatDuration(snap.P50))
	valueRow += statsValueStyle.Render(formatDuration(snap.P90))
	lines = append(lines, valueRow)

	return strings.Join(lines, "\n")
}

func (m Model) renderRequests() string {
	var lines []string
	lines = append(lines, "") // Empty line before
	lines = append(lines, labelStyle.Render("HTTP Requests"))

	for _, req := range m.requests {
		method := MethodText(req.Method)
		path := pathStyle.Render(truncatePath(req.Path, 40))
		status := StatusCodeText(req.Status)
		duration := durationStyle.Render(formatDuration(req.Duration))

		line := fmt.Sprintf("%s %s %s %s", method, path, status, duration)
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

// Helper functions

func formatDuration(d time.Duration) string {
	if d == 0 {
		return "0.00"
	}
	secs := d.Seconds()
	if secs < 1 {
		return fmt.Sprintf("%.2f", secs)
	}
	return fmt.Sprintf("%.1f", secs)
}

func truncatePath(path string, maxLen int) string {
	if len(path) <= maxLen {
		return path
	}
	return path[:maxLen-3] + "..."
}

// Run starts the TUI application
func Run(eventBus *events.Bus, statsTracker *stats.Stats) error {
	model := NewModel(eventBus, statsTracker)
	p := tea.NewProgram(model, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
