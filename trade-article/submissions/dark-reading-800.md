# Dark Reading Commentary cut (700–800 words; paste in email body, no attachments)

**Headline:** The Container Hardening You Skipped Is Free. The Protection You Kept May Not Be Yours.
**Byline:** Akshay Kumar, Independent Researcher
**Bio:** Akshay Kumar builds and measures code-execution sandboxes for autonomous coding agents. His open-source substrate, Boxed, and every raw trace behind this article are MIT-licensed on GitHub.

---

Every AI coding agent runs the same loop: write a snippet, execute it in a sandbox, read the output, try again, dozens of times per task. Whatever the sandbox costs, you pay on every iteration. That is why most teams reach for plain Docker and skip the security flags, on the theory that a read-only root, dropped capabilities, a PID cap and no network each add latency a hot loop cannot afford.

I measured that theory, and it is wrong in a way that is good news. Then a stronger isolation boundary showed me something worse about what my hardening had been doing all along.

**Hardening was faster.** I ran the same create, exec, destroy cycle against the Docker Engine API two ways: stock `docker run` defaults, and the full hardened configuration. Same image, same command, five runs of 200 lifecycles each, interleaved. On a laptop, stock came in at a median of 216 ms and hardened at 153 ms, 63 ms faster. I assumed a bug and repeated the campaign on an idle cloud host running Ubuntu 24.04.4: 368 ms stock, 298 ms hardened, 70 ms faster. The intervals are within 2 ms either side.

The likely reason is that `--network none` skips creating a veth pair, attaching it to the bridge, and tearing both down, and that work costs more than the other flags save. The most security-relevant flag in the list is also the one that removes the most work. If you skipped hardening to keep your loop fast, you made your sandbox weaker and slower at once.

**Then I changed the boundary.** Docker lets you swap the OCI runtime per container. I ran the identical stack under `runc`, under gVisor's user-space kernel, and under Kata Containers 4.1, which boots a QEMU microVM per sandbox. Lifecycles were 354 ms, 405 ms and 7,824 ms. No surprise; a stronger boundary costs boot time.

The surprise was the escape probe: twelve attempts drawn from common container-escape patterns, each scored by a post-condition read from the host afterward, such as the `errno` a syscall returned or a cgroup counter, never by what the attempt printed. Under runc, 12 of 12 denied. Under gVisor, 12 of 12. Under Kata, the strictly stronger boundary, 11 of 12.

The `PTRACE_ATTACH` on my own in-sandbox agent succeeded. Inside the guest the workload and the agent run as the same user, root, so this is a same-UID attach within the microVM, not an escape to the host. It still matters: code under test could rewrite the process that reports on it.

**What was actually protecting me.** runc had never been blocking that attack. The host kernel had. Linux ships the Yama security module, and with `ptrace_scope=1`, Ubuntu's default and the setting on both hosts I measured, a process may only attach to its own descendants. Under runc the container shares the host kernel, so Yama was quietly enforcing that on every sandbox I tested. Kata's guest kernel has no Yama, and a same-uid `ptrace` needs no capability, so the attach went through.

I had `CapDrop ALL` in my configuration and had read that line as covering this. It never did. A host kernel policy I had not written and could not see from inside the container was carrying a control I believed was mine. When you harden a container you configure two things at once: the flags you wrote, and whatever the host kernel enforces underneath. The second set is not in your compose file, varies between distributions, and disappears the moment you move to an isolation technology that brings its own kernel. Upgrading to a stronger boundary can silently remove a protection you did not know you had.

The fix was one line. The agent now calls `prctl(PR_SET_DUMPABLE, 0)` at startup, after which the kernel refuses any tracer lacking `CAP_SYS_PTRACE`, which the driver already drops. Re-running the probe: 12 of 12 under all three runtimes, at no measurable cost.

**Why your escape tests probably lie to you.** None of this surfaces if you score the probe the usual way, by grepping the attempt's output for "Operation not permitted." Against my twelve vectors that method reports two wrong, both times calling an attack a success when the host evidence proves it was stopped. The memory bomb is the clearest case: the cgroup kills it, `SIGKILL` gives the process no chance to print, the harness sees empty output and scores an escape. The quieter your kill, the more likely your test calls it a breach.

Three things, cheapest first. Turn the hardening flags on; on both hosts they were free. Do not assume your configuration is the thing enforcing your configuration; run the same workload under a second runtime and watch which controls survive. And grade sandbox tests on what the host can see afterward, not on what the attack printed.
