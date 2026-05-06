# Boxed Benchmark Harness

Reproducibility for the *Boxed: A Sovereign, Polyglot Sandbox Substrate* paper.

## Hardware reported in the paper

- Apple M-series, macOS 15, Docker Desktop 4.x (primary)
- Optional re-run on Linux 6.x bare metal (Hetzner CX22 or similar)

## Quickstart

```bash
# Terminal 1: control plane
export BOXED_API_KEY=bench
cd .. && make build && ./bin/boxed serve --api-key $BOXED_API_KEY

# Terminal 2: bench
cd bench
make all
```

## Scenarios

| Scenario     | CSV                  | Description                             |
|--------------|----------------------|-----------------------------------------|
| coldstart    | results/coldstart.csv| Sequential create→exec→destroy timings  |
| throughput   | results/throughput.csv| RPS at concurrency = 1..64             |
| overhead     | results/overhead.csv | Per-sandbox idle RSS and CPU            |
| escapes      | results/escapes.csv  | 12 adversarial escape attempts          |
| agent        | results/agent_trace.csv | LLM-driven HumanEval-style end-to-end |

## Required env

- `BOXED_API_KEY` — must match what `boxed serve` was started with
- `GEMINI_API_KEY` — only for `agent` scenario
- `GEMINI_MODEL` — optional override (default `gemini-2.0-flash`)
