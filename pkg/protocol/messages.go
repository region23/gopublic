package protocol

// ErrorCode represents structured error codes for protocol responses.
type ErrorCode string

const (
	ErrorCodeNone             ErrorCode = ""
	ErrorCodeInvalidToken     ErrorCode = "invalid_token"
	ErrorCodeAlreadyConnected ErrorCode = "already_connected"
	ErrorCodeNoDomains        ErrorCode = "no_domains"
)

// AuthRequest is the first message sent by the client to authenticate using a token.
type AuthRequest struct {
	Token string `json:"token"`
	Force bool   `json:"force,omitempty"` // Force disconnect existing session
}

// TunnelRequest follows authentication to request binding of specific domains.
type TunnelRequest struct {
	RequestedDomains []string `json:"requested_domains"`
}

// ServerStats contains user bandwidth statistics from the server.
type ServerStats struct {
	BandwidthToday int64 `json:"bandwidth_today"` // Bytes used today
	BandwidthTotal int64 `json:"bandwidth_total"` // Total bytes used all time
	BandwidthLimit int64 `json:"bandwidth_limit"` // Daily bandwidth limit in bytes
}

// InitResponse is sent by the server to indicate success or failure of the handshake.
type InitResponse struct {
	Success   bool      `json:"success"`
	Error     string    `json:"error,omitempty"`
	ErrorCode ErrorCode `json:"error_code,omitempty"` // Structured error code
	// AssignedDomains could be useful if we support random assignment (future),
	// but for now it confirms what was bound.
	BoundDomains []string     `json:"bound_domains,omitempty"`
	ServerStats  *ServerStats `json:"server_stats,omitempty"` // User bandwidth statistics
}
