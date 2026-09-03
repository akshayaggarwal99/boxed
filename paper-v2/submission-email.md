**To:** techresearch.pub@gmail.com
**Subject:** Paper for review — Boxed: A Docker-Based Code-Execution Substrate for Autonomous Code-Generating Agents

---

Hello,

Ankita shared this address. My paper is ready; the full manuscript is attached (10 pages, IEEE conference format, IEEEtran).

**Title:** Boxed: A Docker-Based Code-Execution Substrate for Autonomous Code-Generating Agents

**Author:** Akshay Kumar, Independent Researcher, United States — akumar8@mt.iitr.ac.in. Sole author, no coauthors.

**Abstract:**

Autonomous large language model (LLM) agents write, run, and revise code inside their reasoning loop, so the sandbox sits on the hot path of every task. Roll-your-own Docker is quick and usually ships without a read-only root, capability drops, process caps, or egress control. Hosted sandbox services are hardened but move the operator's code and data to a third party. Research virtual machine monitors isolate well and are built to boot once. Boxed is an open-source substrate for agentic code execution: a self-hosted Go control plane, a 1.32 MiB Rust in-sandbox agent that streams stdout, stderr, and files over JSON-RPC 2.0, and a driver interface whose four lifecycle methods admit backends beyond the Docker driver that exists today. Against the Docker Engine API on the same host, image, and command, the control plane and agent cost 16% of a raw Docker lifecycle, on a laptop and on an idle server. The hardened configuration beats stock Docker defaults, because dropping the network costs less than setting up the bridge. With the OCI runtime swapped under an unchanged driver, a lifecycle takes 354 ms with runc, 405 ms with gVisor, and 7824 ms with a Kata microVM, and the substrate's own cost stays close to constant. A twelve-vector escape probe scored by post-conditions read from the host denies all twelve under runc and gVisor and eleven under Kata, whose guest kernel lacks the host policy that had been stopping a ptrace attach on the in-sandbox agent; one line in the agent closes it. An internal, inter-container-isolated network and no CPU quota cut the microVM lifecycle to 2856 ms with all twelve denied. On the same host the OpenHands agent-server sandbox, in its SDK's default configuration, takes 7.6 s per lifecycle and stops 9 of the twelve; the cloud metadata endpoint answers from inside the workload, and with it the instance's service-account token. Throughput on one host is bound by the Docker daemon: four times the cores gives 1.2x the sandboxes per second. Boxed, the harness, and every raw trace are MIT-licensed.

**Keywords:** sandboxing, code execution isolation, large language models, autonomous agents, containers, Docker, JSON-RPC

**Artifact:** code, benchmark harness, campaign runner, and every raw trace are MIT-licensed at https://github.com/akshayaggarwal99/boxed (tag v0.3.2-paper). Every number, table, and figure in the paper regenerates from the raw traces with one `make` command.

Before I proceed, could you confirm the process, the timeline, any fees, and which venue and indexing this goes to? I am also weighing an IEEE conference submission, so I need to know whether publishing with you would rule that out.

Thanks,
Akshay Kumar
