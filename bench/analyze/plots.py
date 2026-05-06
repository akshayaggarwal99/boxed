#!/usr/bin/env python3
"""Generate publication-quality PDF plots from bench CSVs.

Usage: plots.py <results_dir> <out_figures_dir>
"""
from __future__ import annotations

import os
import sys
from pathlib import Path

import matplotlib
matplotlib.use("Agg")
import matplotlib.pyplot as plt
import numpy as np
import pandas as pd

plt.rcParams.update({
    "font.family": "serif",
    "font.size": 9,
    "axes.labelsize": 9,
    "legend.fontsize": 8,
    "figure.dpi": 200,
    "savefig.bbox": "tight",
    "savefig.pad_inches": 0.02,
})


def cdf(ax, series, label):
    s = np.sort(series)
    y = np.arange(1, len(s) + 1) / len(s)
    ax.plot(s, y, label=label)


def plot_coldstart(results: Path, out: Path):
    df = pd.read_csv(results / "coldstart.csv")
    fig, ax = plt.subplots(figsize=(3.4, 2.2))
    cdf(ax, df["create_ms"], "create")
    cdf(ax, df["first_exec_ms"], "first exec")
    cdf(ax, df["total_ms"], "create+exec+destroy")
    ax.set_xscale("log")
    ax.set_xlabel("latency (ms, log)")
    ax.set_ylabel("CDF")
    ax.legend(loc="lower right", frameon=False)
    ax.grid(True, alpha=0.3)
    fig.savefig(out / "coldstart_cdf.pdf")
    plt.close(fig)


def plot_throughput(results: Path, out: Path):
    df = pd.read_csv(results / "throughput.csv")
    df = df.sort_values("conc")
    fig, ax = plt.subplots(figsize=(3.4, 2.2))
    ax.plot(df["conc"], df["rps"], marker="o")
    ax.set_xscale("log", base=2)
    ax.set_xlabel("concurrency")
    ax.set_ylabel("sandboxes / sec")
    ax.grid(True, alpha=0.3)
    fig.savefig(out / "throughput_curve.pdf")
    plt.close(fig)


def plot_overhead(results: Path, out: Path):
    df = pd.read_csv(results / "overhead.csv")
    fig, axes = plt.subplots(1, 2, figsize=(3.4, 2.0))
    axes[0].violinplot(df["rss_mib"], showmedians=True)
    axes[0].set_ylabel("RSS (MiB)")
    axes[0].set_xticks([])
    axes[1].violinplot(df["cpu_pct"], showmedians=True)
    axes[1].set_ylabel("CPU (%)")
    axes[1].set_xticks([])
    for a in axes:
        a.grid(True, alpha=0.3)
    fig.savefig(out / "overhead_violin.pdf")
    plt.close(fig)


def plot_agent(results: Path, out: Path):
    p = results / "agent_trace.csv"
    if not p.exists():
        return
    df = pd.read_csv(p, dtype={"passed": str})
    df = df[df["passed"].isin(["true", "True", "false", "False"])].copy()
    if df.empty:
        return
    llm = df["llm_ms"].astype(float)
    sandbox = (df["create_ms"].astype(float)
               + df["exec_ms"].astype(float)
               + df["destroy_ms"].astype(float))
    total = llm + sandbox
    order = total.sort_values().index
    llm = llm.loc[order].reset_index(drop=True)
    sandbox = sandbox.loc[order].reset_index(drop=True)
    x = np.arange(len(llm))
    fig, ax = plt.subplots(figsize=(3.4, 2.2))
    ax.bar(x, llm / 1000.0, label="LLM", color="#4a90e2")
    ax.bar(x, sandbox / 1000.0, bottom=llm / 1000.0,
           label="sandbox", color="#e2844a")
    ax.set_xlabel("task (sorted by total)")
    ax.set_ylabel("latency (s)")
    ax.legend(frameon=False, loc="upper left")
    ax.grid(True, axis="y", alpha=0.3)
    fig.savefig(out / "agent_breakdown.pdf")
    plt.close(fig)


def main():
    if len(sys.argv) != 3:
        print("usage: plots.py <results_dir> <out_figures_dir>", file=sys.stderr)
        sys.exit(2)
    results = Path(sys.argv[1])
    out = Path(sys.argv[2])
    out.mkdir(parents=True, exist_ok=True)
    if (results / "coldstart.csv").exists():
        plot_coldstart(results, out)
    if (results / "throughput.csv").exists():
        plot_throughput(results, out)
    if (results / "overhead.csv").exists():
        plot_overhead(results, out)
    plot_agent(results, out)
    print(f"plots written to {out}", file=sys.stderr)


if __name__ == "__main__":
    main()
