# Cover email to techresearch.pub

## Sent 3 Sep 2026, 11:57 AM

**From:** akshaykumarinusa@gmail.com
**To:** techresearch.pub@gmail.com
**Cc:** ankita, adityamthakre@gmail.com
**Subject:** Paper submission inquiry

---

Hello,

Ankita (from Jinee) shared this address with me. My paper is ready (10 pages, IEEE format).

**Title:** Boxed: A Docker-Based Code-Execution Substrate for Autonomous Code-Generating Agents

**Author:** Akshay Kumar, Independent Researcher, United States. akumar8@mt.iitr.ac.in. CCed coauthor.

**What it is about:**

AI coding agents write code and then run it. That code has to run somewhere safe, and it runs many times in a single task, so the sandbox needs to be both secure and fast.

Today you get to pick two out of three. Plain Docker is fast and easy but usually left unsafe. Paid cloud sandboxes are safe but your code and data go to someone else's servers. Research microVMs are safe but were built for a different job.

Boxed is an open-source tool that gives you all three. You run it yourself, so your code stays with you. It adds only 16% on top of what plain Docker costs. I tested it against 12 known container escape attacks and it stopped all 12. I also tested OpenHands, a popular alternative, on the same machine: it was 21 times slower and stopped only 9 of the 12. One of the three it missed let me read the cloud server's access token from inside the sandbox.

Everything is free and open source, including the code, the test tools, and all the raw measurement data.

**Before I send the paper:**

I am targeting IEEE. Can you place it with an IEEE conference or journal? If yes, please let me know which one, the timeline, and any fees.

Thanks,
Akshay

---

## Differences from the draft, and what they leave open

The sent version dropped three things from the draft. Recording them so the follow-up can pick them up.

1. **The github URL.** The text still says everything is open source but no longer says where. Send `https://github.com/akshayaggarwal99/boxed` (tag `v0.3.2-paper`) in the reply.

2. **The fallback line:** "If it is not IEEE, I will hold the paper back, since IEEE does not accept work that has already been published elsewhere." Without it, a non-IEEE offer is not pre-refused. If the reply names a non-IEEE venue, say no explicitly before anything else moves. The PDF was not attached, so nothing is committed yet.

3. **Sole authorship.** The draft said "I am the only author"; the sent version says "CCed coauthor" (adityamthakre@gmail.com). `main.tex` still has a single-author block. See the open item in SUBMISSION-BLOCKERS.md.

## Status

- Awaiting reply. Nothing sent but title and summary; no PDF, no manuscript file.
- IC2E 2026 closed (papers were due 15 May 2026, conference 13-15 Oct 2026). The next IEEE window is IC2E 2027, CFP not yet published.
