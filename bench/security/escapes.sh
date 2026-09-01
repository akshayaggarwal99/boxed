#!/usr/bin/env bash
# Boxed escape probe, post-condition edition.
#
# Every vector runs in a fresh sandbox. Two verdicts are recorded per vector:
#   signature     - the v1 classifier: grep captured output for a denial string.
#                   Kept so the two methods can be compared on the same run.
#   postcondition - a check of state after the attempt: a file that did or did
#                   not appear, a mount that did or did not exist, a cgroup
#                   counter read from the host, an errno from the syscall
#                   itself. This is the verdict the paper reports.
#
# postcondition verdicts: DENIED (boundary held, with evidence), UNDENIED
# (boundary crossed or limit not enforced, with evidence), ERROR (harness).
#
# Host-side cgroup reads go through `colima ssh` when HOST_SSH is set; on a
# native Linux host set HOST_SSH="" and the reads run locally.
set -uo pipefail
ENDPOINT="${ENDPOINT:-http://127.0.0.1:8080}"
KEY="${BOXED_API_KEY:-bench}"
HOST_SSH="${HOST_SSH:-colima ssh --}"
CG="${CGROUP_ROOT:-/sys/fs/cgroup/docker}"

echo "test,threat,attempt_exit,signature,postcondition,evidence"

api() { curl -s -H "X-Boxed-API-Key: $KEY" -H 'Content-Type: application/json' "$@"; }
mk()  { api -d '{"timeout":120}' "$ENDPOINT/v1/sandbox" | python3 -c 'import sys,json;print(json.load(sys.stdin).get("sandbox_id",""))'; }
rm_() { api -X DELETE "$ENDPOINT/v1/sandbox/$1" >/dev/null; }
run() { # id lang code -> JSON exec response
  local body; body=$(python3 -c 'import json,sys;print(json.dumps({"language":sys.argv[1],"code":sys.argv[2]}))' "$2" "$3")
  api -d "$body" "$ENDPOINT/v1/sandbox/$1/exec"
}
field() { python3 -c 'import sys,json;d=json.load(sys.stdin);v=d.get(sys.argv[1]);print("" if v is None else v)' "$1"; }
hostcat() { $HOST_SSH sudo cat "$1" 2>/dev/null | tr -d '\r'; }
csvq() { printf '%s' "$1" | tr ',' ';' | tr '\n' ' ' | sed 's/"//g'; }

emit() { # name threat exit sig post evidence
  echo "$1,$2,$3,$4,$5,\"$(csvq "$6")\""
}

# ---------------------------------------------------------------- vectors
# Each attempt prints a last line "POST:<verdict>|<evidence>" computed inside
# the sandbox where the post-condition is observable there, or the host side
# computes it afterwards.

v_mount_host() {
  local id out ec so sig post ev
  id=$(mk); [[ -z "$id" ]] && { emit mount_host T1;T2 - ERROR ERROR "no sandbox"; return; }
  out=$(run "$id" bash 'mount -t proc proc /mnt 2>&1; mount --bind / /mnt 2>&1; echo "rc=$?"; n=$(grep -c " /mnt " /proc/mounts); echo "POST:$([[ $n -eq 0 ]] && echo DENIED || echo UNDENIED)|mounts_on_/mnt=$n"')
  ec=$(printf '%s' "$out" | field exit_code); so=$(printf '%s' "$out" | field stdout)
  sig=$(printf '%s' "$so" | grep -qiE 'permission denied|operation not permitted|must be superuser|denied' && echo DENIED || echo UNDENIED)
  post=$(printf '%s' "$so" | sed -n 's/^POST:\([A-Z]*\)|.*/\1/p' | tail -1); ev=$(printf '%s' "$so" | sed -n 's/^POST:[A-Z]*|//p' | tail -1)
  emit mount_host "T1;T2" "$ec" "$sig" "${post:-ERROR}" "$ev"; rm_ "$id"
}

v_proc1_root() {
  local id out ec so sig post ev
  id=$(mk); [[ -z "$id" ]] && { emit proc1_root T1 - ERROR ERROR "no sandbox"; return; }
  out=$(run "$id" bash 'ls /proc/1/root/etc >/dev/null 2>&1; echo "rc=$?"; a=$(stat -c %d:%i /proc/1/root/.); b=$(stat -c %d:%i /.); c=$(cat /proc/1/comm); echo "POST:$([[ "$a" == "$b" ]] && echo DENIED || echo UNDENIED)|proc1_root_dev_ino=$a root_dev_ino=$b pid1=$c"')
  ec=$(printf '%s' "$out" | field exit_code); so=$(printf '%s' "$out" | field stdout)
  sig=$(printf '%s' "$so" | grep -qiE 'permission denied|no such file|denied' && echo DENIED || echo UNDENIED)
  post=$(printf '%s' "$so" | sed -n 's/^POST:\([A-Z]*\)|.*/\1/p' | tail -1); ev=$(printf '%s' "$so" | sed -n 's/^POST:[A-Z]*|//p' | tail -1)
  emit proc1_root T1 "$ec" "$sig" "${post:-ERROR}" "$ev"; rm_ "$id"
}

v_docker_sock() {
  local id out ec so sig post ev
  id=$(mk); [[ -z "$id" ]] && { emit docker_sock "T2;T3" - ERROR ERROR "no sandbox"; return; }
  out=$(run "$id" python 'import os,socket,glob
cands=[p for p in ["/var/run/docker.sock","/run/docker.sock","/var/run/containerd/containerd.sock","/run/containerd/containerd.sock"] if os.path.exists(p)]
got=False
for p in cands:
    try:
        s=socket.socket(socket.AF_UNIX); s.settimeout(2); s.connect(p)
        s.sendall(b"GET /containers/json HTTP/1.0\r\n\r\n"); r=s.recv(200); got=b"HTTP/1" in r
    except Exception as e: pass
print("POST:%s|sockets_found=%d api_reachable=%s"%("UNDENIED" if got else "DENIED",len(cands),got))')
  ec=$(printf '%s' "$out" | field exit_code); so=$(printf '%s' "$out" | field stdout)
  sig=$(printf '%s' "$so" | grep -qiE 'no such file|permission denied|connection refused' && echo DENIED || echo UNDENIED)
  post=$(printf '%s' "$so" | sed -n 's/^POST:\([A-Z]*\)|.*/\1/p' | tail -1); ev=$(printf '%s' "$so" | sed -n 's/^POST:[A-Z]*|//p' | tail -1)
  emit docker_sock "T2;T3" "$ec" "$sig" "${post:-ERROR}" "$ev"; rm_ "$id"
}

v_modprobe() {
  # No modprobe binary in the image (the v1 'not found' verdict was an artifact).
  # Call init_module(2) directly; EPERM means CAP_SYS_MODULE is absent.
  local id out ec so sig post ev
  id=$(mk); [[ -z "$id" ]] && { emit modprobe T2 - ERROR ERROR "no sandbox"; return; }
  out=$(run "$id" python 'import ctypes,os,platform
libc=ctypes.CDLL(None,use_errno=True)
NR={"aarch64":105,"x86_64":175}[platform.machine()]  # init_module
buf=ctypes.create_string_buffer(b"\x7fELF"+b"\0"*60)
r=libc.syscall(NR,buf,ctypes.c_ulong(64),b"")
e=ctypes.get_errno()
print("rc=%d errno=%d %s"%(r,e,os.strerror(e)))
print("POST:%s|init_module errno=%d(%s)"%("DENIED" if e==1 else "UNDENIED",e,os.strerror(e)))')
  ec=$(printf '%s' "$out" | field exit_code); so=$(printf '%s' "$out" | field stdout)
  sig=$(printf '%s' "$so" | grep -qiE 'not found|permission denied|operation not permitted|denied' && echo DENIED || echo UNDENIED)
  post=$(printf '%s' "$so" | sed -n 's/^POST:\([A-Z]*\)|.*/\1/p' | tail -1); ev=$(printf '%s' "$so" | sed -n 's/^POST:[A-Z]*|//p' | tail -1)
  emit modprobe T2 "$ec" "$sig" "${post:-ERROR}" "$ev"; rm_ "$id"
}

v_raw_socket() {
  local id out ec so sig post ev
  id=$(mk); [[ -z "$id" ]] && { emit raw_socket T2 - ERROR ERROR "no sandbox"; return; }
  out=$(run "$id" python 'import socket,errno
try:
    s=socket.socket(socket.AF_INET,socket.SOCK_RAW,socket.IPPROTO_ICMP); s.close()
    print("POST:UNDENIED|raw socket created")
except PermissionError as e:
    print("POST:DENIED|errno=%d(%s)"%(e.errno,e.strerror))')
  ec=$(printf '%s' "$out" | field exit_code); so=$(printf '%s' "$out" | field stdout)
  sig=$(printf '%s' "$so" | grep -qiE 'PermissionError|Operation not permitted' && echo DENIED || echo UNDENIED)
  post=$(printf '%s' "$so" | sed -n 's/^POST:\([A-Z]*\)|.*/\1/p' | tail -1); ev=$(printf '%s' "$so" | sed -n 's/^POST:[A-Z]*|//p' | tail -1)
  emit raw_socket T2 "$ec" "$sig" "${post:-ERROR}" "$ev"; rm_ "$id"
}

v_setns() {
  local id out ec so sig post ev
  id=$(mk); [[ -z "$id" ]] && { emit setns T2 - ERROR ERROR "no sandbox"; return; }
  out=$(run "$id" bash 'before=$(readlink /proc/self/ns/pid); unshare --user --pid --fork bash -c "readlink /proc/self/ns/pid" >/tmp/ns 2>/tmp/err; rc=$?; after=$(cat /tmp/ns 2>/dev/null); echo "rc=$rc err=$(cat /tmp/err)"; if [[ $rc -ne 0 || -z "$after" ]]; then echo "POST:DENIED|unshare rc=$rc $(head -c 80 /tmp/err)"; else echo "POST:UNDENIED|new pid ns $after (was $before)"; fi')
  ec=$(printf '%s' "$out" | field exit_code); so=$(printf '%s' "$out" | field stdout)
  sig=$(printf '%s' "$so" | grep -qiE 'permission denied|operation not permitted|denied' && echo DENIED || echo UNDENIED)
  post=$(printf '%s' "$so" | sed -n 's/^POST:\([A-Z]*\)|.*/\1/p' | tail -1); ev=$(printf '%s' "$so" | sed -n 's/^POST:[A-Z]*|//p' | tail -1)
  emit setns T2 "$ec" "$sig" "${post:-ERROR}" "$ev"; rm_ "$id"
}

v_egress() { # name target
  local id out ec so sig post ev
  id=$(mk); [[ -z "$id" ]] && { emit "$1" T3 - ERROR ERROR "no sandbox"; return; }
  out=$(run "$id" python "import socket,os
ifaces=sorted(os.listdir('/sys/class/net'))
try:
    s=socket.socket(); s.settimeout(3); s.connect(('$2',80)); print('CONNECTED'); ok=True
except Exception as e:
    print('BLOCKED',e); ok=False
print('POST:%s|ifaces=%s'%('UNDENIED' if ok else 'DENIED',','.join(ifaces)))")
  ec=$(printf '%s' "$out" | field exit_code); so=$(printf '%s' "$out" | field stdout)
  sig=$(printf '%s' "$so" | grep -qiE 'BLOCKED|timed out|refused|unreachable' && echo DENIED || echo UNDENIED)
  post=$(printf '%s' "$so" | sed -n 's/^POST:\([A-Z]*\)|.*/\1/p' | tail -1); ev=$(printf '%s' "$so" | sed -n 's/^POST:[A-Z]*|//p' | tail -1)
  emit "$1" T3 "$ec" "$sig" "${post:-ERROR}" "$ev"; rm_ "$id"
}

v_fork_bomb() {
  # Controlled fork storm: attempt 2000 forks of sleeping children, count how
  # many the kernel granted, then read the cgroup's pids.max / pids.events on
  # the host. Children are reaped by the parent so the sandbox stays usable.
  local id out ec so sig post ev pmax pev
  id=$(mk); [[ -z "$id" ]] && { emit fork_bomb T4 - ERROR ERROR "no sandbox"; return; }
  out=$(run "$id" python 'import os,time,signal
kids=[];err=None
for i in range(2000):
    try:
        p=os.fork()
        if p==0:
            time.sleep(30); os._exit(0)
        kids.append(p)
    except BlockingIOError as e:
        err="EAGAIN after %d forks"%len(kids); break
for p in kids:
    try: os.kill(p,signal.SIGKILL)
    except Exception: pass
for p in kids:
    try: os.waitpid(p,0)
    except Exception: pass
print("forked=%d %s"%(len(kids),err or "no EAGAIN"))
print("POST:%s|forked=%d %s"%("DENIED" if err else "UNDENIED",len(kids),err or "no EAGAIN in 2000 forks"))')
  ec=$(printf '%s' "$out" | field exit_code); so=$(printf '%s' "$out" | field stdout)
  pmax=$(hostcat "$CG/$id/pids.max"); pev=$(hostcat "$CG/$id/pids.events" | tr '\n' ' ')
  sig=$(printf '%s' "$so" | grep -qiE 'killed|cannot fork|resource temporarily unavailable|EAGAIN' && echo DENIED || echo UNDENIED)
  post=$(printf '%s' "$so" | sed -n 's/^POST:\([A-Z]*\)|.*/\1/p' | tail -1); ev="$(printf '%s' "$so" | sed -n 's/^POST:[A-Z]*|//p' | tail -1) host:pids.max=$pmax pids.events=$pev"
  emit fork_bomb T4 "$ec" "$sig" "${post:-ERROR}" "$ev"; rm_ "$id"
}

v_mem_bomb() {
  # Touch pages, not just reserve them: allocate 64 MiB chunks and write to
  # each until the cgroup kills the process. Post-condition is the host-side
  # memory.events oom_kill counter plus the exit status.
  local id out ec so sig post ev mmax mev
  id=$(mk); [[ -z "$id" ]] && { emit mem_bomb T4 - ERROR ERROR "no sandbox"; return; }
  out=$(run "$id" python 'import sys
chunks=[]
try:
    for i in range(64):
        b=bytearray(64*1024*1024)
        for j in range(0,len(b),4096): b[j]=1
        chunks.append(b)
    print("POST:UNDENIED|allocated 4 GiB without kill")
except MemoryError:
    print("POST:DENIED|MemoryError after %d MiB"%(64*len(chunks)))')
  ec=$(printf '%s' "$out" | field exit_code); so=$(printf '%s' "$out" | field stdout)
  mmax=$(hostcat "$CG/$id/memory.max"); mev=$(hostcat "$CG/$id/memory.events" | grep -E 'oom_kill' | tr '\n' ' ')
  sig=$(printf '%s' "$so" | grep -qiE 'MemoryError|killed|cannot allocate' && echo DENIED || echo UNDENIED)
  post=$(printf '%s' "$so" | sed -n 's/^POST:\([A-Z]*\)|.*/\1/p' | tail -1)
  if [[ -z "$post" ]]; then
    # process was SIGKILLed before it could print: the cgroup bound.
    if [[ "$mev" == *"oom_kill 0"* || -z "$mev" ]]; then post=UNDENIED; else post=DENIED; fi
  fi
  ev="exit=$ec host:memory.max=$mmax $mev"
  emit mem_bomb T4 "$ec" "$sig" "$post" "$ev"; rm_ "$id"
}

v_ro_root() {
  local id out ec so sig post ev
  id=$(mk); [[ -z "$id" ]] && { emit ro_root T1 - ERROR ERROR "no sandbox"; return; }
  out=$(run "$id" bash 'echo x > /etc/boxed_escape 2>/tmp/e1; r1=$?; echo x > /usr/bin/boxed_escape 2>/tmp/e2; r2=$?; a=$([[ -e /etc/boxed_escape ]] && echo present || echo absent); b=$([[ -e /usr/bin/boxed_escape ]] && echo present || echo absent); echo "rc=$r1,$r2 $(cat /tmp/e1 | head -c 60)"; echo "POST:$([[ $a == absent && $b == absent ]] && echo DENIED || echo UNDENIED)|/etc:$a /usr/bin:$b $(cat /tmp/e1 | sed "s/.*: //" | head -c 40)"')
  ec=$(printf '%s' "$out" | field exit_code); so=$(printf '%s' "$out" | field stdout)
  sig=$(printf '%s' "$so" | grep -qiE 'read-only|permission denied|denied' && echo DENIED || echo UNDENIED)
  post=$(printf '%s' "$so" | sed -n 's/^POST:\([A-Z]*\)|.*/\1/p' | tail -1); ev=$(printf '%s' "$so" | sed -n 's/^POST:[A-Z]*|//p' | tail -1)
  emit ro_root T1 "$ec" "$sig" "${post:-ERROR}" "$ev"; rm_ "$id"
}

v_ptrace_agent() {
  # The v1 probe targeted PID 1 (the tail placeholder). The process that
  # matters is boxed-agent, a peer in the same PID namespace: attaching to it
  # would let the workload tamper with the RPC channel. Attempt PTRACE_ATTACH
  # on the agent and report the errno.
  local id out ec so sig post ev
  id=$(mk); [[ -z "$id" ]] && { emit ptrace_agent "T2;T3" - ERROR ERROR "no sandbox"; return; }
  out=$(run "$id" python 'import ctypes,os,glob
libc=ctypes.CDLL(None,use_errno=True)
target=None
for d in glob.glob("/proc/[0-9]*"):
    try:
        if open(d+"/comm").read().strip()=="boxed-agent": target=int(d.split("/")[-1]); break
    except Exception: pass
scope=open("/proc/sys/kernel/yama/ptrace_scope").read().strip()
if target is None:
    print("POST:ERROR|boxed-agent not visible"); raise SystemExit(0)
r=libc.ptrace(16,target,0,0); e=ctypes.get_errno()
if r==0:
    libc.ptrace(17,target,0,0)
    print("POST:UNDENIED|PTRACE_ATTACH pid=%d succeeded yama=%s"%(target,scope))
else:
    print("POST:DENIED|PTRACE_ATTACH pid=%d errno=%d(%s) yama=%s"%(target,e,os.strerror(e),scope))')
  ec=$(printf '%s' "$out" | field exit_code); so=$(printf '%s' "$out" | field stdout)
  sig=$(printf '%s' "$so" | grep -qiE 'Operation not permitted|No such process|permission denied|errno' && echo DENIED || echo UNDENIED)
  post=$(printf '%s' "$so" | sed -n 's/^POST:\([A-Z]*\)|.*/\1/p' | tail -1); ev=$(printf '%s' "$so" | sed -n 's/^POST:[A-Z]*|//p' | tail -1)
  emit ptrace_agent "T2;T3" "$ec" "$sig" "${post:-ERROR}" "$ev"; rm_ "$id"
}

v_mount_host
v_proc1_root
v_docker_sock
v_modprobe
v_raw_socket
v_setns
v_egress rfc1918 10.0.0.1
v_egress imds 169.254.169.254
v_fork_bomb
v_mem_bomb
v_ro_root
v_ptrace_agent
