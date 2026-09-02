package scenarios

// Baseline lifecycle measured directly against the Docker Engine API, with no
// Boxed control plane and no in-sandbox agent. The same image, placeholder
// command, and (in hardened mode) the same HostConfig the Boxed driver applies
// are used, so Boxed minus baseline isolates the cost of the HTTP control
// plane, the agent launch, and the JSON-RPC exec path.
//
// Modes:
//   default  - stock `docker run` defaults: writable rootfs, default caps,
//              bridge network, no pids limit, no resource quota.
//   hardened - ReadonlyRootfs, CapDrop ALL, no-new-privileges, PidsLimit 256,
//              network none, 1 CPU, 512 MiB, tmpfs /tmp /output /workspace.

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

const baselineImage = "python:3.10-slim"

func baselineHostConfig(mode string) *container.HostConfig {
	rt := os.Getenv("BOXED_DOCKER_RUNTIME") // same runtime switch the driver honours
	if mode != "hardened" {
		return &container.HostConfig{Runtime: rt}
	}
	pids := int64(256)
	var nano int64 = 1e9
	if os.Getenv("BOXED_CPU_QUOTA") == "off" {
		nano = 0
	}
	net := "none"
	if n := os.Getenv("BOXED_ISOLATED_NETWORK"); n != "" {
		net = n
	}
	return &container.HostConfig{
		Resources: container.Resources{
			NanoCPUs:  nano,
			Memory:    512 * 1024 * 1024,
			PidsLimit: &pids,
		},
		ReadonlyRootfs: true,
		CapDrop:        []string{"ALL"},
		SecurityOpt:    []string{"no-new-privileges:true"},
		Mounts: []mount.Mount{
			{Type: mount.TypeTmpfs, Target: "/tmp", TmpfsOptions: &mount.TmpfsOptions{Mode: 01777}},
			{Type: mount.TypeTmpfs, Target: "/output", TmpfsOptions: &mount.TmpfsOptions{Mode: 01777}},
			{Type: mount.TypeTmpfs, Target: "/workspace", TmpfsOptions: &mount.TmpfsOptions{Mode: 01777}},
		},
		NetworkMode: container.NetworkMode(net),
		Runtime:     rt,
	}
}

func RunBaseline(mode string, n int, outDir string) {
	if mode != "default" && mode != "hardened" {
		log.Fatalf("baseline mode must be default|hardened, got %q", mode)
	}
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()

	f, err := os.Create(filepath.Join(outDir, "baseline_"+mode+".csv"))
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	w := csv.NewWriter(f)
	defer w.Flush()
	_ = w.Write([]string{"i", "create_ms", "first_exec_ms", "destroy_ms", "total_ms", "exit_code"})

	hc := baselineHostConfig(mode)
	for i := 0; i < n; i++ {
		t0 := time.Now()
		resp, err := cli.ContainerCreate(ctx,
			&container.Config{
				Image:      baselineImage,
				Cmd:        []string{"tail", "-f", "/dev/null"},
				WorkingDir: "/workspace",
				Labels:     map[string]string{"boxed.bench.baseline": mode},
			}, hc, nil, nil, "")
		if err != nil {
			log.Printf("iter %d create: %v", i, err)
			continue
		}
		id := resp.ID
		if err := cli.ContainerStart(ctx, id, types.ContainerStartOptions{}); err != nil {
			log.Printf("iter %d start: %v", i, err)
			_ = cli.ContainerRemove(ctx, id, types.ContainerRemoveOptions{Force: true})
			continue
		}
		t1 := time.Now()

		code := -1
		ex, err := cli.ContainerExecCreate(ctx, id, types.ExecConfig{
			Cmd:          []string{"python3", "-c", "print('ok')"},
			AttachStdout: true,
			AttachStderr: true,
		})
		if err != nil {
			log.Printf("iter %d exec create: %v", i, err)
		} else {
			att, err := cli.ContainerExecAttach(ctx, ex.ID, types.ExecStartCheck{})
			if err != nil {
				log.Printf("iter %d exec attach: %v", i, err)
			} else {
				var so, se bytes.Buffer
				_, _ = stdcopy.StdCopy(&so, &se, att.Reader)
				att.Close()
				if insp, err := cli.ContainerExecInspect(ctx, ex.ID); err == nil {
					code = insp.ExitCode
				}
				_ = io.Discard
			}
		}
		t2 := time.Now()

		if err := cli.ContainerRemove(ctx, id, types.ContainerRemoveOptions{Force: true}); err != nil {
			log.Printf("iter %d remove: %v", i, err)
		}
		t3 := time.Now()

		_ = w.Write([]string{
			strconv.Itoa(i),
			fmt.Sprintf("%.3f", ms(t1.Sub(t0))),
			fmt.Sprintf("%.3f", ms(t2.Sub(t1))),
			fmt.Sprintf("%.3f", ms(t3.Sub(t2))),
			fmt.Sprintf("%.3f", ms(t3.Sub(t0))),
			strconv.Itoa(code),
		})
		w.Flush()
	}
}
