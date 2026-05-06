#!/usr/bin/env bash
# Adversarial escape suite for Boxed.
# Each test creates a fresh sandbox, runs an attempt, classifies the outcome.
# DENIED = sandbox blocked the action (good).
# SUCCEEDED = sandbox failed to block (bad — must be reported in paper).
# ERROR = test infrastructure failure (does not count either way).
set -uo pipefail
ENDPOINT="${ENDPOINT:-http://127.0.0.1:8080}"
KEY="${BOXED_API_KEY:-bench}"

echo "test,result"

run_test() {
  local name="$1" lang="$2" code="$3" denied_re="$4"
  local id status
  id=$(curl -s -H "X-Boxed-API-Key: $KEY" -H 'Content-Type: application/json' \
        -d '{"timeout":60}' "$ENDPOINT/v1/sandbox" | python3 -c \
        'import sys,json;print(json.load(sys.stdin).get("sandbox_id",""))') || true
  if [[ -z "$id" ]]; then
    echo "$name,ERROR"
    return
  fi
  local body
  body=$(python3 -c 'import json,sys;print(json.dumps({"language":sys.argv[1],"code":sys.argv[2]}))' "$lang" "$code")
  out=$(curl -s -H "X-Boxed-API-Key: $KEY" -H 'Content-Type: application/json' \
        -d "$body" "$ENDPOINT/v1/sandbox/$id/exec" || true)
  curl -s -X DELETE -H "X-Boxed-API-Key: $KEY" "$ENDPOINT/v1/sandbox/$id" >/dev/null

  if echo "$out" | grep -qiE "$denied_re"; then
    status=DENIED
  else
    status=SUCCEEDED
  fi
  echo "$name,$status"
}

# 1. Mount the host root
run_test mount_host bash 'mount -t proc proc /mnt 2>&1; echo done' \
  'permission denied|operation not permitted|must be superuser|read-only|denied'

# 2. Read host-process root via /proc/1/root
run_test proc1_root bash 'ls /proc/1/root/etc 2>&1' \
  'permission denied|no such file|denied|operation not permitted'

# 3. Touch the docker socket
run_test docker_sock bash 'ls -l /var/run/docker.sock 2>&1; cat /var/run/docker.sock 2>&1 || true' \
  'no such file|permission denied|cannot open|connection refused'

# 4. Load a kernel module
run_test modprobe bash 'modprobe dummy 2>&1' \
  'not found|permission denied|operation not permitted|cannot|denied'

# 5. Raw socket bind (requires CAP_NET_RAW)
run_test raw_socket python 'import socket; socket.socket(socket.AF_INET, socket.SOCK_RAW, socket.IPPROTO_ICMP)' \
  'PermissionError|Operation not permitted|errno 1'

# 6. setns/unshare to host namespace
run_test setns bash 'unshare --user --pid bash -c "echo in" 2>&1' \
  'permission denied|operation not permitted|denied|write error'

# 7. Egress to RFC1918 (sandbox should block private network)
run_test rfc1918 python 'import socket,sys; s=socket.socket(); s.settimeout(2);
try:
  s.connect(("10.0.0.1",80)); print("CONNECTED")
except Exception as e:
  print("BLOCKED",e)' \
  'BLOCKED|timed out|refused|unreachable'

# 8. Egress to link-local metadata 169.254.169.254
run_test imds python 'import socket,sys; s=socket.socket(); s.settimeout(2);
try:
  s.connect(("169.254.169.254",80)); print("CONNECTED")
except Exception as e:
  print("BLOCKED",e)' \
  'BLOCKED|timed out|refused|unreachable'

# 9. Fork bomb — sandbox should kill or limit
run_test fork_bomb bash ':(){ :|:& };: 2>&1 & sleep 2; echo survived' \
  'killed|cannot fork|resource temporarily unavailable|terminated'

# 10. Memory bomb (allocate 4 GB)
run_test mem_bomb python 'a=bytearray(4*1024*1024*1024)' \
  'MemoryError|killed|cannot allocate'

# 11. Write outside writable root (try /etc, /usr)
run_test ro_root bash 'echo x > /etc/escape 2>&1; cat /etc/escape 2>&1' \
  'read-only|permission denied|denied|cannot'

# 12. ptrace another process (host PID 1 if visible)
run_test ptrace_host python 'import ctypes,os; r=ctypes.CDLL("libc.so.6").ptrace(16,1,0,0); print(r,os.strerror(ctypes.get_errno()))' \
  'Operation not permitted|No such process|permission denied|errno'
