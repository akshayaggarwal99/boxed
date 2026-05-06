# Boxed Benchmark Harness

Reproducibility for the *Boxed: A Sovereign, Polyglot Sandbox Substrate* paper.

## Hardware reported in the paper

- MacBook Pro, Apple M1 Pro, 16 GB unified memory, macOS, Docker Desktop (primary; what the paper reports)
- Optional re-run on a quiesced Linux host (recommended for cleaner numbers)

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
- `GEMINI_MODEL` — optional override (default `gemini-2.5-flash`)
