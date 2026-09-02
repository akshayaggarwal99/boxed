#!/usr/bin/env bash
# Full measurement campaign for the paper. Writes results/<RUN>/...
# Interleaves Boxed and raw-Docker baseline runs so that host drift over the
# campaign affects both equally.
set -uo pipefail
RUN="${RUN:-hardened-2026-09}"
CAMPAIGN_NOTE="${CAMPAIGN_NOTE:-}"
REPEATS="${REPEATS:-5}"
TP_REPEATS="${TP_REPEATS:-10}"
N="${N:-200}"        # lifecycles per cold-start / baseline run
TPN="${TPN:-80}"     # lifecycles per throughput level
OVN="${OVN:-30}"     # idle sandboxes sampled
ENDPOINT="${ENDPOINT:-http://127.0.0.1:8080}"
export BOXED_API_KEY="${BOXED_API_KEY:-bench}"
OUT="results/$RUN"; mkdir -p "$OUT"

{
  echo "date: $(date -u +%FT%TZ)"
  echo "git: $(git -C .. rev-parse HEAD) dirty=$(git -C .. status --porcelain | wc -l | tr -d ' ')"
  if command -v sysctl >/dev/null && sysctl -n machdep.cpu.brand_string >/dev/null 2>&1; then
    echo "host: $(sysctl -n machdep.cpu.brand_string) $(( $(sysctl -n hw.memsize) / 1073741824 )) GiB macOS $(sw_vers -productVersion)"
    echo "vm: $(colima version 2>/dev/null | head -1) cpu=$(colima list -j 2>/dev/null | python3 -c 'import sys,json;d=json.load(sys.stdin);print(d.get("cpu"),d.get("memory"),d.get("runtime"))' 2>/dev/null)"
  else
    echo "host: $(lscpu | grep 'Model name' | sed 's/.*: *//') $(nproc) cores $(( $(grep MemTotal /proc/meminfo | awk '{print $2}') / 1048576 )) GiB $(. /etc/os-release; echo $PRETTY_NAME) kernel $(uname -r)"
    echo "vm: none (native Linux host: $(curl -s -H 'Metadata-Flavor: Google' http://metadata.google.internal/computeMetadata/v1/instance/machine-type 2>/dev/null | sed 's|.*/||' || echo unknown))"
  fi
  echo "docker: $(docker version --format '{{.Server.Version}} {{.Server.Os}}/{{.Server.Arch}} kernel {{.Server.KernelVersion}}')"
  echo "image: $(docker image inspect python:3.10-slim --format '{{.Id}} {{.Created}}')"
  echo "load: $(uptime)"
  echo "runtime: ${BOXED_DOCKER_RUNTIME:-default(runc)} $(docker info --format '{{.DefaultRuntime}} available={{range $k,$v := .Runtimes}}{{$k}} {{end}}' 2>/dev/null)"
  [ -n "$CAMPAIGN_NOTE" ] && echo "note: $CAMPAIGN_NOTE"
} > "$OUT/ENV.txt"

for r in $(seq 1 "$REPEATS"); do
  d="$OUT/r$r"; mkdir -p "$d"
  echo "== repeat $r coldstart (boxed)"; ./bin/boxed-bench --scenario=coldstart --n=$N --endpoint="$ENDPOINT" --out="$d"
  echo "== repeat $r baseline hardened"; ./bin/boxed-bench --scenario=baseline --mode=hardened --n=$N --out="$d"
  echo "== repeat $r baseline default";  ./bin/boxed-bench --scenario=baseline --mode=default  --n=$N --out="$d"
done

for r in $(seq 1 "$TP_REPEATS"); do
  d="$OUT/tp$r"; mkdir -p "$d"
  echo "== throughput sweep $r"
  for c in 1 2 4 8 16 32; do
    ./bin/boxed-bench --scenario=throughput --n=$TPN --conc=$c --endpoint="$ENDPOINT" --out="$d"
  done
  sleep 5
done

echo "== overhead"; ./bin/boxed-bench --scenario=overhead --n=$OVN --endpoint="$ENDPOINT" --out="$OUT"

for r in 1 2 3; do
  echo "== escapes $r"; ENDPOINT="$ENDPOINT" bash security/escapes.sh > "$OUT/escapes_r$r.csv"
done
echo "== done $(date -u +%FT%TZ)" >> "$OUT/ENV.txt"
