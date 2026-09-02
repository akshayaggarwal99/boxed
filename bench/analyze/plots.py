#!/usr/bin/env python3
"""Generate the paper's figures from a benchmark campaign.

Usage: plots.py <results/RUN> <out_figures_dir>
"""
from __future__ import annotations

import glob
import sys
from pathlib import Path

import matplotlib
matplotlib.use("Agg")
import matplotlib.ticker
import matplotlib.pyplot as plt
import numpy as np
import pandas as pd

plt.rcParams.update({
    "font.family": "serif", "font.size": 8, "axes.labelsize": 8,
    "legend.fontsize": 7, "figure.dpi": 200, "savefig.bbox": "tight",
    "savefig.pad_inches": 0.02,
})
C_BOXED, C_HARD, C_DEF = "#1f5fbf", "#e07b39", "#7a7a7a"


def cdf(ax, series, label, **kw):
    s = np.sort(np.asarray(series, dtype=float))
    ax.plot(s, np.arange(1, len(s) + 1) / len(s), label=label, **kw)


def load_all(run: Path, name: str) -> pd.DataFrame:
    return pd.concat([pd.read_csv(p) for p in sorted(glob.glob(str(run / f"r*/{name}")))], ignore_index=True)


def plot_lifecycle(run: Path, out: Path):
    boxed, hard, dflt = load_all(run, "coldstart.csv"), load_all(run, "baseline_hardened.csv"), load_all(run, "baseline_default.csv")
    fig, a = plt.subplots(figsize=(3.4, 2.0))
    cdf(a, dflt["total_ms"], "raw Docker, stock defaults", color=C_DEF, ls=":")
    cdf(a, hard["total_ms"], "raw Docker, hardened config", color=C_HARD, ls="--")
    cdf(a, boxed["total_ms"], "Boxed (plane + agent)", color=C_BOXED)
    a.set_xscale("log"); a.set_xlabel("create+exec+destroy (ms, log)"); a.set_ylabel("CDF")
    a.set_xticks([100, 200, 500, 1000, 2000]); a.set_xticklabels(["100", "200", "500", "1000", "2000"])
    a.xaxis.set_minor_formatter(matplotlib.ticker.NullFormatter())
    a.legend(loc="lower right", frameon=False); a.grid(True, alpha=0.3)
    fig.savefig(out / "lifecycle.pdf"); plt.close(fig)


def plot_throughput(run: Path, out: Path):
    tps = sorted(glob.glob(str(run / "tp*/throughput.csv")))
    if not tps:
        return
    tp = pd.concat([pd.read_csv(p).assign(run=i) for i, p in enumerate(tps)], ignore_index=True)
    g = tp.groupby("conc")["rps"]
    fig, ax = plt.subplots(figsize=(3.4, 1.8))
    for _, r in tp.groupby("run"):
        r = r.sort_values("conc"); ax.plot(r["conc"], r["rps"], color=C_BOXED, alpha=0.15, lw=0.8)
    ax.errorbar(g.mean().index, g.mean(), yerr=g.std(ddof=1), marker="o", color=C_BOXED, capsize=3, lw=1.2, label=f"mean $\\pm$ sd, {len(tps)} sweeps")
    ax.set_xscale("log", base=2); ax.set_xlabel("client concurrency"); ax.set_ylabel("sandboxes / s")
    ax.legend(frameon=False, loc="lower center"); ax.grid(True, alpha=0.3)
    fig.savefig(out / "throughput_curve.pdf"); plt.close(fig)


def plot_overhead(run: Path, out: Path):
    p = run / "overhead.csv"
    if not p.exists():
        return
    df = pd.read_csv(p)
    fig, axes = plt.subplots(1, 2, figsize=(3.4, 1.9))
    for ax, col, lab in ((axes[0], "rss_mib", "working set (MiB)"), (axes[1], "cpu_pct", "CPU (%)")):
        jitter = (np.random.default_rng(1).random(len(df)) - 0.5) * 0.3
        ax.scatter(jitter, df[col], s=8, color=C_BOXED, alpha=0.7)
        ax.hlines(np.median(df[col]), -0.3, 0.3, color=C_HARD, lw=1.2, label="median")
        ax.set_xlim(-0.6, 0.6); ax.set_xticks([]); ax.set_ylabel(lab); ax.grid(True, axis="y", alpha=0.3)
        ax.set_title(f"n={len(df)}", fontsize=7)
    axes[0].legend(frameon=False, fontsize=6)
    fig.savefig(out / "overhead_strip.pdf"); plt.close(fig)


def plot_agent(run: Path, out: Path):
    p = run / "agent_trace.csv"
    if not p.exists():
        return
    reps = [p] + sorted(glob.glob(str(run / "agent_rep*/agent_trace.csv")))
    df = pd.concat([pd.read_csv(x, dtype={"passed": str}) for x in reps], ignore_index=True)
    df = df[df["passed"].isin(["true", "false"])].copy()
    if df.empty:
        return
    llm = df["llm_ms"].astype(float)
    sb = df["create_ms"].astype(float) + df["exec_ms"].astype(float) + df["destroy_ms"].astype(float)
    order = (llm + sb).sort_values().index
    llm, sb, ok = llm.loc[order].values / 1000, sb.loc[order].values / 1000, (df["passed"].loc[order] == "true").values
    x = np.arange(len(llm))
    fig, ax = plt.subplots(figsize=(3.4, 1.9))
    ax.bar(x, llm, label="model inference", color=C_DEF)
    ax.bar(x, sb, bottom=llm, label="sandbox create+exec+destroy", color=C_BOXED)
    for xi, passed in zip(x, ok):
        if not passed:
            ax.text(xi, (llm + sb)[xi], "x", ha="center", va="bottom", fontsize=7, color="red")
    ax.set_xlabel("HumanEval task execution (sorted by total)"); ax.set_ylabel("wall-clock (s)")
    ax.legend(frameon=False, loc="upper left"); ax.grid(True, axis="y", alpha=0.3)
    fig.savefig(out / "agent_breakdown.pdf"); plt.close(fig)


def main():
    run, out = Path(sys.argv[1]), Path(sys.argv[2])
    out.mkdir(parents=True, exist_ok=True)
    if glob.glob(str(run / "r*/coldstart.csv")):
        plot_lifecycle(run, out)
    plot_throughput(run, out); plot_overhead(run, out); plot_agent(run, out)
    print(f"plots written to {out}", file=sys.stderr)


if __name__ == "__main__":
    main()
