package docker

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/akshayaggarwal99/boxed/internal/driver"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/rs/zerolog/log"
)

const (
	DriverName      = "docker"
	AgentBinaryPath = "/usr/local/bin/boxed-agent"
	ManagedLabel    = "xyz.boxed.managed"
)

// DockerDriver implements the driver.Driver interface using the Docker engine.
type DockerDriver struct {
	cli *client.Client
	// hostAgentPath is the path to the compiled agent binary on the host
	hostAgentPath string
}

// New creates a new DockerDriver.
// cfg["agent_path"] can be used to specify the host path to the boxed-agent binary.
func New(cfg map[string]any) (driver.Driver, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}

	// Perform startup cleanup of orphaned containers
	go cleanupOrphans(cli)

	agentPath := "boxed-agent" // Default expectation: in PATH or current dir?
	if p, ok := cfg["agent_path"].(string); ok {
		agentPath = p
	} else {
		// Fallback: try to find it in strict project path for dev
		// In a real release, this would be packaged differently
		absPath, _ := filepath.Abs("agent/target/release/boxed-agent")
		agentPath = absPath
	}

	return &DockerDriver{
		cli:           cli,
		hostAgentPath: agentPath,
	}, nil
}

func init() {
	driver.RegisterDriver(DriverName, New)
}

func (d *DockerDriver) DriverName() string {
	return DriverName
}

func (d *DockerDriver) Healthy(ctx context.Context) error {
	_, err := d.cli.Ping(ctx)
	return err
}

func (d *DockerDriver) Close() error {
	return d.cli.Close()
}

func cleanupOrphans(cli *client.Client) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	log.Info().Msg("Performing startup garbage collection of orphaned containers...")
	list, err := cli.ContainerList(ctx, types.ContainerListOptions{
		All:     true,
		Filters: filters.NewArgs(filters.Arg("label", ManagedLabel+"=true")),
	})
	if err != nil {
		log.Warn().Err(err).Msg("Failed to list orphaned containers")
		return
	}

	count := 0
	for _, c := range list {
		// Stop/Remove them
		err := cli.ContainerRemove(ctx, c.ID, types.ContainerRemoveOptions{Force: true})
		if err != nil {
			log.Warn().Str("id", c.ID).Err(err).Msg("Failed to remove orphan")
		} else {
			count++
		}
	}
	if count > 0 {
		log.Info().Int("count", count).Msg("Removed orphaned containers")
	} else {
		log.Info().Msg("No orphans found")
	}
}

func (d *DockerDriver) Create(ctx context.Context, cfg driver.SandboxConfig) (string, error) {
	if err := cfg.Validate(); err != nil {
		return "", err
	}

	// Prepare resources
	// NanoCPUs: 1.0 = 1e9.
	nanoCPUs := int64(cfg.CPUCores * 1e9)
	memoryBytes := cfg.MemoryMB * 1024 * 1024

	hostConfig := &container.HostConfig{
		Resources: container.Resources{
			NanoCPUs: nanoCPUs,
			Memory:   memoryBytes,
		},
		Mounts: []mount.Mount{
			// Mount the agent binary
			{
				Type:     mount.TypeBind,
				Source:   d.hostAgentPath,
				Target:   AgentBinaryPath,
				ReadOnly: true,
			},
			// Ephemeral /tmp
			{
				Type:   mount.TypeTmpfs,
				Target: "/tmp",
			},
			// Ephemeral /output
			{
				Type:   mount.TypeTmpfs,
				Target: "/output",
			},
		},
		// Drop capabilities for security (basic set)
		// CapDrop: []string{"ALL"},
		// CapAdd:  []string{"NET_BIND_SERVICE"},
	}

	// Network configuration
	if !cfg.EnableNetworking {
		hostConfig.NetworkMode = "none"
	}

	// Environment variables
	env := []string{
		"BOXED_AGENT_MODE=docker",
	}
	for k, v := range cfg.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	// For the "warm pool" or just basic execution, the container needs to stay alive.
	// We use "tail -f /dev/null" as the entrypoint so we can exec into it later.
	// NOTE: The image must have 'tail' installed.
	// Alternatively, we could run the agent as the primary process, but the blueprint
	// suggests "Attaches to Exec stream", which implies we might exec *into* it.
	// Let's stick to the "sleep infinity" pattern for the container, then exec the agent.

	// Check if image exists, pull if not (optional, but good for UX)
	// d.pullImage(ctx, cfg.Image) // Simplified: assume user has image or Docker will handle

	// Ensure image exists locally, otherwise pull it
	_, _, err := d.cli.ImageInspectWithRaw(ctx, cfg.Image)
	if client.IsErrNotFound(err) {
		log.Info().Str("image", cfg.Image).Msg("Image not found locally, pulling...")
		reader, err := d.cli.ImagePull(ctx, cfg.Image, types.ImagePullOptions{})
		if err != nil {
			return "", fmt.Errorf("failed to pull image %s: %w", cfg.Image, err)
		}
		// Drain the output to ensure pull completes
		io.Copy(io.Discard, reader)
		reader.Close()
	} else if err != nil {
		return "", fmt.Errorf("failed to inspect image: %w", err)
	}

	labels := cfg.Labels
	if labels == nil {
		labels = make(map[string]string)
	}
	labels[ManagedLabel] = "true"

	resp, err := d.cli.ContainerCreate(ctx,
		&container.Config{
			Image:      cfg.Image,
			Cmd:        []string{"tail", "-f", "/dev/null"},
			Env:        env,
			Labels:     labels,
			WorkingDir: cfg.WorkDir,
		},
		hostConfig,
		nil,
		nil,
		"", // let Docker assign name or generate one
	)
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	// Context Injection
	for _, file := range cfg.Context {
		data, err := base64.StdEncoding.DecodeString(file.ContentBase64)
		if err != nil {
			log.Error().Err(err).Str("path", file.Path).Msg("Failed to decode context file")
			continue
		}

		// Ensure absolute path
		targetPath := file.Path
		if !filepath.IsAbs(targetPath) {
			targetPath = filepath.Join(cfg.WorkDir, targetPath)
		}

		if err := d.PutFile(ctx, resp.ID, targetPath, bytes.NewReader(data)); err != nil {
			log.Error().Err(err).Str("path", file.Path).Msg("Failed to inject context file")
			d.Stop(ctx, resp.ID)
			return "", fmt.Errorf("failed to inject file %s: %w", file.Path, err)
		}
	}

	// Enforce TTL
	go func(id string, timeout time.Duration) {
		time.Sleep(timeout)
		// Check if it still exists? Stop is idempotent.
		// Use a fresh context for cleanup
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		d.Stop(ctx, id)
	}(resp.ID, cfg.Timeout)

	return resp.ID, nil
}

func (d *DockerDriver) Start(ctx context.Context, id string) error {
	if err := d.cli.ContainerStart(ctx, id, types.ContainerStartOptions{}); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	// Wait a brief moment to ensure it's actually running?
	// Usually ContainerStart returns once the process is launched.
	return nil
}

func (d *DockerDriver) Stop(ctx context.Context, id string) error {
	// Force remove (kills + deletes)
	opts := types.ContainerRemoveOptions{
		Force:         true,
		RemoveVolumes: true,
	}
	if err := d.cli.ContainerRemove(ctx, id, opts); err != nil {
		if client.IsErrNotFound(err) {
			return driver.ErrSandboxNotFound
		}
		return fmt.Errorf("failed to stop/remove container: %w", err)
	}
	return nil
}

func (d *DockerDriver) Connect(ctx context.Context, id string) (io.ReadWriteCloser, error) {
	// Check if running
	info, err := d.cli.ContainerInspect(ctx, id)
	if err != nil {
		if client.IsErrNotFound(err) {
			return nil, driver.ErrSandboxNotFound
		}
		return nil, err
	}
	if !info.State.Running {
		return nil, driver.ErrSandboxNotRunning
	}

	// Exec the agent
	execConfig := types.ExecConfig{
		Cmd:          []string{AgentBinaryPath},
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          false, // Raw stream for JSON-RPC
	}

	execIDResp, err := d.cli.ContainerExecCreate(ctx, id, execConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create exec: %w", err)
	}

	resp, err := d.cli.ContainerExecAttach(ctx, execIDResp.ID, types.ExecStartCheck{})
	if err != nil {
		return nil, fmt.Errorf("failed to attach to exec: %w", err)
	}

	// response.Conn is the underlying connection.
	// response.Reader handles the multiplexing if Tty is false (Docker headers).
	// However, ContainerExecAttach with Tty=false returns a reader that includes Docker header bytes (Stream types).
	// We need stdcopy or similar if we want to separate stdout/stderr, OR we just trust the connection
	// BUT, strict JSON-RPC requires a clean stream.
	// The `docker/pkg/stdcopy` package exists to de-multiplex.
	//
	// WAIT. The agent communicates over stdin/stdout.
	// If Tty=false, Docker multiplexes stdout and stderr on the same connection with headers.
	// The agent might write debug logs to stderr and JSON-RPC to stdout.
	// We MUST de-multiplex.
	//
	// However, `resp.Conn` is a raw net.Conn.
	// We can wrap this.
	// actually `types.HijackedResponse` has a Reader.

	// PROBLEM: We need a clean io.ReadWriteCloser that sends writes to container stdin
	// and reads from container stdout (ignoring stderr or logging it).
	//
	// If the agent writes JSON-RPC to stdout, we need to strip the Docker headers.

	return NewDockerStream(resp), nil
}

func (d *DockerDriver) Info(ctx context.Context, id string) (*driver.SandboxInfo, error) {
	json, err := d.cli.ContainerInspect(ctx, id)
	if err != nil {
		if client.IsErrNotFound(err) {
			return nil, driver.ErrSandboxNotFound
		}
		return nil, err
	}

	state := driver.StateStopped
	if json.State.Running {
		state = driver.StateReady // simplified
	} else if json.State.Dead || json.State.OOMKilled {
		state = driver.StateError
	}

	// Map created time
	created, _ := time.Parse(time.RFC3339Nano, json.Created)

	return &driver.SandboxInfo{
		ID:         json.ID,
		State:      state,
		CreatedAt:  created,
		DriverType: DriverName,
		IPAddress:  json.NetworkSettings.IPAddress,
		// Config: rebuild from inspect if needed, or skipped
	}, nil
}

func (d *DockerDriver) List(ctx context.Context, states []driver.SandboxState) ([]*driver.SandboxInfo, error) {
	// Not implemented efficiently for now
	containers, err := d.cli.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		return nil, err
	}

	var results []*driver.SandboxInfo
	for _, c := range containers {
		// Filter by label to only show boxed containers?
		// For now, allow all or check a label if we added one.

		state := driver.StateStopped
		if c.State == "running" {
			state = driver.StateReady
		}

		results = append(results, &driver.SandboxInfo{
			ID:         c.ID,
			State:      state,
			DriverType: DriverName,
		})
	}
	return results, nil
}

// DockerWrapper handles the multiplexed stream from Docker (StdCopy format)
type DockerStream struct {
	resp types.HijackedResponse
	// We need a way to demux.
	// Since we can't implement stdcopy on a single Read() call easily matching the RWC interface directly
	// without buffering.
	//
	// For MVP, if we run with Tty=true, the streams are combined and raw.
	// BUT Tty=true causes CR/LF issues with JSON RPC sometimes.
	//
	// Let's use stdcopy logic in a goroutine to pipe stdout to a pipe, and log stderr.
	reader *io.PipeReader
	writer *io.PipeWriter
}

func NewDockerStream(resp types.HijackedResponse) *DockerStream {
	pr, pw := io.Pipe()
	ds := &DockerStream{
		resp:   resp,
		reader: pr,
		writer: pw,
	}

	go ds.demux()

	return ds
}

func (ds *DockerStream) demux() {
	// Use stdcopy.StdCopy to demultiplex
	// stdout goes to our pipe (to the driver user)
	// stderr goes to logger (or discarded)

	// We need a dummy internal import or custom implementation if we want to avoid
	// heavyweight dependencies, but since we have `docker/docker`, we likely have `pkg/stdcopy`.
	// Wait, `stdcopy` is in `github.com/docker/docker/pkg/stdcopy`.
	// I need to check imports.
	// Assuming it's available or we implement a basic demux.
	// Docker header: [8]byte: STREAM_TYPE, 0, 0, 0, SIZE_b3, SIZE_b2, SIZE_b1, SIZE_b0

	// For now, let's assume we can import it. If not, I'll add it to imports.
	// Implementation below:

	// Since I can't easily edit imports in this single file block without risking breaking context,
	// I will just implement a simple loop.

	defer ds.writer.Close()

	for {
		header := make([]byte, 8)
		_, err := io.ReadFull(ds.resp.Reader, header)
		if err != nil {
			return
		}

		// Parse header
		// streamType := header[0] // 1=stdin, 2=stdout, 0=stdin?
		payloadSize := int(header[4])<<24 | int(header[5])<<16 | int(header[6])<<8 | int(header[7])

		if payloadSize < 0 {
			return
		}

		// Copy payload
		switch header[0] {
		case 1: // Stdout
			_, err := io.CopyN(ds.writer, ds.resp.Reader, int64(payloadSize))
			if err != nil {
				return
			}
		case 2: // Stderr
			// Discard or log
			// We discard for now to keep the JSON-RPC stream clean
			io.CopyN(os.Stderr, ds.resp.Reader, int64(payloadSize))
		default:
			// Stream info or other
			io.CopyN(io.Discard, ds.resp.Reader, int64(payloadSize))
		}
	}
}

func (ds *DockerStream) Read(p []byte) (n int, err error) {
	return ds.reader.Read(p)
}

func (ds *DockerStream) Write(p []byte) (n int, err error) {
	return ds.resp.Conn.Write(p)
}

func (ds *DockerStream) Close() error {
	ds.resp.Close()
	ds.writer.Close() // Ensure pipe ensures EOF
	return nil
}
