# Boxed Benchmark Harness

Reproducibility for *Boxed: A Docker-Based Code-Execution Substrate for Autonomous Code-Generating Agents*.

## Hardware reported in the paper

- MacBook Pro, Apple M1 Pro, 16 GB, macOS 26.5, Docker Engine 29.5 inside a colima VM (vz, 4 vCPU, 8 GiB). See `results/hardened-2026-09/ENV.txt`.
- The May 2026 Docker Desktop traces (pre-hardening driver) are kept in `results/legacy-2026-05-docker-desktop/` for audit only.
- A native Linux rerun is welcome: set `HOST_SSH=""` so the escape probe reads cgroup files locally.

## Quickstart

```bash
# Terminal 1: control plane (DOCKER_HOST needed with colima)
export BOXED_API_KEY=bench DOCKER_HOST=unix://$HOME/.colima/default/docker.sock
cd .. && make build && ./bin/boxed serve --api-key $BOXED_API_KEY

# Terminal 2: full campaign (5x cold start + raw-Docker baselines, 10 throughput sweeps,
# overhead, 3x escape probe), then the agent trace, then tables + figures
cd bench && make build
RUN=hardened-$(date +%Y-%m) ./run-all.sh
make agent RUN=hardened-$(date +%Y-%m)   # needs ANTHROPIC_API_KEY
make plots RUN=hardened-$(date +%Y-%m)
```

## Scenarios

| Scenario     | CSV                  | Description                             |
|--------------|----------------------|-----------------------------------------|
| coldstart    | r*/coldstart.csv      | Sequential create→exec→destroy through Boxed |
| baseline     | r*/baseline_{default,hardened}.csv | Same lifecycle via the Docker Engine API, no Boxed |
| throughput   | tp*/throughput.csv    | create→destroy rate at concurrency 1..32 |
| overhead     | overhead.csv          | Per-sandbox idle working set and CPU    |
| escapes      | escapes_r*.csv        | 12 vectors, signature + post-condition verdicts, evidence |
| agent        | agent_trace.csv/.jsonl| claude-opus-5 on official HumanEval/0..19; raw completions in the JSONL |

## Required env

- `BOXED_API_KEY` — must match what `boxed serve` was started with
- `ANTHROPIC_API_KEY` — only for the `agent` scenario
- `BOXED_AGENT_MODEL` — optional override (default `claude-opus-5`)
- `HUMANEVAL_PATH` — optional (default `data/HumanEval.jsonl`, the official release)
- `HOST_SSH` — how the escape probe reaches the Docker host to read cgroup files (default `colima ssh --`; use `""` on native Linux)
