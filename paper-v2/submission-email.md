**To:** techresearch.pub@gmail.com
**Subject:** Paper submission — Boxed: A Docker-Based Code-Execution Substrate for Autonomous Code-Generating Agents

---

Hello,

Ankita shared this address with me. My paper is ready and attached (10 pages, IEEE format).

**Title:** Boxed: A Docker-Based Code-Execution Substrate for Autonomous Code-Generating Agents

**Author:** Akshay Kumar, Independent Researcher, United States. akumar8@mt.iitr.ac.in. I am the only author.

**What it is about:**

AI coding agents write code and then run it. That code has to run somewhere safe, and it runs many times in a single task, so the sandbox needs to be both secure and fast.

Today you get to pick two out of three. Plain Docker is fast and easy but usually left unsafe. Paid cloud sandboxes are safe but your code and data go to someone else's servers. Research microVMs are safe but were built for a different job.

Boxed is an open-source tool that gives you all three. You run it yourself, so your code stays with you. It adds only 16% on top of what plain Docker costs. I tested it against 12 known container escape attacks and it stopped all 12. I also tested OpenHands, a popular alternative, on the same machine: it was 21 times slower and stopped only 9 of the 12. One of the three it missed let me read the cloud server's access token from inside the sandbox.

Everything is free and open source, including the code, the test tools, and all the raw measurement data: https://github.com/akshayaggarwal99/boxed

**One thing before we go ahead:**

I am targeting IEEE for this paper. Can you place it with an IEEE conference or journal? If yes, please let me know which one, the timeline, and any fees.

If it is not IEEE, I would rather hold the paper back, since IEEE does not accept work that has already been published elsewhere.

Thanks,
Akshay Kumar
