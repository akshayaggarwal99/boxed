#!/usr/bin/env python3
"""One figure: median create->exec->destroy lifecycle for every configuration
measured in the paper, on a log axis, grouped by host.

Usage: summary_fig.py <results_dir> <out_figures_dir>
"""
import glob, sys
from pathlib import Path
import numpy as np, pandas as pd
import matplotlib; matplotlib.use("Agg"); import matplotlib.pyplot as plt
plt.rcParams.update({"font.family": "serif", "font.size": 7.5, "axes.labelsize": 8,
                     "figure.dpi": 200, "savefig.bbox": "tight", "savefig.pad_inches": 0.02})

res, out = Path(sys.argv[1]), Path(sys.argv[2])
def med(run, name):
    fs = sorted(glob.glob(str(res / run / f"r*/{name}")))
    return np.median(pd.concat([pd.read_csv(f) for f in fs])["total_ms"]) if fs else np.nan

C_BOXED, C_HARD, C_DEF, C_OH = "#1f5fbf", "#e07b39", "#9a9a9a", "#b03030"
rows = [  # (label, value, colour, group)
 ("raw Docker, stock",        med("hardened-2026-09", "baseline_default.csv"),  C_DEF,   "Laptop, colima VM (runc)"),
 ("raw Docker, hardened",     med("hardened-2026-09", "baseline_hardened.csv"), C_HARD,  "Laptop, colima VM (runc)"),
 ("Boxed",                    med("hardened-2026-09", "coldstart.csv"),         C_BOXED, "Laptop, colima VM (runc)"),
 ("raw Docker, stock",        med("linux-2026-09", "baseline_default.csv"),     C_DEF,   "GCE n2-standard-4, native (runc)"),
 ("raw Docker, hardened",     med("linux-2026-09", "baseline_hardened.csv"),    C_HARD,  "GCE n2-standard-4, native (runc)"),
 ("Boxed",                    med("linux-2026-09", "coldstart.csv"),            C_BOXED, "GCE n2-standard-4, native (runc)"),
 ("Boxed, runc",              med("linux-runc-2026-09", "coldstart.csv"),       C_BOXED, "GCE n2-standard-4, nested virt"),
 ("Boxed, gVisor",            med("linux-runsc-2026-09", "coldstart.csv"),      C_BOXED, "GCE n2-standard-4, nested virt"),
 ("Boxed, Kata",              med("linux-kata-2026-09", "coldstart.csv"),       C_BOXED, "GCE n2-standard-4, nested virt"),
 ("Boxed, Kata + cfg",        med("linux-kata-aware-2026-09", "coldstart.csv"), C_BOXED, "GCE n2-standard-4, nested virt"),
 ("OpenHands agent-server",   med("linux-openhands-2026-09", "coldstart.csv"),  C_OH,    "GCE n2-standard-4, nested virt"),
]
rows = [r for r in rows if not np.isnan(r[1])]
# header rows: an empty entry before each group
items = []; last = None
for r in rows:
    if r[3] != last:
        items.append((r[3], None, None, r[3])); last = r[3]
    items.append(r)
fig, ax = plt.subplots(figsize=(3.4, 2.9))
y = np.arange(len(items))[::-1]
for yi, it in zip(y, items):
    if it[1] is None: continue
    ax.barh(yi, it[1], color=it[2], height=0.7)
    v = it[1]; ax.text(v * 1.08, yi, f"{v/1000:.2f} s" if v >= 1000 else f"{v:.0f} ms", va="center", fontsize=6.5)
ax.set_yticks(y)
ax.set_yticklabels([it[0] for it in items])
for lab, it in zip(ax.get_yticklabels(), items):
    if it[1] is None: lab.set_fontweight("bold"); lab.set_fontsize(7)
ax.set_xscale("log"); ax.set_xlim(100, 30000); ax.set_xlabel("median create+exec+destroy (ms, log)")
ax.set_xticks([100, 300, 1000, 3000, 10000]); ax.set_xticklabels(["100", "300", "1000", "3000", "10000"])
ax.xaxis.set_minor_formatter(matplotlib.ticker.NullFormatter())
ax.grid(True, axis="x", alpha=0.3)
fig.savefig(out / "summary.pdf"); plt.close(fig)
print("figures/summary.pdf written", file=sys.stderr)
