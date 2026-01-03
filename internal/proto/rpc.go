// Package proto defines the JSON-RPC message types for Control Plane <-> Agent communication.
package proto

// Request represents a JSON-RPC 2.0 request.
type Request struct {
	JSONRPC string         `json:"jsonrpc"`
	Method  string         `json:"method"`
	Params  map[string]any `json:"params,omitempty"`
	ID      any            `json:"id,omitempty"`
}

// Response represents a JSON-RPC 2.0 response.
type Response struct {
	JSONRPC string    `json:"jsonrpc"`
	Result  any       `json:"result,omitempty"`
	Error   *RPCError `json:"error,omitempty"`
	ID      any       `json:"id"`
}

// RPCError represents a JSON-RPC 2.0 error object.
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// Standard JSON-RPC 2.0 error codes.
const (
	ParseError     = -32700
	InvalidRequest = -32600
	MethodNotFound = -32601
	InvalidParams  = -32602
	InternalError  = -32603
)

// ExecParams contains parameters for the "exec" method.
type ExecParams struct {
	Cmd     string            `json:"cmd"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	Timeout int64             `json:"timeout,omitempty"` // milliseconds
}

// ReplStartParams contains parameters for the "repl.start" method.
type ReplStartParams struct {
	Cmd  string            `json:"cmd"`
	Args []string          `json:"args,omitempty"`
	Env  map[string]string `json:"env,omitempty"`
}

// ReplInputParams contains parameters for the "repl.input" method.
type ReplInputParams struct {
	Data string `json:"data"`
}

// StreamEvent represents an event streamed from Agent to Control Plane.
// These are sent as JSON-RPC notifications (no ID).

// StdoutEvent is sent when the process writes to stdout.
type StdoutEvent struct {
	Chunk string `json:"chunk"`
}

// StderrEvent is sent when the process writes to stderr.
type StderrEvent struct {
	Chunk string `json:"chunk"`
}

// ExitEvent is sent when the process terminates.
type ExitEvent struct {
	Code int `json:"code"`
}

// ArtifactEvent is sent when a new file is detected in /output.
type ArtifactEvent struct {
	Path       string `json:"path"`
	MIME       string `json:"mime"`
	DataBase64 string `json:"data_base64,omitempty"`
	URL        string `json:"url,omitempty"` // For large files uploaded to S3
}

// ErrorEvent is sent when an error occurs during execution.
type ErrorEvent struct {
	Message string `json:"message"`
}

// NewRequest creates a new JSON-RPC 2.0 request.
func NewRequest(method string, params map[string]any, id any) *Request {
	return &Request{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
		ID:      id,
	}
}

// NewNotification creates a JSON-RPC 2.0 notification (no response expected).
func NewNotification(method string, params map[string]any) *Request {
	return &Request{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}
}

// NewSuccessResponse creates a successful JSON-RPC 2.0 response.
func NewSuccessResponse(id any, result any) *Response {
	return &Response{
		JSONRPC: "2.0",
		Result:  result,
		ID:      id,
	}
}

// NewErrorResponse creates an error JSON-RPC 2.0 response.
func NewErrorResponse(id any, code int, message string) *Response {
	return &Response{
		JSONRPC: "2.0",
		Error: &RPCError{
			Code:    code,
			Message: message,
		},
		ID: id,
	}
}
