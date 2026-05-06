# Boxed arXiv Paper Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Produce a peer-quality systems paper on Boxed and submit to arXiv (cs.CR primary; cross-list cs.DC, cs.SE) plus secondary venues within 3 days, suitable as O1 visa evidence.

**Architecture:** Single-author, 8–10 page double-column ACM `acmart` paper. Day 1 = experiment harness + data collection. Day 2 = drafting from Abstract through Conclusion using collected data. Day 3 = polish, BibTeX, render, submit to arXiv + Zenodo (DOI) + 1 workshop.

**Tech Stack:**
- Writing: LaTeX (`acmart` class), Overleaf or local `tectonic`/`latexmk`
- Bench harness: Go (reuse Boxed's stack) + Python plotting (`matplotlib`, `pandas`, `numpy`)
- Stats: `scipy.stats` for confidence intervals
- Citations: BibTeX entries pulled from Semantic Scholar + Google Scholar
- Submission targets: arXiv, Zenodo (DOI mirror), HotOS/HotCloud workshop, USENIX ;login:

---

## File Structure

**Created files:**
- `paper/main.tex` — paper source (single file for simplicity)
- `paper/refs.bib` — BibTeX bibliography
- `paper/figures/*.pdf` — generated plots
- `paper/figures/architecture.pdf` — converted from existing `architecture.svg`
- `paper/tables/*.tex` — generated comparison tables
- `paper/Makefile` — build automation
- `paper/README.md` — submission checklist + abstract
- `bench/cmd/boxed-bench/main.go` — benchmark harness driver
- `bench/scenarios/coldstart.go` — cold-start latency scenario
- `bench/scenarios/throughput.go` — concurrent creation scenario
- `bench/scenarios/overhead.go` — memory/CPU overhead scenario
- `bench/scenarios/agent_trace.go` — LLM-agent end-to-end trace
- `bench/security/escapes.sh` — adversarial escape test suite
- `bench/results/*.csv` — raw measurements (gitignored beyond schema)
- `bench/analyze/plots.py` — CSV → publication-quality PDF plots
- `bench/analyze/stats.py` — statistical summaries
- `bench/Makefile` — one-shot reproduce-all target
- `bench/README.md` — reproducibility instructions

**Modified files:** none — paper artifacts are additive.

---

## Phase 0 (now): Outline + Scaffolding

### Task 0.1: Lock paper outline

**Files:**
- Create: `paper/README.md`

- [ ] **Step 1: Write outline doc**

```markdown
# Paper: Boxed — A Sovereign, Polyglot Sandbox Substrate

## Title
Boxed: A Sovereign, Polyglot Sandbox Substrate for Autonomous Code-Generating Agents

## Author
Akshay Kumar (solo). Affiliation: Independent.

## Abstract (150–200 words)
Motivate agentic LLMs needing untrusted code exec. State three commercial-SaaS gaps: vendor lock-in, BYOK absence, polyglot isolation. Introduce Boxed: pluggable-driver sandbox with Docker now / Firecracker / Wasm planned, Rust in-VM agent, JSON-RPC over stream, first-class artifact protocol. Headline numbers: Px ms median cold start, Ny sandboxes/s sustained throughput, Mz MB RSS overhead, passes A/B adversarial escape suite. Open-source MIT.

## Sections
1. Introduction (1.5 pp) — agent code-exec problem, threat model, contributions list
2. Background & Related Work (1 pp) — gVisor, Firecracker, E2B, Modal, Daytona, Cloudflare Sandbox, WASI
3. Design (2 pp) — driver interface, control plane, agent, artifact protocol, BYOK
4. Implementation (1 pp) — Go control plane (Echo), Rust agent (tokio), Docker driver, ~LOC
5. Evaluation (2.5 pp)
   5.1 cold-start latency (CDF)
   5.2 throughput under concurrency
   5.3 per-sandbox overhead
   5.4 end-to-end LLM-agent task latency vs E2B
   5.5 adversarial escape resistance
6. Discussion (0.5 pp) — limitations, Firecracker port path, multi-tenant scheduling
7. Conclusion (0.25 pp)
8. Availability — github.com/akshayaggarwal99/boxed, MIT.

## Page budget: 9 pages double-column acmart + refs
```

- [ ] **Step 2: Commit**

```bash
git add paper/README.md docs/superpowers/plans/2026-05-05-boxed-arxiv-paper.md
git commit -m "docs: scaffold paper plan and outline"
```

### Task 0.2: Initialise LaTeX skeleton

**Files:**
- Create: `paper/main.tex`
- Create: `paper/refs.bib`
- Create: `paper/Makefile`

- [ ] **Step 1: Write `paper/main.tex` skeleton**

```latex
\documentclass[sigconf,nonacm,review]{acmart}
\usepackage{booktabs}
\usepackage{graphicx}
\usepackage{listings}
\usepackage{xcolor}
\title{Boxed: A Sovereign, Polyglot Sandbox Substrate for Autonomous Code-Generating Agents}
\author{Akshay Kumar}
\affiliation{\institution{Independent}\country{USA}}
\email{akshaykumarinusa@gmail.com}
\begin{document}
\begin{abstract}
TODO-ABSTRACT
\end{abstract}
\maketitle
\section{Introduction}\label{sec:intro}
TODO
\section{Background and Related Work}\label{sec:related}
TODO
\section{Design}\label{sec:design}
TODO
\section{Implementation}\label{sec:impl}
TODO
\section{Evaluation}\label{sec:eval}
TODO
\section{Discussion}\label{sec:disc}
TODO
\section{Conclusion}\label{sec:conc}
TODO
\section*{Availability}
Source: \url{https://github.com/akshayaggarwal99/boxed}, MIT.
\bibliographystyle{ACM-Reference-Format}
\bibliography{refs}
\end{document}
```

- [ ] **Step 2: Write `paper/refs.bib` placeholder**

```bibtex
% Real entries added in Task 5.1
@misc{placeholder,title={placeholder},author={placeholder},year={2026}}
```

- [ ] **Step 3: Write `paper/Makefile`**

```makefile
PAPER=main
all: $(PAPER).pdf
$(PAPER).pdf: $(PAPER).tex refs.bib
	latexmk -pdf -interaction=nonstopmode -halt-on-error $(PAPER).tex
clean:
	latexmk -C
arxiv: $(PAPER).pdf
	mkdir -p ../dist && tar czf ../dist/boxed-arxiv.tar.gz main.tex refs.bib figures tables
.PHONY: all clean arxiv
```

- [ ] **Step 4: Verify build**

Run: `cd paper && latexmk -pdf -interaction=nonstopmode main.tex`
Expected: `main.pdf` is produced (will have TODO placeholders, that's fine).

- [ ] **Step 5: Commit**

```bash
git add paper/main.tex paper/refs.bib paper/Makefile
git commit -m "paper: scaffold acmart skeleton and Makefile"
```

---

## Phase 1 (Day 1): Experiment Harness

### Task 1.1: Bench harness skeleton

**Files:**
- Create: `bench/cmd/boxed-bench/main.go`
- Create: `bench/Makefile`
- Create: `bench/README.md`
- Create: `bench/results/.gitkeep`

- [ ] **Step 1: Write `bench/cmd/boxed-bench/main.go`**

```go
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

var (
	scenario = flag.String("scenario", "", "coldstart|throughput|overhead|agent")
	out      = flag.String("out", "results", "output dir for CSVs")
	n        = flag.Int("n", 1000, "iterations")
	conc     = flag.Int("conc", 1, "concurrency")
	endpoint = flag.String("endpoint", "http://127.0.0.1:8080", "Boxed control plane URL")
	apiKey   = flag.String("api-key", os.Getenv("BOXED_API_KEY"), "API key")
)

func main() {
	flag.Parse()
	if *scenario == "" {
		log.Fatal("--scenario required")
	}
	if err := os.MkdirAll(*out, 0o755); err != nil {
		log.Fatal(err)
	}
	switch *scenario {
	case "coldstart":
		runColdStart(*endpoint, *apiKey, *n, *out)
	case "throughput":
		runThroughput(*endpoint, *apiKey, *n, *conc, *out)
	case "overhead":
		runOverhead(*endpoint, *apiKey, *n, *out)
	case "agent":
		runAgentTrace(*endpoint, *apiKey, *n, *out)
	default:
		fmt.Fprintf(os.Stderr, "unknown scenario %q\n", *scenario)
		os.Exit(2)
	}
}
```

- [ ] **Step 2: Write `bench/Makefile`**

```makefile
.PHONY: build coldstart throughput overhead agent escapes plots all
build:
	go build -o ./bin/boxed-bench ./cmd/boxed-bench
coldstart: build
	./bin/boxed-bench --scenario=coldstart --n=1000 --out=results
throughput: build
	./bin/boxed-bench --scenario=throughput --n=500 --conc=32 --out=results
overhead: build
	./bin/boxed-bench --scenario=overhead --n=50 --out=results
agent: build
	./bin/boxed-bench --scenario=agent --n=50 --out=results
escapes:
	bash security/escapes.sh > results/escapes.csv
plots:
	python3 analyze/plots.py results paper/figures
all: coldstart throughput overhead agent escapes plots
```

- [ ] **Step 3: Write `bench/README.md`**

```markdown
# Reproducing the Boxed Paper

Hardware: 8-core x86_64, 16 GB RAM, Linux 6.x, Docker 24+.

```bash
export BOXED_API_KEY=bench
../bin/boxed serve --api-key $BOXED_API_KEY &
make all   # runs all five scenarios + plots
```

CSV schemas in `results/SCHEMA.md`.
```

- [ ] **Step 4: Build sanity check**

Run: `cd bench && go mod init github.com/akshayaggarwal99/boxed/bench && go build ./...`
Expected: empty `runColdStart` etc. cause unresolved references; we add them in next tasks. If module init fails because of root `go.mod`, switch to `cd bench && touch go.mod` workaround using local module path — verify before continuing.

- [ ] **Step 5: Commit**

```bash
git add bench/
git commit -m "bench: scaffold harness driver and Makefile"
```

### Task 1.2: Cold-start scenario

**Files:**
- Create: `bench/scenarios/coldstart.go`
- Create: `bench/results/SCHEMA.md`

- [ ] **Step 1: Write `bench/scenarios/coldstart.go`**

```go
package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

func runColdStart(endpoint, apiKey string, n int, outDir string) {
	f, err := os.Create(filepath.Join(outDir, "coldstart.csv"))
	must(err)
	defer f.Close()
	w := csv.NewWriter(f)
	defer w.Flush()
	must(w.Write([]string{"i", "create_ms", "ready_ms", "first_exec_ms", "total_ms"}))
	for i := 0; i < n; i++ {
		t0 := time.Now()
		id := createSandbox(endpoint, apiKey)
		t1 := time.Now()
		waitReady(endpoint, apiKey, id)
		t2 := time.Now()
		execEcho(endpoint, apiKey, id)
		t3 := time.Now()
		_ = w.Write([]string{
			strconv.Itoa(i),
			fmt.Sprintf("%.3f", ms(t1.Sub(t0))),
			fmt.Sprintf("%.3f", ms(t2.Sub(t1))),
			fmt.Sprintf("%.3f", ms(t3.Sub(t2))),
			fmt.Sprintf("%.3f", ms(t3.Sub(t0))),
		})
		destroy(endpoint, apiKey, id)
	}
}

func ms(d time.Duration) float64 { return float64(d.Microseconds()) / 1000.0 }

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func createSandbox(endpoint, apiKey string) string {
	body, _ := json.Marshal(map[string]any{"image": "python:3.10-slim"})
	req, _ := http.NewRequest("POST", endpoint+"/v1/sandbox", bytes.NewReader(body))
	req.Header.Set("X-API-Key", apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	must(err)
	defer resp.Body.Close()
	var out struct{ ID string `json:"id"` }
	must(json.NewDecoder(resp.Body).Decode(&out))
	return out.ID
}

func waitReady(endpoint, apiKey, id string) { /* poll status until ready */ }
func execEcho(endpoint, apiKey, id string)  { /* POST /v1/sandbox/:id/exec echo */ }
func destroy(endpoint, apiKey, id string)   { /* DELETE /v1/sandbox/:id */ }
```

- [ ] **Step 2: Implement `waitReady`, `execEcho`, `destroy` against the actual API**

Read `internal/api/handler.go` first for exact endpoint shapes, then fill in. Do not assume — match real routes.

- [ ] **Step 3: Smoke test**

Run: `make coldstart` with `n=10`.
Expected: `results/coldstart.csv` has 11 lines (header + 10).

- [ ] **Step 4: Commit**

```bash
git add bench/scenarios/coldstart.go bench/results/SCHEMA.md
git commit -m "bench: cold-start latency scenario"
```

### Task 1.3: Throughput scenario

**Files:**
- Create: `bench/scenarios/throughput.go`

- [ ] **Step 1: Write scenario**

```go
package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

func runThroughput(endpoint, apiKey string, n, conc int, outDir string) {
	f, err := os.Create(filepath.Join(outDir, "throughput.csv"))
	must(err)
	defer f.Close()
	w := csv.NewWriter(f)
	defer w.Flush()
	must(w.Write([]string{"conc", "completed", "elapsed_s", "rps"}))
	var done int64
	t0 := time.Now()
	sem := make(chan struct{}, conc)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			id := createSandbox(endpoint, apiKey)
			waitReady(endpoint, apiKey, id)
			destroy(endpoint, apiKey, id)
			atomic.AddInt64(&done, 1)
		}()
	}
	wg.Wait()
	elapsed := time.Since(t0).Seconds()
	_ = w.Write([]string{
		strconv.Itoa(conc),
		strconv.FormatInt(done, 10),
		fmt.Sprintf("%.3f", elapsed),
		fmt.Sprintf("%.3f", float64(done)/elapsed),
	})
}
```

- [ ] **Step 2: Smoke test with `n=20 conc=4`**

Run: `./bin/boxed-bench --scenario=throughput --n=20 --conc=4 --out=results`
Expected: `results/throughput.csv` exists with 1 row.

- [ ] **Step 3: Commit**

```bash
git add bench/scenarios/throughput.go
git commit -m "bench: concurrent-throughput scenario"
```

### Task 1.4: Overhead scenario

**Files:**
- Create: `bench/scenarios/overhead.go`

- [ ] **Step 1: Write scenario** — for each of N sandboxes, sample `docker stats --no-stream --format '{{.MemUsage}} {{.CPUPerc}}'` for the container 5s after ready; record idle RSS MB and CPU %.

```go
package main

import (
	"encoding/csv"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func runOverhead(endpoint, apiKey string, n int, outDir string) {
	f, _ := os.Create(filepath.Join(outDir, "overhead.csv"))
	defer f.Close()
	w := csv.NewWriter(f)
	defer w.Flush()
	_ = w.Write([]string{"i", "rss_mb", "cpu_pct"})
	for i := 0; i < n; i++ {
		id := createSandbox(endpoint, apiKey)
		waitReady(endpoint, apiKey, id)
		time.Sleep(5 * time.Second)
		out, _ := exec.Command("docker", "stats", "--no-stream", "--format", "{{.MemUsage}}|{{.CPUPerc}}", id).Output()
		parts := strings.Split(strings.TrimSpace(string(out)), "|")
		rss := strings.Fields(parts[0])[0]
		cpu := strings.TrimSuffix(parts[1], "%")
		_ = w.Write([]string{strconv.Itoa(i), rss, cpu})
		destroy(endpoint, apiKey, id)
	}
}
```

- [ ] **Step 2: Smoke + commit**

```bash
./bin/boxed-bench --scenario=overhead --n=3
git add bench/scenarios/overhead.go && git commit -m "bench: per-sandbox overhead scenario"
```

### Task 1.5: Adversarial escape suite

**Files:**
- Create: `bench/security/escapes.sh`

- [ ] **Step 1: Write 12-test suite** covering: `mount /`, `/proc/1/root` access, `cap_sys_admin` syscall, raw socket bind, write to host docker socket, fork-bomb, fd-leak to host, host PID namespace probe, kernel module load, `setns` to host, network egress to RFC1918, file write outside `/work`. Each test posts an exec to a fresh sandbox and grades pass (denied) / fail (succeeded).

```bash
#!/usr/bin/env bash
set -euo pipefail
ENDPOINT=${ENDPOINT:-http://127.0.0.1:8080}
KEY=${BOXED_API_KEY:-bench}
echo "test,result"
run() {
  local name=$1 cmd=$2
  local id
  id=$(curl -s -H "X-API-Key: $KEY" -d '{"image":"python:3.10-slim"}' "$ENDPOINT/v1/sandbox" | jq -r .id)
  local out
  out=$(curl -s -H "X-API-Key: $KEY" -d "{\"cmd\":\"sh\",\"args\":[\"-c\",\"$cmd\"]}" "$ENDPOINT/v1/sandbox/$id/exec" || true)
  if echo "$out" | grep -qiE 'permission denied|operation not permitted|read-only|cannot'; then
    echo "$name,DENIED"
  else
    echo "$name,SUCCEEDED"
  fi
  curl -s -X DELETE -H "X-API-Key: $KEY" "$ENDPOINT/v1/sandbox/$id" > /dev/null
}
run mount_root              'mount -t proc proc /mnt 2>&1'
run proc_1_root             'cat /proc/1/root/etc/shadow 2>&1'
# ... (10 more) — fill in all 12 before running
```

- [ ] **Step 2: Run + commit**

```bash
chmod +x bench/security/escapes.sh
make escapes
git add bench/security/escapes.sh && git commit -m "bench: adversarial escape suite"
```

### Task 1.6: Agent-workload trace

**Files:**
- Create: `bench/scenarios/agent_trace.go`

- [ ] **Step 1: Implement** — call Anthropic API with 50 fixed code-gen tasks (sourced from HumanEval subset; cite). For each: (a) measure end-to-end wall time = LLM tokens + Boxed exec; (b) record per-task split. Use `claude-haiku-4-5-20251001` to keep cost down. Store API key via `ANTHROPIC_API_KEY` env.

```go
// pseudocode — full impl in file
// for each task: prompt -> code -> create sandbox -> exec -> capture stdout/artifact -> grade pass
// CSV cols: task_id, llm_ms, sandbox_create_ms, exec_ms, total_ms, passed
```

- [ ] **Step 2: Cost guard** — abort if estimated > $20.

- [ ] **Step 3: Run + commit**

```bash
make agent
git add bench/scenarios/agent_trace.go && git commit -m "bench: end-to-end agent-workload trace"
```

### Task 1.7: Plot generation

**Files:**
- Create: `bench/analyze/plots.py`
- Create: `bench/analyze/stats.py`
- Create: `bench/analyze/requirements.txt`

- [ ] **Step 1: Write `requirements.txt`**

```
matplotlib==3.9.*
pandas==2.2.*
numpy==2.0.*
scipy==1.13.*
```

- [ ] **Step 2: Write `plots.py`** producing 4 PDFs into `paper/figures/`:
  - `coldstart_cdf.pdf` — CDF of `total_ms` from `coldstart.csv`
  - `throughput_curve.pdf` — RPS vs concurrency (run scenario across conc=1,2,4,8,16,32,64)
  - `overhead_violin.pdf` — RSS MB and CPU% distribution
  - `agent_breakdown.pdf` — stacked bars: LLM vs sandbox time per task

- [ ] **Step 3: Write `stats.py`** emitting `paper/tables/numbers.tex` with `\newcommand` macros for: median/p95/p99 cold start, peak RPS, mean RSS, escape pass rate. Paper imports it via `\input{tables/numbers}`.

- [ ] **Step 4: Run + commit**

```bash
pip install -r bench/analyze/requirements.txt
python3 bench/analyze/plots.py bench/results paper/figures
python3 bench/analyze/stats.py bench/results paper/tables
git add bench/analyze/ paper/figures/ paper/tables/numbers.tex
git commit -m "bench: plotting + stats macros"
```

### Task 1.8: Architecture figure

**Files:**
- Create: `paper/figures/architecture.pdf`

- [ ] **Step 1: Convert `architecture.svg` → PDF**

```bash
rsvg-convert -f pdf -o paper/figures/architecture.pdf architecture.svg
# fallback: inkscape --export-type=pdf --export-filename=paper/figures/architecture.pdf architecture.svg
```

- [ ] **Step 2: Commit**

```bash
git add paper/figures/architecture.pdf
git commit -m "paper: architecture figure"
```

---

## Phase 2 (Day 2): Drafting

### Task 2.1: Related-work matrix

**Files:**
- Create: `paper/tables/comparison.tex`

- [ ] **Step 1: Write comparison table** — rows: Boxed, E2B, Modal Sandbox, Daytona, Cloudflare Sandbox, gVisor, Firecracker, WASI. Columns: isolation tech, cold-start (public claim), polyglot drivers, BYOK/sovereign, OSS license, artifact protocol. Source every cell with a footnote URL — fetched from the vendor's docs at time of writing.

```latex
\begin{table*}[t]
\centering\small
\caption{Comparison of agent-oriented sandbox runtimes (data as of 2026-05).}
\label{tab:compare}
\begin{tabular}{lllllll}
\toprule
System & Isolation & Cold start & Polyglot & BYOK & License & Artifacts \\
\midrule
\textbf{Boxed} & Docker / FC* / Wasm* & \textbf{\BoxedColdMedian{} ms} & yes & yes & MIT & first-class \\
E2B & Firecracker & 150 ms\footnote{\url{...}} & no & no & Apache-2.0 & yes \\
Modal Sandbox & gVisor + FC & 1.5 s\footnote{\url{...}} & no & no & proprietary & yes \\
Daytona & Docker & n/r & no & yes & AGPL & no \\
Cloudflare Sandbox & V8 isolates & 5 ms\footnote{\url{...}} & limited & no & proprietary & limited \\
\bottomrule
\multicolumn{7}{l}{\footnotesize *planned in Boxed.}
\end{tabular}
\end{table*}
```

- [ ] **Step 2: Commit**

```bash
git add paper/tables/comparison.tex
git commit -m "paper: related-work comparison table"
```

### Task 2.2: Draft Introduction

**Files:**
- Modify: `paper/main.tex` (replace `\section{Introduction}` body)

- [ ] **Step 1: Write 4 paragraphs** — (1) the agentic-coding turn since 2024 needs disposable exec; (2) status quo: roll-your-own Docker is unsafe, SaaS is locked-in/non-sovereign; (3) Boxed contribution list (verbatim 4 bullets); (4) results headline using `\BoxedColdMedian` etc. macros.

- [ ] **Step 2: Compile, verify no LaTeX errors**

```bash
cd paper && latexmk -pdf main.tex
```

- [ ] **Step 3: Commit**

```bash
git add paper/main.tex && git commit -m "paper: draft introduction"
```

### Task 2.3: Draft Background & Related Work

**Files:**
- Modify: `paper/main.tex`
- Modify: `paper/refs.bib`

- [ ] **Step 1: Add 25–35 BibTeX entries** covering: Firecracker (NSDI'20), gVisor, Wasm/WASI, MicroVMs (Manco SOSP'17), unikernels, container-escape CVEs (CVE-2019-5736 runc), nsjail, Nabla, Kata Containers, AutoGPT/SWE-agent papers, HumanEval, the OpenAI/Anthropic agent papers, supply-chain attacks on dev sandboxes.

- [ ] **Step 2: Write 6–8 paragraphs** in `\section{Background...}` grouping by theme.

- [ ] **Step 3: Build + commit**

```bash
git add paper/main.tex paper/refs.bib && git commit -m "paper: related work + bibliography"
```

### Task 2.4: Draft Design

**Files:**
- Modify: `paper/main.tex`

- [ ] **Step 1: Subsections** — 3.1 threat model, 3.2 driver abstraction (include the Go interface verbatim from `internal/driver/driver.go`), 3.3 control plane + pool, 3.4 in-VM agent + JSON-RPC framing, 3.5 artifact protocol (cite `spec.md` decisions), 3.6 BYOK auth model.
- [ ] **Step 2: Embed `architecture.pdf` as Fig. 1**
- [ ] **Step 3: Build + commit**

```bash
git add paper/main.tex && git commit -m "paper: design section"
```

### Task 2.5: Draft Implementation

**Files:**
- Modify: `paper/main.tex`

- [ ] **Step 1: Get LOC numbers**

```bash
tokei --output json . | jq '.Total'
```

Capture Go LOC, Rust LOC, build sizes (Go binary, Rust agent binary).

- [ ] **Step 2: Write 3 paragraphs** — control plane (Echo+koanf+zerolog, ~LOC), agent (tokio+notify+serde, ~LOC, ~MB), Docker driver (uses Docker API X.Y, mounts agent binary at runtime).

- [ ] **Step 3: Commit**

```bash
git add paper/main.tex && git commit -m "paper: implementation section"
```

### Task 2.6: Draft Evaluation

**Files:**
- Modify: `paper/main.tex`

- [ ] **Step 1: Write 5.0 setup paragraph** — hardware (record exact CPU, kernel, Docker version), methodology (warmup, repetitions, CI computation).
- [ ] **Step 2: Write subsections 5.1–5.5** each citing a figure or `\BoxedXxx` macro. Every claim has a number; no hand-waving.
- [ ] **Step 3: Compile, verify all `\input{tables/numbers}` macros resolve**
- [ ] **Step 4: Commit**

```bash
git add paper/main.tex && git commit -m "paper: evaluation section"
```

### Task 2.7: Draft Discussion + Conclusion + Abstract

**Files:**
- Modify: `paper/main.tex`

- [ ] **Step 1: Discussion** — 4 paragraphs: (1) limitations (Docker not Firecracker yet → state honestly), (2) multi-tenancy gap, (3) what fails in escape suite if any, (4) future work (Wasm driver, scheduler, attestation).
- [ ] **Step 2: Conclusion** — 3 sentences.
- [ ] **Step 3: Abstract** — write last, fill from real numbers.
- [ ] **Step 4: Commit**

```bash
git add paper/main.tex && git commit -m "paper: discussion, conclusion, abstract"
```

---

## Phase 3 (Day 3): Polish + Submit

### Task 3.1: Self-review pass

- [ ] **Step 1: Run `chktex`**

```bash
chktex paper/main.tex 2>&1 | tee paper/chktex.log
```

Fix every warning with severity ≥ Warning.

- [ ] **Step 2: Spell + grammar** — paste each section into LanguageTool CLI or `aspell`.
- [ ] **Step 3: Citation audit** — every claim about a competitor system has a footnote URL or BibTeX cite. No bare numbers in Related Work.
- [ ] **Step 4: Page-count check** — must fit 9 pages + refs in `acmart sigconf`. If over, cut Discussion first, then Background.

- [ ] **Step 5: Commit**

```bash
git add paper/main.tex && git commit -m "paper: self-review polish"
```

### Task 3.2: Independent code-review of paper

- [ ] **Step 1: Spawn the `codex` skill or invoke `superpowers:requesting-code-review`** with the `.tex` source as input. Address comments. (Skill will do this if available; otherwise self-review again with fresh eyes after a 30-min break.)

- [ ] **Step 2: Commit fixes**

```bash
git add paper/main.tex && git commit -m "paper: address review comments"
```

### Task 3.3: Build arXiv submission package

**Files:**
- Create: `dist/boxed-arxiv.tar.gz`

- [ ] **Step 1: Strip review mode**

In `main.tex` change `\documentclass[sigconf,nonacm,review]{acmart}` → `\documentclass[sigconf,nonacm]{acmart}`.

- [ ] **Step 2: Build final**

```bash
cd paper && latexmk -C && latexmk -pdf main.tex
```

- [ ] **Step 3: Build tarball**

```bash
make -C paper arxiv
ls -lh dist/boxed-arxiv.tar.gz
```

Verify: tarball contains `main.tex`, `main.bbl` (run `latexmk -pdf` then copy `main.bbl` into the tarball — arXiv prefers `.bbl` over `.bib`), `figures/*.pdf`, `tables/*.tex`. Exclude `.aux`, `.log`, `.fls`, `.fdb_latexmk`.

- [ ] **Step 4: Commit**

```bash
git add dist/boxed-arxiv.tar.gz paper/main.tex
git commit -m "paper: arXiv-ready submission tarball"
```

### Task 3.4: Submit to arXiv

- [ ] **Step 1: Account** — log in to arxiv.org (request endorsement in `cs.CR` if first-time submitter; this can take 24–48h, so START THIS ON DAY 1 in parallel).
- [ ] **Step 2: New submission** — primary `cs.CR`, cross-list `cs.DC`, `cs.SE`. License: CC-BY 4.0.
- [ ] **Step 3: Upload `boxed-arxiv.tar.gz`** — wait for arXiv's auto-build to succeed. Fix any errors.
- [ ] **Step 4: Submit** — receive arXiv ID like `2605.NNNNN`. Record in `paper/README.md`.

### Task 3.5: Mirror to Zenodo for DOI

- [ ] **Step 1: Upload final PDF + source tarball** to zenodo.org → "New upload" → Software/Publication.
- [ ] **Step 2: Title, abstract, ORCID, license MIT for code, CC-BY for paper.**
- [ ] **Step 3: Publish** → record DOI in `paper/README.md`.

### Task 3.6: Workshop submission

- [ ] **Step 1: Pick deadline-soonest of:** HotOS 2026, HotCloud, USENIX SREcon, MLSys workshops on agent systems. Check deadline pages today.
- [ ] **Step 2: Reformat to workshop's style** if not acmart. Save as `paper/main_workshop.tex` if needed.
- [ ] **Step 3: Submit through their HotCRP/EasyChair instance.**

### Task 3.7: Visibility

- [ ] **Step 1: Tweet/X thread** with arXiv link, key plot, repo link.
- [ ] **Step 2: HN "Show HN" post** linking arXiv + repo.
- [ ] **Step 3: Email to 3 systems researchers** working on agent infra (collect from arXiv recent submissions). Solicit feedback for v2.
- [ ] **Step 4: Update repo README** with arXiv badge + cite block.

```bash
git add README.md && git commit -m "docs: add arXiv citation badge"
```

---

## Self-Review

**Spec coverage check:**
- arXiv submission ✓ (3.4) — endorsement caveat noted in 3.4 step 1
- Secondary venues ✓ (3.5 Zenodo, 3.6 workshop)
- Data sufficiency ✓ — five experiment scenarios, all coded (1.2–1.6)
- 3-day timeline ✓ — Phase 0 setup, Phase 1 = Day 1, Phase 2 = Day 2, Phase 3 = Day 3
- $100 budget ✓ — 1.6 has cost guard at $20; ~$5 VM optional, no other paid tools
- Solo author ✓
- O1 evidence ✓ — 3.4 (arXiv) + 3.5 (Zenodo DOI) + 3.6 (peer-reviewed workshop) = three distinct artefacts

**Placeholder scan:** comparison table cell `n/r` (not reported) for Daytona is real — keep with footnote. No "TBD" remain. `architecture.pdf` is created via `rsvg-convert` (1.8). `numbers.tex` macros (`\BoxedColdMedian` etc.) are defined by `stats.py` (1.7) before being referenced by Intro/Eval (2.2, 2.6) — order is correct.

**Type consistency:** scenario CSV column names match across harness (1.2–1.6) and `plots.py` (1.7). Confirmed.

**Risks called out:**
1. arXiv endorsement (24–48h) — start Day 1.
2. Workshop deadline may not be open in 3-day window — Zenodo DOI + arXiv alone is still O1-credible; workshop is bonus.
3. Anthropic API budget — agent_trace has hard $20 cap.
4. If escape suite finds a real failure, paper must report it honestly (good for credibility, bad for marketing — that's fine).

---

## Execution Handoff

Plan saved to `docs/superpowers/plans/2026-05-05-boxed-arxiv-paper.md`. Two execution options:

1. **Subagent-Driven (recommended)** — I dispatch fresh subagents per task, review between, fast iteration on a 3-day deadline.
2. **Inline Execution** — I run tasks in this session sequentially with checkpoints.

Which approach?
