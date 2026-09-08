# Draft — VentureBeat guest post (target 800–1200 words)

**Status: skeleton for the author to rewrite.** VentureBeat bans AI-generated
content. Every number below is verified against the raw traces at
https://github.com/akshayaggarwal99/boxed/tree/v0.3.2-paper/bench/results ; the sentences
need to be retyped in your own voice before submission.

**Headline:** I hardened my AI agent sandbox and it got faster. Then a microVM
showed me what I had actually secured.

**Byline:** Akshay Kumar, Independent Researcher — akumar8@mt.iitr.ac.in

**One-sentence bio:** Akshay Kumar is an independent researcher who builds and
measures code-execution sandboxes for autonomous coding agents.

---

Every AI coding agent has the same shape underneath. It writes a snippet, runs
it, reads the output, and tries again. Dozens of times per task. That means the
sandbox sits inside the loop, and you pay for it on every iteration.

Which is why almost everyone reaches for plain Docker and then quietly skips the
security flags. Read-only root filesystem, drop all capabilities,
`no-new-privileges`, a PID cap, a memory limit, no network. The folk wisdom is
that each one costs latency you cannot afford in a hot loop.

I measured it. The folk wisdom is wrong, and finding that out led to a worse
surprise about what my hardening was actually doing.

## Hardening was faster

I ran the same create, exec, destroy cycle three ways against the Docker Engine
API: stock `docker run` defaults, the same thing with the full hardened host
config, and my own substrate on top. Same host, same image, same command. Five
runs of 200 sequential lifecycles each, interleaved so host drift hit all three
equally.

On a laptop, stock defaults came in at a median of 216 ms. The hardened
configuration: 153 ms. Hardening was **63 ms faster**. I assumed I had a bug, so
I repeated the whole campaign on an idle GCE `n2-standard-4`. Same direction:
368 ms stock, 298 ms hardened, 70 ms faster.

The reason is dull once you see it. `--network none` means Docker never creates
a veth pair, never attaches the container to the bridge, and never tears any of
it down. That setup and teardown costs more than the read-only root, the
capability drop and the tmpfs mounts save. The most security-relevant flag in
the list is also the one that removes the most work.

If you skipped hardening to keep your agent loop fast, you made your sandbox
weaker and slower at the same time.

## Then I changed the boundary

Docker lets you swap the OCI runtime per container. So I ran the identical
control plane, agent, image and harness under three of them: `runc`, gVisor's
`runsc` user-space kernel, and Kata Containers, which boots a real QEMU microVM
per sandbox.

Lifecycle medians: 354 ms under runc, 405 ms under gVisor, 7,824 ms under Kata.
No surprise there. You pay for a stronger boundary in boot time.

The surprise was the escape probe. I run twelve attempts drawn from common
container-escape patterns: a filesystem mount, reading the host through
`/proc/1/root`, the Docker socket, `init_module`, a raw socket, a new user and
PID namespace, egress to RFC1918 and to the cloud metadata endpoint, a fork
storm, a memory bomb, a root filesystem write, and `PTRACE_ATTACH` on my
in-sandbox agent.

Under runc: 12 of 12 denied. Under gVisor: 12 of 12. Under Kata, the strictly
stronger boundary: **11 of 12**.

The `ptrace` attach worked.

## What was actually protecting me

runc had never been blocking that attack. The host kernel had.

Linux ships a Yama LSM, and with `ptrace_scope=1` it limits attachment to
descendant processes. Under runc the container shares the host kernel, so Yama
was quietly doing the work. Kata's guest kernel has no Yama at all. Inside that
microVM my workload and my agent both run as root, and a same-uid `ptrace` needs
no capability. So the attach succeeded.

I had `CapDrop ALL` in my config. I had looked at that line many times and read
it as covering this. It never did. A host kernel policy I had not written, did
not configure and could not see from inside was carrying a control I believed
was mine.

That is the part worth generalizing. When you harden a container you are
configuring two things at once: your own flags, and whatever the host kernel
happens to enforce. The second set is invisible in your compose file, varies
between hosts, and disappears the moment you move to a stronger boundary that
brings its own kernel. Upgrading isolation can silently remove a protection.

The fix was one line. The agent now calls `prctl(PR_SET_DUMPABLE, 0)` at
startup, after which the kernel's own access check refuses any tracer lacking
`CAP_SYS_PTRACE`, which the driver already drops. Re-running the probe: 12 of 12
under all three runtimes, three runs each. It sits off the latency path and cost
nothing.

## Why your escape tests probably lie to you

None of this would have surfaced if I scored the probe the usual way.

Most escape harnesses capture stdout and stderr and grep for a denial string.
Run that against my twelve vectors and it reports **2 of them wrong**, both
times claiming an attack succeeded when the host evidence proves it was stopped.

The memory bomb is the clearest case. The cgroup kills it. But `SIGKILL` gives
the process no chance to print anything, so the harness sees empty output, finds
no denial string, and scores it as a successful escape. The fork storm is
similar: it gets exactly the forks the PID limit allows, then `EAGAIN`, and what
it prints matches nobody's denial pattern.

The method fails hardest precisely where enforcement is working best. The
quieter your kill, the more likely your test calls it a breach.

Score post-conditions from the host instead. After each attempt, check state:

- Did a new mount appear in `/proc/mounts`?
- Does the written file exist?
- What `errno` did the syscall return?
- Which device and inode does `/proc/1/root` resolve to?
- Did the cgroup's `memory.events` oom_kill counter tick? What about
  `pids.events`?

That is what caught the Kata gap. The evidence line reads `PTRACE_ATTACH pid=2
errno=1 yama=absent`. Not a string the process printed, but what the kernel
returned and what the host could still see.

## What to take away

Three things, in order of how cheaply you can act on them.

Turn the hardening flags on. On both hosts I measured, they were free, and
`--network none` beat the bridge.

Do not assume your config is the thing enforcing your config. Run the same
workload under a different runtime and watch which controls survive.

And stop grading sandbox tests on what the attack printed. Grade them on what
the host can see afterward.

The substrate, the harness, and every raw trace are MIT-licensed.

Repository: https://github.com/akshayaggarwal99/boxed
Numbers in this article: https://github.com/akshayaggarwal99/boxed/tree/v0.3.2-paper
