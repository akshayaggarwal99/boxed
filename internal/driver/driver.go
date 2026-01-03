// Package driver defines the abstraction layer for sandbox backends.
//
// This interface is the core of Boxed's "run anywhere" architecture, allowing
// seamless switching between Docker (local development), Firecracker (production),
// and future backends like WebAssembly sandboxes.
package driver

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"
)

// Common errors returned by Driver implementations.
var (
	// ErrSandboxNotFound indicates the requested sandbox does not exist.
	ErrSandboxNotFound = errors.New("sandbox not found")

	// ErrSandboxAlreadyRunning indicates an attempt to start an already running sandbox.
	ErrSandboxAlreadyRunning = errors.New("sandbox already running")

	// ErrSandboxNotRunning indicates an attempt to stop or connect to a non-running sandbox.
	ErrSandboxNotRunning = errors.New("sandbox not running")

	// ErrConnectionFailed indicates failure to establish connection to the agent.
	ErrConnectionFailed = errors.New("failed to connect to sandbox agent")

	// ErrResourceExhausted indicates no resources available to create new sandboxes.
	ErrResourceExhausted = errors.New("resource limit exhausted")

	// ErrTimeout indicates an operation exceeded its deadline.
	ErrTimeout = errors.New("operation timed out")

	// ErrInvalidConfig indicates the provided configuration is invalid.
	ErrInvalidConfig = errors.New("invalid sandbox configuration")
)

// SandboxState represents the current state of a sandbox.
type SandboxState string

const (
	// StateCreating indicates the sandbox is being provisioned.
	StateCreating SandboxState = "creating"

	// StateReady indicates the sandbox is running and agent is responsive.
	StateReady SandboxState = "ready"

	// StateStopping indicates the sandbox is being terminated.
	StateStopping SandboxState = "stopping"

	// StateStopped indicates the sandbox has been terminated.
	StateStopped SandboxState = "stopped"

	// StateError indicates the sandbox encountered an unrecoverable error.
	StateError SandboxState = "error"
)

// SandboxConfig defines the specifications for the requested execution environment.
// This is the contract between the Control Plane and the Driver implementations.
type SandboxConfig struct {
	// Image specifies the base environment (e.g., "boxed-python:3.9", "boxed-node:20")
	Image string `json:"image"`

	// MemoryMB sets the memory limit in megabytes (default: 512)
	MemoryMB int64 `json:"memory_mb"`

	// CPUCores sets the CPU limit as fractional cores (e.g., 1.0 = 1 core, 0.5 = half core)
	CPUCores float64 `json:"cpu_cores"`

	// Env contains environment variables to inject into the sandbox
	Env map[string]string `json:"env,omitempty"`

	// Timeout specifies the maximum lifetime of the sandbox (default: 5 minutes)
	Timeout time.Duration `json:"timeout"`

	// EnableNetworking allows outbound network access (subject to egress filtering)
	EnableNetworking bool `json:"enable_networking"`

	// AllowedHosts is the whitelist for outbound connections when networking is enabled
	// Supports wildcards (e.g., "*.google.com", "pypi.org")
	AllowedHosts []string `json:"allowed_hosts,omitempty"`

	// WorkDir sets the working directory inside the sandbox (default: "/workspace")
	WorkDir string `json:"work_dir,omitempty"`

	// Labels are arbitrary key-value pairs for metadata (e.g., user_id, session_id)
	Labels map[string]string `json:"labels,omitempty"`

	// NetworkPolicy controls internet access
	NetworkPolicy NetworkPolicy `json:"network_policy"`

	// Context contains files to inject at startup
	Context []FileInjection `json:"context,omitempty"`
}

// NetworkPolicy defines network access rules
type NetworkPolicy struct {
	EnableInternet bool     `json:"enable_internet"`
	AllowDomains   []string `json:"allow_domains,omitempty"`
}

// FileInjection represents a file to be written to the sandbox at boot
type FileInjection struct {
	Path          string `json:"path"`
	ContentBase64 string `json:"content_base64"`
}

// FileEntry represents a file or directory in the sandbox
type FileEntry struct {
	Name         string    `json:"name"`
	Path         string    `json:"path"`
	Size         int64     `json:"size"`
	Mode         int64     `json:"mode"`
	IsDir        bool      `json:"is_dir"`
	LastModified time.Time `json:"last_modified"`
}

// Driver is the abstraction interface for sandbox backends.
// Implementations must be safe for concurrent use.
//
// The lifecycle of a sandbox is:
//  1. Create() - Provisions the environment (container/VM)
//  2. Start() - Boots the sandbox and waits for agent readiness
//  3. Connect() - Establishes a raw stream to the internal agent
//  4. Stop() - Terminates and cleans up all resources
//
// All methods accept a context.Context for timeout/cancellation support.
// Implementations should respect context deadlines and return ErrTimeout
// if the operation cannot complete in time.
type Driver interface {
	// Create provisions a new sandbox environment based on the given configuration.
	// It returns a unique sandbox ID that can be used for subsequent operations.
	//
	// The sandbox is NOT started automatically. Call Start() to boot it.
	// This separation allows for "warm pool" implementations where sandboxes
	// are pre-created but only started on demand.
	//
	// Returns ErrInvalidConfig if the configuration is invalid.
	// Returns ErrResourceExhausted if no resources are available.
	Create(ctx context.Context, cfg SandboxConfig) (id string, err error)

	// Start boots a previously created sandbox and waits for the internal
	// agent to become responsive.
	//
	// After Start returns successfully, the sandbox is in StateReady and
	// Connect() can be called to establish communication with the agent.
	//
	// Returns ErrSandboxNotFound if the sandbox doesn't exist.
	// Returns ErrSandboxAlreadyRunning if the sandbox is already running.
	// Returns ErrTimeout if the agent doesn't become responsive in time.
	Start(ctx context.Context, id string) error

	// Stop terminates a running sandbox and releases all associated resources.
	// This includes stopping the container/VM, cleaning up network interfaces,
	// and removing ephemeral storage.
	//
	// Stop is idempotent - calling it on an already stopped sandbox is a no-op.
	//
	// Returns ErrSandboxNotFound if the sandbox doesn't exist.
	Stop(ctx context.Context, id string) error

	// Connect establishes a bidirectional stream to the Boxed Agent running
	// inside the sandbox. The returned io.ReadWriteCloser is used for
	// JSON-RPC 2.0 communication.
	//
	// For Docker: This attaches to a container exec stream over TCP.
	// For Firecracker: This connects via vsock.
	//
	// The caller is responsible for closing the connection when done.
	//
	// Returns ErrSandboxNotRunning if the sandbox is not in StateReady.
	// Returns ErrConnectionFailed if the connection cannot be established.
	Connect(ctx context.Context, id string) (io.ReadWriteCloser, error)

	// FILESYSTEM API

	// ListFiles returns a list of files in the specified directory path.
	ListFiles(ctx context.Context, id, path string) ([]*FileEntry, error)

	// PutFile uploads a file to the sandbox.
	// content is a reader for the raw file data.
	PutFile(ctx context.Context, id, path string, content io.Reader) error

	// GetFile downloads a file from the sandbox.
	// Returns a reader for the raw file data. Caller must Close it.
	GetFile(ctx context.Context, id, path string) (io.ReadCloser, error)

	// Info returns runtime information about a sandbox.
	//
	// Returns ErrSandboxNotFound if the sandbox doesn't exist.
	Info(ctx context.Context, id string) (*SandboxInfo, error)

	// List returns all sandboxes managed by this driver, optionally filtered
	// by state. Pass nil for states to list all sandboxes.
	List(ctx context.Context, states []SandboxState) ([]*SandboxInfo, error)

	// DriverName returns the identifier for this driver type (e.g., "docker", "firecracker").
	DriverName() string

	// Healthy performs a health check on the driver's backend.
	// Returns nil if the backend is operational.
	Healthy(ctx context.Context) error

	// Close releases any resources held by the driver itself.
	// After Close is called, the driver should not be used.
	Close() error
}

// Validate checks if the configuration is valid and applies defaults.
func (c *SandboxConfig) Validate() error {
	if c.Image == "" {
		return fmt.Errorf("%w: image is required", ErrInvalidConfig)
	}

	// Apply defaults
	if c.MemoryMB <= 0 {
		c.MemoryMB = 512
	}
	if c.CPUCores <= 0 {
		c.CPUCores = 1.0
	}
	if c.Timeout <= 0 {
		c.Timeout = 5 * time.Minute
	}
	if c.WorkDir == "" {
		c.WorkDir = "/workspace"
	}

	// Validate constraints
	if c.MemoryMB > 8192 {
		return fmt.Errorf("%w: memory cannot exceed 8GB", ErrInvalidConfig)
	}
	if c.CPUCores > 4.0 {
		return fmt.Errorf("%w: CPU cannot exceed 4 cores", ErrInvalidConfig)
	}
	if c.Timeout > 30*time.Minute {
		return fmt.Errorf("%w: timeout cannot exceed 30 minutes", ErrInvalidConfig)
	}

	return nil
}

// SandboxInfo contains runtime information about a sandbox.
type SandboxInfo struct {
	// ID is the unique identifier for this sandbox
	ID string `json:"id"`

	// State is the current lifecycle state
	State SandboxState `json:"state"`

	// CreatedAt is when the sandbox was created
	CreatedAt time.Time `json:"created_at"`

	// Config is the original configuration used to create the sandbox
	Config SandboxConfig `json:"config"`

	// DriverType identifies which driver is managing this sandbox
	DriverType string `json:"driver_type"`

	// IPAddress is the internal IP address (if networking is enabled)
	IPAddress string `json:"ip_address,omitempty"`

	// Error contains the last error message if State is StateError
	Error string `json:"error,omitempty"`
}

// PooledDriver extends Driver with warm pool capabilities for sub-second startup.
type PooledDriver interface {
	Driver

	// WarmUp pre-creates sandboxes that can be claimed later for fast startup.
	// Returns the IDs of the created sandboxes.
	WarmUp(ctx context.Context, cfg SandboxConfig, count int) ([]string, error)

	// Claim takes a pre-warmed sandbox from the pool and returns its ID.
	// Returns ErrResourceExhausted if no sandboxes are available.
	Claim(ctx context.Context, cfg SandboxConfig) (id string, err error)

	// PoolStatus returns the current pool statistics.
	PoolStatus(ctx context.Context) (*PoolStats, error)
}

// PoolStats contains statistics about the warm pool.
type PoolStats struct {
	// Available is the number of pre-warmed sandboxes ready to be claimed
	Available int `json:"available"`

	// InUse is the number of claimed sandboxes currently running
	InUse int `json:"in_use"`

	// Total is the total capacity (Available + InUse)
	Total int `json:"total"`

	// Target is the desired number of warm sandboxes
	Target int `json:"target"`
}

// DriverFactory creates Driver instances based on configuration.
// This enables runtime selection of the backend (e.g., based on environment).
type DriverFactory func(cfg map[string]any) (Driver, error)

// Registry holds registered driver factories.
var driverRegistry = make(map[string]DriverFactory)

// RegisterDriver registers a driver factory under the given name.
// This is typically called in init() functions of driver implementations.
func RegisterDriver(name string, factory DriverFactory) {
	driverRegistry[name] = factory
}

// NewDriver creates a new Driver instance using the registered factory.
// Returns an error if the driver name is not registered.
func NewDriver(name string, cfg map[string]any) (Driver, error) {
	factory, ok := driverRegistry[name]
	if !ok {
		return nil, fmt.Errorf("unknown driver: %s", name)
	}
	return factory(cfg)
}

// AvailableDrivers returns the names of all registered drivers.
func AvailableDrivers() []string {
	names := make([]string, 0, len(driverRegistry))
	for name := range driverRegistry {
		names = append(names, name)
	}
	return names
}
