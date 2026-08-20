# Submission Blockers

Do not submit `main.tex` to an IEEE venue until each item below is completed
and the manuscript is updated with the resulting evidence.

1. Run `bench/security/escapes.sh` against the hardened Docker revision and
   replace signature-only classification with post-condition checks.
2. Repeat cold-start, throughput, and overhead experiments on a quiesced native
   Linux host. Run each throughput point at least 30 times and report a
   dispersion measure or confidence interval.
3. Rerun the agent trace after the exit-status correction. Record the model
   identifier, date, prompts, decoding configuration, task IDs, expected tests,
   raw completions, and real process exit status. Do not label it HumanEval
   unless official HumanEval tasks and tests are used.
4. Compare the same lifecycle workload against at least one relevant baseline
   under the same host/image/resource constraints.
5. Either implement and evaluate a second backend or narrow the paper's claim
   to an extensible Docker prototype.
6. Resolve the remaining `ptrace` protection with a tested seccomp policy, or
   clearly state that Docker isolation remains unsuitable for hostile code.
7. Build the final PDF with the selected venue's template, validate references,
   run the IEEE PDF checker, remove all Word review comments from any submitted
   DOCX/PDF, and add the required ORCID metadata.
