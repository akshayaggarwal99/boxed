#!/usr/bin/env bash
# Lifecycle of the OpenHands agent-server sandbox, driven the way the OpenHands
# SDK's DockerWorkspace drives it (docker run ... IMAGE --host 0.0.0.0 --port
# 8000; wait for GET /health; POST /api/bash/execute_bash_command), with the
# same create -> first exec -> destroy phases the Boxed cold-start scenario
# records. Usage: openhands_lifecycle.sh <n> <out_dir>
set -uo pipefail
N="${1:-100}"; OUT="${2:-results/openhands}"; mkdir -p "$OUT"
IMAGE="${OPENHANDS_IMAGE:-ghcr.io/openhands/agent-server:latest-python}"
PORT="${OPENHANDS_PORT:-8010}"
CSV="$OUT/coldstart.csv"; echo "i,create_ms,first_exec_ms,destroy_ms,total_ms,exit_code" > "$CSV"
now() { date +%s%N; }
for i in $(seq 0 $((N-1))); do
  t0=$(now)
  id=$(docker run -d --ulimit nofile=65536:65536 --name "oh-bench-$i" -p $PORT:8000 "$IMAGE" --host 0.0.0.0 --port 8000 2>/dev/null)
  until curl -s --max-time 1 -o /dev/null -w '%{http_code}' "http://127.0.0.1:$PORT/health" 2>/dev/null | grep -q '^2'; do
    sleep 0.05; [[ $(( ($(now)-t0)/1000000000 )) -gt 180 ]] && break
  done
  t1=$(now)
  code=$(curl -s --max-time 60 -X POST "http://127.0.0.1:$PORT/api/bash/execute_bash_command" -H 'Content-Type: application/json' \
    -d '{"command":"python3 -c \"print(\\\"ok\\\")\"","timeout":30}' | python3 -c 'import sys,json
d=json.load(sys.stdin); items=(d.get("items") if isinstance(d,dict) and "items" in d else ([d] if isinstance(d,dict) else d))
ec=[x.get("exit_code") for x in items if x.get("exit_code") is not None]
print(ec[-1] if ec else -1)' 2>/dev/null || echo -1)
  t2=$(now)
  docker rm -f "oh-bench-$i" >/dev/null 2>&1
  t3=$(now)
  printf '%d,%.3f,%.3f,%.3f,%.3f,%s\n' "$i" "$(( t1-t0 ))e-6" "$(( t2-t1 ))e-6" "$(( t3-t2 ))e-6" "$(( t3-t0 ))e-6" "$code" | awk -F, '{printf "%s,%.3f,%.3f,%.3f,%.3f,%s\n",$1,$2,$3,$4,$5,$6}' >> "$CSV"
done
echo "wrote $CSV"
