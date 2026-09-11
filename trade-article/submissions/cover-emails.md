# Cover emails, one per outlet. Send ONE at a time; each outlet requires exclusivity.

Unlisted review copy with figures (noindex, expires 2026-10-10):
https://hiakshay-469dc--article1-review-6qtpqwzq.web.app/

Figures to attach where attachments are allowed:
figures/fig-1b-hardening-faster@2x.png, figures/fig-1a-two-kernels@2x.png

---

## 1. InfoQ (editors@infoq.com): full draft, first choice

Subject: Article submission: container hardening measured faster, and a microVM removed a protection (1,900 words, original data)

Hello InfoQ editors,

I would like to submit an original, unpublished article for InfoQ's architecture and infrastructure coverage.

Title: Hardening was free, a stronger boundary removed a protection, and the test that caught it
Length: about 1,900 words, two figures, two tables.

What it reports, all from my own measurements on an open-source sandbox substrate for AI coding agents:

- Applying the full Docker hardening set (read-only root, CapDrop ALL, no-new-privileges, PID and memory limits, network none) made the create/exec/destroy lifecycle 63 ms faster on a laptop and 70 ms faster on an idle Compute Engine host, against stock defaults. Five runs of 200 lifecycles per configuration, bootstrap intervals reported.
- Running the identical stack under runc, gVisor and Kata Containers, a twelve-vector escape probe was denied 12/12 under runc and gVisor but 11/12 under Kata. The PTRACE_ATTACH that succeeded had been blocked under runc not by my configuration but by the host kernel's Yama policy, which a microVM's guest kernel does not have. The one-line fix and re-verification are included.
- Scoring escape tests by grepping for denial strings misclassified 2 of the 12 vectors; scoring by host post-conditions (errno, cgroup counters, /proc/mounts) caught the gap.

The substrate, the probe, and every raw trace are MIT-licensed, and every number regenerates from the traces with one make command. Disclosure: I am the author of the substrate (Boxed) and say so in the piece.

The full text is pasted below. A review copy with the figures rendered is at the unlisted link above; the two figures are attached as PNG.

Byline: Akshay Kumar, Independent Researcher, akumar8@mt.iitr.ac.in
Bio: Akshay Kumar builds and measures code-execution sandboxes for autonomous coding agents.

Thank you for considering it.

Akshay Kumar

[paste full article body from article-1-hardening-yama-scoring.md, from "Every AI coding agent" to the repository links]

---

## 2. Security Boulevard (editorial@securityboulevard.com or the paperform): 1,000-word cut

Subject: Contributed article: Hardening my AI agent sandbox made it faster; a microVM showed what had really been protecting it

Hello Security Boulevard editorial team,

I would like to contribute an original, exclusive, vendor-neutral article for practitioners running untrusted or AI-generated code in containers. It has not appeared anywhere else.

Title: Hardening my AI agent sandbox made it faster. A microVM showed me what had really been protecting it.
Length: about 1,000 words.

In one paragraph: I measured the full Docker hardening set against stock defaults on two hosts and found it faster, not slower, because network none skips bridge setup. Running the same stack under runc, gVisor and Kata, a twelve-vector escape probe passed 12/12 under the first two and 11/12 under Kata, because the ptrace protection I thought my capability drop provided was actually the host kernel's Yama policy, which a microVM's guest kernel lacks. The piece gives the one-line fix, and shows that grep-for-denial-string escape tests misgrade 2 of 12 vectors where host post-condition scoring does not.

The body contains no product links; the open-source repository is mentioned only in the bio. The full text is pasted below and a review copy with figures is at the unlisted link above.

Byline: Akshay Kumar, Independent Researcher
Bio: Akshay Kumar builds and measures code-execution sandboxes for autonomous coding agents. His open-source substrate, Boxed, and every raw trace behind this article are MIT-licensed on GitHub.

Thank you,
Akshay Kumar
akumar8@mt.iitr.ac.in

[paste submissions/security-boulevard-1000.md body]

---

## 3. Dark Reading Commentary (commentary@darkreading.com): 800-word cut, text in body, NO attachments

Subject: Commentary submission: The container hardening you skipped is free. The protection you kept may not be yours.

Hello Dark Reading commentary editors,

Please consider the following original, exclusive Commentary piece (about 800 words). It has not been published elsewhere and contains no vendor pitch.

It reports my own measurements: the standard Docker hardening flags made an AI agent's sandbox lifecycle faster on two hosts, and moving the same workload from runc to a Kata microVM removed a ptrace protection that turned out to belong to the host kernel's Yama policy rather than to my configuration. It closes with three practical takeaways, including why grep-based escape tests misgrade quiet kills.

Byline: Akshay Kumar, Independent Researcher
Bio: Akshay Kumar builds and measures code-execution sandboxes for autonomous coding agents. His open-source substrate and every raw trace behind this piece are MIT-licensed on GitHub.
Headshot: available on request.

If it is useful, a review copy with the two figures rendered is at the unlisted link above.

Thank you,
Akshay Kumar
akumar8@mt.iitr.ac.in

[paste submissions/dark-reading-800.md body]

---

## 4. Infosecurity Magazine (beth.maundrill@rxglobal.com, cc james.coker@rxglobal.com): pitch only

Subject: Op-ed pitch: container hardening is free, and a stronger sandbox boundary can remove a protection you never configured

Hello Beth,

I would like to pitch an exclusive op-ed of 800 to 1,000 words for Infosecurity Magazine.

Proposed title: The container hardening you skipped is free. The protection you kept may not be yours.

Synopsis: Teams running AI-generated code skip Docker's hardening flags to protect latency. I measured the full flag set against stock defaults on two hosts and it was faster by 63 and 70 ms per lifecycle, because network none skips bridge setup. I then ran the same sandbox under runc, gVisor and a Kata microVM with a twelve-vector escape probe. Kata, the strongest boundary, was the only one that let a ptrace attach through, because the control I credited to my capability drop was really the host kernel's Yama policy, which a guest kernel lacks. The piece argues, with the counter-evidence, that string-matching escape tests would have hidden this, and gives the one-line fix and a post-condition scoring method readers can adopt.

All numbers are my own and reproducible from MIT-licensed traces. I am the author of the open-source substrate measured and would disclose that in the piece.

Name: Akshay Kumar. Title: Independent Researcher. Headshot and short bio available on acceptance.

Thank you for your time,
Akshay Kumar
akumar8@mt.iitr.ac.in

---

## 5. VentureBeat: Typeform, not email
https://venturebeat.com/guest-posts (form at r39crwmcu9m.typeform.com/to/NEzWFTji). Needs an editable Google Doc link to the 1,195-word draft-venturebeat.md. Retype first: VentureBeat bans AI-assisted writing.

## 6. Cybersecurity Dive: web form, not email
https://www.cybersecuritydive.com/opinion/submit-opinion/ ~1,000 words; no AI use; exclusive 30 days.
