# Security Boulevard cut (target 600–1,000 words; vendor-neutral, no links in body)

**Headline:** Hardening my AI agent sandbox made it faster. A microVM showed me what had really been protecting it.
**Byline:** Akshay Kumar, Independent Researcher
**Bio (for the form):** Akshay Kumar builds and measures code-execution sandboxes for autonomous coding agents. His open-source substrate, Boxed, and every raw trace behind this article are MIT-licensed on GitHub.

---

Every AI coding agent has the same shape underneath. It writes a snippet, runs it, reads the output, and tries again, dozens of times per task. The sandbox sits inside that loop, so you pay for it on every iteration.

That is why most teams reach for plain Docker and quietly skip the security flags: read-only root filesystem, drop all capabilities, `no-new-privileges`, a PID cap, a memory limit, no network. The folk wisdom is that each one costs latency you cannot afford in a hot loop.

I measured it. The folk wisdom is wrong, and finding that out led to a worse surprise about what my hardening had been doing all along.

## Hardening was faster

I ran the same create, exec, destroy cycle three ways against the Docker Engine API: stock `docker run` defaults, the same call with the full hardened configuration, and my own control plane on top of that. Same host, image and command, five runs of 200 lifecycles each, interleaved so host drift hit all three alike.

On a laptop, stock defaults came in at a median of 216 ms. Hardened: 153 ms, or 63 ms faster. I assumed I had a bug and repeated the campaign on an idle cloud host. Same direction: 368 ms stock, 298 ms hardened. The laptop ran Docker 29.5.2 in a Colima VM on kernel 6.8.0-117-generic; the cloud host ran Ubuntu 24.04.4, kernel 6.17.0-1022-gcp, Docker 29.7.2. Bootstrap 95 percent intervals on the laptop medians are within 2 ms either side.

The likely reason is dull once you see it, though I have not profiled the daemon to confirm the split. `--network none` means Docker never creates a veth pair, never attaches the container to the bridge, and never tears any of it down. That work costs more than the read-only root, the capability drop and the tmpfs mounts save. The most security-relevant flag in the list is also the one that removes the most work.

If you skipped hardening to keep your agent loop fast, you made your sandbox weaker and slower at the same time.

## Then I changed the boundary

Docker lets you swap the OCI runtime per container. So I ran the identical control plane, agent, image and harness under three of them on one nested-virtualization host: `runc`, gVisor's `runsc` user-space kernel, and Kata Containers 4.1, which boots a QEMU microVM per sandbox.

Lifecycle medians were 354 ms under runc, 405 ms under gVisor, and 7,824 ms under Kata. No surprise. A stronger boundary costs boot time.

The surprise was the escape probe. Each attempt is judged by a post-condition read from the host afterwards, such as the `errno` the syscall returned or a cgroup counter, never by what the attempt printed. Twelve attempts drawn from common container-escape patterns: a filesystem mount, reading the host through `/proc/1/root`, the Docker socket, `init_module`, a raw socket, a new user and PID namespace, egress to RFC1918 and to the cloud metadata endpoint, a fork storm, a memory bomb, a root filesystem write, and `PTRACE_ATTACH` on my in-sandbox agent.

Under runc, 12 of 12 denied. Under gVisor, 12 of 12. Under Kata, the strictly stronger boundary, 11 of 12.

The `PTRACE_ATTACH` call succeeded. Inside the guest the workload and my agent run as the same user, root, so this is a same-UID attach within the microVM, not an escape to the host. It still matters: code under test could rewrite the process that reports on it.

## What was actually protecting me

runc had never been blocking that attack. The host kernel had.

Linux ships a Yama LSM, and with `ptrace_scope=1`, Ubuntu's default and the setting on both hosts I measured, it limits attachment to descendant processes. Under runc the container shares the host kernel, so Yama was quietly doing the work. Kata's guest kernel has no Yama at all, and a same-uid `ptrace` needs no capability. So the attach succeeded.

I had `CapDrop ALL` in my config and had read that line as covering exactly this case. It never did. A host kernel policy I had not written, did not configure and could not see from inside was carrying a control I believed was mine. When you harden a container you are configuring two things at once: your own flags, and whatever the host kernel happens to enforce. The second set is invisible in your compose file, and it disappears the moment you move to a boundary that brings its own kernel.

The fix was one line. The agent now calls `prctl(PR_SET_DUMPABLE, 0)` at startup, after which the kernel refuses any tracer lacking `CAP_SYS_PTRACE`, which the driver already drops. Re-running the probe: 12 of 12 under all three runtimes, three runs each, at no measurable cost.

## Why your escape tests probably lie to you

None of this would have surfaced if I had scored the probe the usual way. Most escape harnesses capture stdout and stderr and grep for a denial string. Run that against my twelve vectors and it reports two of them wrong, both times claiming an attack succeeded when the host evidence proves it was stopped. The memory bomb is the clearest case: the cgroup kills it, `SIGKILL` gives the process no chance to print anything, the harness sees empty output and scores an escape. The quieter your kill, the more likely your test calls it a breach.

Score post-conditions from the host instead. Did a mount appear in `/proc/mounts`? Does the written file exist? What `errno` came back? Did the cgroup's `oom_kill` counter tick?

## What to take away

Turn the hardening flags on. On both hosts they were free, and `--network none` beat the bridge. If your agents need egress, an internal Docker network with inter-container traffic disabled keeps most of the speed.

Do not assume your config is the thing enforcing your config. Run the same workload under a second runtime and watch which controls survive.

And stop grading sandbox tests on what the attack printed. Grade them on what the host can see afterward.
