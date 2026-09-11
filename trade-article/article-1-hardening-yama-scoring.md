# Article 1: Hardening was free, a stronger boundary removed a protection, and the test that caught it

**Source:** Boxed paper (paper-v2), Sections V.B, V.E, V.G and V.H.
**Length:** ~1,900 words. A 1,031-word cut for VentureBeat exists as a separate draft.
**Audience:** platform, infrastructure and security engineers who run untrusted code in containers.
**Status:** review draft. Every number is verified against the raw traces at https://github.com/akshayaggarwal99/boxed/tree/v0.3.2-paper/bench/results . Prose to be retyped by the author before submission; several target outlets ban AI-assisted writing.

## Title options

1. I hardened my AI agent sandbox and it got faster. Then a microVM showed me what I had actually secured.
2. The container hardening you skipped is free. The hardening you kept may not be yours.
3. Your sandbox's best protection might belong to the host kernel
4. Hardening made my containers faster, and a stronger boundary made one thing weaker
5. What a Kata microVM taught me about my Docker hardening

**Byline:** Akshay Kumar, Independent Researcher, akumar8@mt.iitr.ac.in

---

Every AI coding agent has the same shape underneath. It writes a snippet, runs it, reads the output, and tries again. Dozens of times per task. The sandbox that runs the snippet sits inside that loop, so whatever it costs, you pay on every iteration.

That is why most teams reach for plain Docker and then quietly skip the security flags. Read-only root filesystem, drop every capability, `no-new-privileges`, a PID cap, a memory limit, no network. The folk wisdom is that each flag costs latency you cannot afford in a hot loop, and that a container is a container anyway.

I spent this year measuring that. The folk wisdom is wrong in a way that is good news, and finding that out led to a worse discovery about what my hardening had been doing all along.

## Hardening was faster

I ran the same create, exec, destroy cycle three ways against the Docker Engine API on one machine. Stock `docker run` defaults. The same call with the full hardened host configuration: read-only root, `CapDrop ALL`, `no-new-privileges`, a PID limit, network `none`, one CPU, 512 MiB, tmpfs mounts. And my own control plane on top of that hardened configuration. Same image, same command, five runs of 200 sequential lifecycles per configuration, interleaved so that whatever the host was doing at the time hit all three alike.

| Configuration | Laptop (colima VM) | GCE n2-standard-4, idle |
|---|---|---|
| Raw Docker, stock defaults | 216 ms | 368 ms |
| Raw Docker, hardened | **153 ms** | **298 ms** |
| Hardened plus control plane and agent | 178 ms | 347 ms |

The laptop runs Docker 29.5.2 inside a Colima VM on kernel 6.8.0-117-generic, arm64. The cloud host is a Compute Engine n2-standard-4 on Ubuntu 24.04.4, kernel 6.17.0-1022-gcp, Docker 29.7.2. Those are medians; with a bootstrap 95 percent interval on the laptop they are 216 ms (214 to 218, interquartile range 44) for stock, 153 ms (151 to 155, IQR 42) hardened, and 178 ms (177 to 179, IQR 33) with the control plane.

> **[Figure 1b: `figures/fig-1b-hardening-faster@2x.png`]**
> *Hardening was faster than stock on both hosts. Median create, exec, destroy lifecycle against the Docker Engine API; five runs of 200 sequential lifecycles per configuration, same image and command, shared scale across panels.*

Hardening was 63 ms faster than stock on the laptop. I assumed I had a bug and repeated the whole campaign on an idle Compute Engine host. Same direction: 70 ms faster. The absolute numbers move between hosts. The sign does not.

The likely reason is dull once you see it. `--network none` means Docker never creates a veth pair, never attaches it to the bridge, never programs the address, and never tears any of that down. That work costs more than the read-only root, the capability drop and the tmpfs mounts save. I have not profiled the daemon to confirm that split; it is the standard account of what the flag skips, and the measured difference is the part to trust. The single most consequential security flag in the list is also the one that removes the most work from the lifecycle.

The control plane I built on top costs 25 ms, or 16 percent of the raw lifecycle: 22 ms on create, 2 ms on the first exec, 1 ms on destroy. An idle sandbox holds 0.43 MiB and 0.01 percent of a CPU. I mention those so nobody reads this as a claim that my software is faster than Docker. It is not. The hardening is.

If you skipped hardening to keep your agent loop fast, you made your sandbox weaker and slower at the same time.

## Then I changed the boundary

Docker lets you name an alternative OCI runtime per container. The driver passes that through as one field. So I ran the identical control plane, agent, image and harness under three runtimes on one host: `runc`, gVisor's `runsc`, which interposes a user-space kernel, and Kata Containers, which boots a QEMU microVM with its own guest kernel for every sandbox. The versions were runc as bundled with Docker CE 29.7.2, runsc from gVisor's apt repository as of September 2026, and Kata 4.1 with runtime-rs on QEMU, all on the same nested-virtualization Compute Engine host.

Lifecycle medians were 354 ms under runc, 405 ms under gVisor, and 7,824 ms under Kata. No surprise. A stronger boundary costs boot time, and it costs density too: at a concurrency of sixteen, Kata failed to create 23 of 72 sandboxes where the other two runtimes had no errors. The companion capacity-planning article has those numbers.

The surprise was the escape probe. Each attempt is scored two ways. The usual way greps the attempt's output for a denial string. The way I trust reads a post-condition from the host after the attempt: the `errno` the syscall returned, a counter in the sandbox's cgroup, an entry in `/proc/mounts`, a file that does or does not exist afterwards. The post-condition decides every verdict below; the string method is kept only so the two can be compared later. I run twelve attempts drawn from common container-escape patterns, each in a fresh sandbox, the whole suite three times:

- mount a filesystem
- read the host through `/proc/1/root`
- reach the Docker or containerd socket
- load a kernel module with `init_module`
- open a raw socket
- create a new user and PID namespace
- egress to an RFC1918 address
- egress to the link-local cloud metadata endpoint
- a fork storm
- a memory bomb that touches every page
- write to the root filesystem
- `PTRACE_ATTACH` on my own in-sandbox agent

Under runc, twelve of twelve denied, identical across all three runs. Under gVisor, twelve of twelve. Under Kata, the strictly stronger boundary, **eleven of twelve**.

The `PTRACE_ATTACH` call succeeded. Inside the microVM the workload and my agent run as the same user, root, so this is a same-UID attach within the guest, not an escape to the host. It still matters. Code the agent is supposed to be supervising could read and rewrite the process that reports its results.

## What was actually protecting me

runc had never been blocking that attack. The host kernel had.

Linux ships a Yama security module. With `ptrace_scope=1`, which is Ubuntu's default and was the setting on both hosts I measured (the probe's evidence line under runc records `yama=1`), a process may only attach to its own descendants. Under runc the container shares the host kernel, so Yama was quietly enforcing that on every sandbox I had ever measured. Kata's guest kernel has no Yama. Inside the microVM my workload and my agent both run as root, a same-uid `ptrace` needs no capability at all, and so the attach succeeded.

I had `CapDrop ALL` in the configuration. I had looked at that line many times and read it as covering exactly this case. It never did. A host kernel policy I had not written, did not configure, and could not see from inside the container was carrying a control I believed was mine.

> **[Figure 1a: `figures/fig-1a-two-kernels@2x.png`]**
> *Same flags, same attack, different kernel. Under runc the container shares the host kernel, and Yama `ptrace_scope=1` refuses the attach. Under Kata the microVM's guest kernel has no Yama, and the identical attach succeeds. The fix, `prctl(PR_SET_DUMPABLE, 0)` in the agent, closes it under all three runtimes.*

That is the part worth generalizing. When you harden a container you are configuring two things at once: the flags you wrote, and whatever the host kernel happens to enforce underneath them. The second set does not appear in your compose file. It varies between distributions and between cloud images. And it disappears the moment you move to an isolation technology that brings its own kernel. Upgrading to a stronger boundary can silently remove a protection you did not know you were relying on.

The fix was one line. The agent now calls `prctl(PR_SET_DUMPABLE, 0)` at startup. After that the kernel's own access check refuses any tracer that lacks `CAP_SYS_PTRACE`, and the driver already drops that capability. I re-ran the probe with the patched agent under all three runtimes, three runs each: twelve of twelve everywhere. The evidence line under Kata reads `PTRACE_ATTACH pid=2 errno=1 yama=absent`. The call sits off the lifecycle path and cost nothing I could measure.

Two other things changed under the stronger boundary, and they point the same direction. Under gVisor and Kata, `unshare` succeeds inside the sandbox, but the new namespace's inode never appears in the host's `/proc`; the namespace exists only inside the sandbox's own kernel. And under both, the fork storm and the memory bomb kill the whole sandbox, agent included, where under runc only the offending process dies. Different kernel, different failure shape. The probe has to be scored by what actually happened, not by what a runc-shaped probe expects to see.

## Why your escape tests probably lie to you

None of this would have surfaced if I had scored the probe the usual way.

Most escape harnesses capture stdout and stderr from the attempt and grep for a denial string: "Operation not permitted", "Read-only file system", and so on. I score every vector twice, once that way and once by post-conditions read from the host after the attempt. On the same twelve attempts the signature method reports **two of them wrong**, both times claiming an attack succeeded when the host evidence proves it was stopped.

The memory bomb is the clearest case. The cgroup kills it, and the host's `memory.events` counter records the `oom_kill`. But `SIGKILL` gives the process no chance to print anything, so the harness sees empty output, finds no denial string, and scores a successful escape. The fork storm is similar. It gets exactly the number of forks the PID limit allows, then `EAGAIN`, and nothing it prints matches anyone's denial pattern.

The method fails hardest precisely where enforcement is working best. The quieter your kill, the more likely your test calls it a breach.

Score post-conditions instead. After each attempt, read state from the host:

| Vector | What to check |
|---|---|
| Mount | Did a new entry appear in `/proc/mounts`? |
| Root filesystem write | Does the file exist afterwards? |
| Raw socket, module load, ptrace | What `errno` did the syscall return? |
| Host filesystem read | Which device and inode does `/proc/1/root` resolve to? |
| Memory bomb | Did the cgroup's `memory.events` oom_kill counter tick? |
| Fork storm | Did `pids.events` max tick? What did `fork` return? |
| Namespace | Does the new namespace inode appear in the host's `/proc`? |

That is what caught the Kata gap, and it is what made the runtime comparison meaningful at all. Not a string the process printed. What the kernel returned, and what the host could still see.

## What I would do differently

I would have run the runtime swap first. The whole point of building the driver behind a narrow interface was that the boundary could be replaced, and I spent months measuring one boundary before trying the second. The second one found the bug in a week.

I would also have written the post-condition scorer before the first probe rather than after the third. Signature scoring is what the existing tools do, so I copied it. It would have told me my hardening was fine under runc and I would have believed it.

## What to take away

Three things, cheapest first.

Turn the hardening flags on. On both hosts I measured, they were free, and `--network none` beat the bridge. If your agents need egress for package installs or API calls, `none` is not available to you; the driver's alternative is an operator-created internal network with inter-container traffic disabled, which keeps the bridge and most of the speed, and the companion article measures it.

Do not assume your configuration is the thing enforcing your configuration. Run the same workload under a second runtime and watch which controls survive the move. The ones that do not were never yours.

Stop grading sandbox tests on what the attack printed. Grade them on what the host can see afterwards.

The substrate, the harness, the campaign runner, and every raw trace are MIT-licensed, and every number in this article regenerates from those traces with one `make` command.

Repository: https://github.com/akshayaggarwal99/boxed
Numbers in this article: https://github.com/akshayaggarwal99/boxed/tree/v0.3.2-paper
