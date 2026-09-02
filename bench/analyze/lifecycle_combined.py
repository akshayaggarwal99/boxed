#!/usr/bin/env python3
"""One lifecycle table for several hosts.

Usage: lifecycle_combined.py <results_dir> <out_tables_dir> LABEL=RUN ...
  e.g. lifecycle_combined.py results ../paper-v2/tables "MacBook Pro M1 Pro, colima VM (in use)=hardened-2026-09" "GCE n2-standard-4, native (idle)=linux-2026-09"
"""
import glob, sys
from pathlib import Path
import numpy as np, pandas as pd

def pooled(run, name):
    return pd.concat([pd.read_csv(f) for f in sorted(glob.glob(str(run / f"r*/{name}")))], ignore_index=True)

res, out = Path(sys.argv[1]), Path(sys.argv[2])
L = ["\\begin{table}[t]", "\\centering",
     "\\caption{Lifecycle latency on two hosts, medians over 5 runs $\\times$ 200 sequential create$\\rightarrow$exec$\\rightarrow$destroy cycles per configuration, same image and command. Raw Docker rows call the Engine API directly with no control plane and no agent. All values in ms.}",
     "\\label{tab:lifecycle}", "\\scriptsize", "\\setlength{\\tabcolsep}{2.5pt}", "\\begin{tabular}{@{}lrrrrrr@{}}", "\\toprule",
     "Configuration & create & exec & destroy & total & IQR & p95 \\\\"]
for spec in sys.argv[3:]:
    label, run = spec.split("=", 1)
    L += ["\\midrule", f"\\multicolumn{{7}}{{@{{}}l}}{{\\emph{{{label}}}}} \\\\"]
    for name, f in (("Raw Docker, stock", "baseline_default.csv"), ("Raw Docker, hardened", "baseline_hardened.csv"), ("Boxed (plane + agent)", "coldstart.csv")):
        df = pooled(res / run, f); t = df["total_ms"]
        L.append(f"\\quad {name} & {np.median(df['create_ms']):.0f} & {np.median(df['first_exec_ms']):.0f} & {np.median(df['destroy_ms']):.0f} & "
                 f"\\textbf{{{np.median(t):.0f}}} & {np.percentile(t,75)-np.percentile(t,25):.0f} & {np.percentile(t,95):.0f} \\\\")
L += ["\\bottomrule", "\\end{tabular}", "\\end{table}"]
(out / "lifecycle.tex").write_text("\n".join(L) + "\n")
print("wrote lifecycle.tex (combined)", file=sys.stderr)
