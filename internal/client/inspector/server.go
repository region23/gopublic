package inspector

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

//go:embed index.html
var indexHTML []byte

// HTTPExchange represents a complete HTTP request/response pair
type HTTPExchange struct {
	ID        int64         `json:"id"`
	Request   *HTTPRequest  `json:"request"`
	Response  *HTTPResponse `json:"response,omitempty"`
	Duration  int64         `json:"duration_ms"`
	Timestamp time.Time     `json:"timestamp"`
}

// HTTPRequest captures request details
type HTTPRequest struct {
	Method  string              `json:"method"`
	URL     string              `json:"url"`
	Proto   string              `json:"proto"`
	Headers map[string][]string `json:"headers"`
	Body    string              `json:"body"`
	Size    int64               `json:"size"`
}

// HTTPResponse captures response details
type HTTPResponse struct {
	Status  int                 `json:"status"`
	Proto   string              `json:"proto"`
	Headers map[string][]string `json:"headers"`
	Body    string              `json:"body"`
	Size    int64               `json:"size"`
}

var (
	exchanges  []HTTPExchange
	mu         sync.RWMutex
	nextID     int64
	localPort  string
	maxBodySize int64 = 1024 * 1024 // 1MB max body capture
)

// SetLocalPort configures the local port for replay functionality
func SetLocalPort(port string) {
	localPort = port
}

// AddExchange records a complete HTTP exchange
func AddExchange(req *http.Request, reqBody []byte, resp *http.Response, respBody []byte, duration time.Duration) int64 {
	mu.Lock()
	defer mu.Unlock()

	id := nextID
	nextID++

	exchange := HTTPExchange{
		ID:        id,
		Timestamp: time.Now(),
		Duration:  duration.Milliseconds(),
		Request: &HTTPRequest{
			Method:  req.Method,
			URL:     req.URL.String(),
			Proto:   req.Proto,
			Headers: req.Header,
			Body:    truncateBody(reqBody),
			Size:    int64(len(reqBody)),
		},
	}

	if resp != nil {
		exchange.Response = &HTTPResponse{
			Status:  resp.StatusCode,
			Proto:   resp.Proto,
			Headers: resp.Header,
			Body:    truncateBody(respBody),
			Size:    int64(len(respBody)),
		}
	}

	// Prepend to list (newest first)
	exchanges = append([]HTTPExchange{exchange}, exchanges...)
	if len(exchanges) > 100 {
		exchanges = exchanges[:100]
	}

	return id
}

// truncateBody limits body size for storage
func truncateBody(body []byte) string {
	if int64(len(body)) > maxBodySize {
		return string(body[:maxBodySize]) + "\n... (truncated)"
	}
	return string(body)
}

// GetExchange retrieves a specific exchange by ID
func GetExchange(id int64) (*HTTPExchange, bool) {
	mu.RLock()
	defer mu.RUnlock()

	for _, ex := range exchanges {
		if ex.ID == id {
			return &ex, true
		}
	}
	return nil, false
}

// Start launches the inspector web server
func Start(port string) {
	mux := http.NewServeMux()

	// Serve UI
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		w.Write(indexHTML)
	})

	// List all exchanges
	mux.HandleFunc("/api/exchanges", func(w http.ResponseWriter, r *http.Request) {
		mu.RLock()
		defer mu.RUnlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(exchanges)
	})

	// Get single exchange
	mux.HandleFunc("/api/exchanges/", func(w http.ResponseWriter, r *http.Request) {
		idStr := strings.TrimPrefix(r.URL.Path, "/api/exchanges/")

		// Handle replay endpoint
		if strings.HasPrefix(idStr, "replay/") {
			handleReplay(w, r, strings.TrimPrefix(idStr, "replay/"))
			return
		}

		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			http.Error(w, "Invalid ID", http.StatusBadRequest)
			return
		}

		exchange, ok := GetExchange(id)
		if !ok {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(exchange)
	})

	// Replay endpoint
	mux.HandleFunc("/api/replay/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		handleReplay(w, r, strings.TrimPrefix(r.URL.Path, "/api/replay/"))
	})

	go http.ListenAndServe(":"+port, mux)
}

// handleReplay replays a captured request to the local server
func handleReplay(w http.ResponseWriter, r *http.Request, idStr string) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	exchange, ok := GetExchange(id)
	if !ok {
		http.Error(w, "Exchange not found", http.StatusNotFound)
		return
	}

	if localPort == "" {
		http.Error(w, "Replay not configured (no local port)", http.StatusInternalServerError)
		return
	}

	// Reconstruct the request
	reqURL := "http://localhost:" + localPort + exchange.Request.URL
	req, err := http.NewRequest(exchange.Request.Method, reqURL, bytes.NewReader([]byte(exchange.Request.Body)))
	if err != nil {
		http.Error(w, "Failed to create request: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Copy headers
	for k, vv := range exchange.Request.Headers {
		for _, v := range vv {
			req.Header.Add(k, v)
		}
	}

	// Execute request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Replay failed: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	// Return response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  resp.StatusCode,
		"headers": resp.Header,
		"body":    string(respBody),
	})
}

// Legacy function for backward compatibility
func AddRequest(method, host, path string, status int) {
	// Create a minimal exchange for backward compatibility
	mu.Lock()
	defer mu.Unlock()

	id := nextID
	nextID++

	exchange := HTTPExchange{
		ID:        id,
		Timestamp: time.Now(),
		Request: &HTTPRequest{
			Method: method,
			URL:    path,
			Headers: map[string][]string{
				"Host": {host},
			},
		},
	}

	if status > 0 {
		exchange.Response = &HTTPResponse{
			Status: status,
		}
	}

	exchanges = append([]HTTPExchange{exchange}, exchanges...)
	if len(exchanges) > 100 {
		exchanges = exchanges[:100]
	}
}
