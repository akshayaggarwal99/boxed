# Pitch — The New Stack / InfoQ

Send as plain email. One venue at a time; The New Stack first.

**Subject:** Pitch: hardening containers made them faster, then a microVM broke a protection I thought I had

---

Hello,

I'd like to pitch a ~1500-word contributed article on two results I did not expect while benchmarking sandboxes for AI coding agents.

Hardening a container made it **faster**. Read-only root, `CapDrop ALL`, `no-new-privileges`, a PID cap and `--network none` ran 63 ms below `docker run`'s defaults on a laptop and 70 ms below on a GCE instance, because skipping bridge setup costs more than the other flags save. The config people drop to stay fast is negative-cost.

Then I swapped runc for a Kata microVM, a strictly stronger boundary, and an attack that had been blocked started working: `PTRACE_ATTACH` on my in-sandbox agent. runc was never stopping it. The host kernel's Yama policy was, and Kata's guest kernel has no Yama. My capability drops looked like they covered that. They didn't.

I caught it only because I score escape probes on post-conditions read from the host instead of grepping stdout for a denial string. That method scored 2 of my 12 vectors wrong: the cgroup SIGKILLs the memory bomb before it prints, and the usual probe reads that silence as a successful attack.

So: why hardening is free, why a stronger boundary exposed a weaker assumption, and how to score a sandbox test so it stops lying to you. Vendor-neutral, reproducible on plain `docker run`. My tooling is MIT and the raw traces are public, but it isn't a product and the article doesn't pitch it.

I'm an independent researcher; the underlying measurement paper is going to IEEE IC2E. Happy to send an outline or the full draft.

Thanks,
Akshay Kumar
akshaykumarinusa@gmail.com
github.com/akshayaggarwal99/boxed
