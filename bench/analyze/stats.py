#!/usr/bin/env python3
"""Emit LaTeX macros and data tables from a benchmark campaign.

Usage: stats.py <results/RUN> <out_tables_dir> [<results/LEGACY>]

Layout of <results/RUN> (written by run-all.sh):
  r<k>/coldstart.csv, r<k>/baseline_hardened.csv, r<k>/baseline_default.csv
  tp<k>/throughput.csv
  overhead.csv
  escapes_r<k>.csv
  agent_trace.csv (+ agent_trace.jsonl)

Produces in <out_tables_dir>:
  numbers.tex      \newcommand macros, every number the paper quotes
  throughput.tex   throughput table generated from tp*/throughput.csv
  escapes.tex      escape-probe table generated from escapes_r*.csv
  lifecycle.tex    Boxed vs raw-Docker lifecycle breakdown table

Every macro is derived here; nothing is hand-typed into the manuscript.
"""
from __future__ import annotations

import glob
import sys
from pathlib import Path

import numpy as np
import pandas as pd
from scipy import stats as sps

RNG = np.random.default_rng(20260901)


def fmt_ms(x: float) -> str:
    if x >= 1000:
        return f"{x/1000:.2f}\\,s"
    return f"{x:.0f}\\,ms"


def boot_median_ci(x, n=10000):
    x = np.asarray(x, dtype=float)
    meds = np.median(RNG.choice(x, size=(n, len(x)), replace=True), axis=1)
    return np.percentile(meds, 2.5), np.percentile(meds, 97.5)


def pooled(paths, col="total_ms"):
    frames = [pd.read_csv(p) for p in paths]
    return frames, pd.concat(frames, ignore_index=True)


def latency_block(prefix, paths, m):
    frames, df = pooled(paths)
    tot = df["total_ms"]
    lo, hi = boot_median_ci(tot)
    run_meds = [np.median(f["total_ms"]) for f in frames]
    m[f"{prefix}Runs"] = str(len(frames))
    m[f"{prefix}N"] = str(len(frames[0]))
    m[f"{prefix}Total"] = str(len(df))
    m[f"{prefix}Median"] = fmt_ms(np.median(tot))
    m[f"{prefix}MedianCI"] = f"{lo:.0f}--{hi:.0f}\\,ms"
    m[f"{prefix}RunMedianMin"] = fmt_ms(min(run_meds))
    m[f"{prefix}RunMedianMax"] = fmt_ms(max(run_meds))
    m[f"{prefix}IQR"] = fmt_ms(np.percentile(tot, 75) - np.percentile(tot, 25))
    m[f"{prefix}PNinetyFive"] = fmt_ms(np.percentile(tot, 95))
    m[f"{prefix}PNinetyNine"] = fmt_ms(np.percentile(tot, 99))
    m[f"{prefix}CreateMedian"] = fmt_ms(np.median(df["create_ms"]))
    m[f"{prefix}ExecMedian"] = fmt_ms(np.median(df["first_exec_ms"]))
    m[f"{prefix}DestroyMedian"] = fmt_ms(np.median(df["destroy_ms"]))
    if "exit_code" in df:
        m[f"{prefix}NonZeroExit"] = str(int((df["exit_code"] != 0).sum()))
    # Tail clustering: how many of the samples above p95 fall in a single run.
    p95 = np.percentile(tot, 95)
    slow = df[tot > p95]
    per_run = [int((f["total_ms"] > p95).sum()) for f in frames]
    m[f"{prefix}TailN"] = str(len(slow))
    m[f"{prefix}TailClusterMax"] = str(max(per_run))
    m[f"{prefix}TailCreateShare"] = f"{100*(slow['create_ms'] > slow['first_exec_ms']).mean():.0f}\\%"
    return df


def main():
    args = [a for a in sys.argv[1:]]
    prefix = ""
    if "--prefix" in args:
        i = args.index("--prefix"); prefix = args[i + 1]; del args[i:i + 2]
    if len(args) < 2:
        print(__doc__, file=sys.stderr)
        sys.exit(2)
    run = Path(args[0])
    out = Path(args[1])
    legacy = Path(args[2]) if len(args) > 2 else None
    out.mkdir(parents=True, exist_ok=True)
    suffix = f"-{prefix.lower()}" if prefix else ""
    m: dict[str, str] = {}

    # ---------------------------------------------------------------- cold start
    boxed = latency_block("BoxedCold", sorted(glob.glob(str(run / "r*/coldstart.csv"))), m)
    hard = latency_block("BaseHard", sorted(glob.glob(str(run / "r*/baseline_hardened.csv"))), m)
    dflt = latency_block("BaseDef", sorted(glob.glob(str(run / "r*/baseline_default.csv"))), m)

    bm, hm, dm = (np.median(boxed["total_ms"]), np.median(hard["total_ms"]), np.median(dflt["total_ms"]))
    m["BoxedOverheadMedian"] = fmt_ms(bm - hm)
    m["BoxedOverheadPct"] = f"{100*(bm-hm)/hm:.0f}\\%"
    m["HardeningDeltaMedian"] = fmt_ms(abs(hm - dm))
    m["HardeningDeltaSign"] = "faster" if hm < dm else "slower"
    for phase, key in (("create_ms", "Create"), ("first_exec_ms", "Exec"), ("destroy_ms", "Destroy")):
        m[f"BoxedOverhead{key}"] = fmt_ms(np.median(boxed[phase]) - np.median(hard[phase]))
    # Bootstrap CI on the difference of medians (independent resampling).
    diffs = (np.median(RNG.choice(boxed["total_ms"].values, size=(10000, len(boxed)), replace=True), axis=1)
             - np.median(RNG.choice(hard["total_ms"].values, size=(10000, len(hard)), replace=True), axis=1))
    m["BoxedOverheadCI"] = f"{np.percentile(diffs,2.5):.0f}--{np.percentile(diffs,97.5):.0f}\\,ms"

    # lifecycle table
    rows = [("Raw Docker, stock defaults", dflt), ("Raw Docker, Boxed's hardened config", hard), ("Boxed (control plane + agent)", boxed)]
    L = ["\\begin{table}[t]", "\\centering",
         "\\caption{Lifecycle latency, medians over " + m["BoxedColdRuns"] + " runs $\\times$ " + m["BoxedColdN"] +
         " sequential create$\\rightarrow$exec$\\rightarrow$destroy cycles, same host, image, and command. Raw Docker rows call the Engine API directly with no control plane and no agent.}",
         f"\\label{{tab:lifecycle{suffix}}}", "\\footnotesize", "\\begin{tabular}{@{}lrrrrr@{}}", "\\toprule",
         "Configuration & create & exec & destroy & total & p95 \\\\", "\\midrule"]
    for name, df in rows:
        L.append(f"{name} & {np.median(df['create_ms']):.0f} & {np.median(df['first_exec_ms']):.0f} & "
                 f"{np.median(df['destroy_ms']):.0f} & \\textbf{{{np.median(df['total_ms']):.0f}}} & {np.percentile(df['total_ms'],95):.0f} \\\\")
    L += ["\\bottomrule", "\\end{tabular}", "\\\\[2pt]{\\footnotesize All values in ms.}", "\\end{table}"]
    (out / f"lifecycle{suffix}.tex").write_text("\n".join(L) + "\n")

    # ---------------------------------------------------------------- throughput
    tps = sorted(glob.glob(str(run / "tp*/throughput.csv")))
    if tps:
        tp = pd.concat([pd.read_csv(p).assign(run=i) for i, p in enumerate(tps)], ignore_index=True)
        g = tp.groupby("conc")["rps"]
        summ = pd.DataFrame({"mean": g.mean(), "sd": g.std(ddof=1), "min": g.min(), "max": g.max(), "n": g.count()})
        summ["ci"] = sps.t.ppf(0.975, summ["n"] - 1) * summ["sd"] / np.sqrt(summ["n"])
        best = summ["mean"].idxmax()
        m["BoxedTPRuns"] = str(len(tps))
        m["BoxedTPPerLevel"] = str(int(tp["n"].iloc[0]))
        m["BoxedPeakRPS"] = f"{summ.loc[best,'mean']:.1f}"
        m["BoxedPeakConc"] = str(int(best))
        m["BoxedPeakCI"] = f"$\\pm${summ.loc[best,'ci']:.1f}"
        m["BoxedTPatOne"] = f"{summ.loc[1,'mean']:.1f}"
        m["BoxedTPatThirtyTwo"] = f"{summ.loc[32,'mean']:.1f}" if 32 in summ.index else "n/a"
        m["BoxedTPErrors"] = str(int(tp["errors"].sum()))
        T = ["\\begin{table}[t]", "\\centering",
             "\\caption{Throughput sweep: " + m["BoxedTPPerLevel"] + " create$\\rightarrow$destroy lifecycles per level, " +
             m["BoxedTPRuns"] + " independent sweeps. Mean sandboxes/s with 95\\% confidence interval (Student's $t$) and the range across sweeps. " +
             m["BoxedTPErrors"] + " errors in " + str(int(tp["n"].sum())) + " lifecycles.}",
             f"\\label{{tab:throughput{suffix}}}", "\\footnotesize", "\\begin{tabular}{@{}rrrr@{}}", "\\toprule",
             "Concurrency $c$ & Sandboxes/s (mean) & 95\\% CI & Range \\\\", "\\midrule"]
        for c, r in summ.iterrows():
            b = "\\textbf" if c == best else ""
            T.append(f"{int(c)} & {b}{{{r['mean']:.2f}}} & $\\pm${r['ci']:.2f} & {r['min']:.2f}--{r['max']:.2f} \\\\")
        T += ["\\bottomrule", "\\end{tabular}", "\\end{table}"]
        (out / f"throughput{suffix}.tex").write_text("\n".join(T) + "\n")
        summ.to_csv(out / f"throughput_summary{suffix}.csv")

    # ---------------------------------------------------------------- overhead
    ov = run / "overhead.csv"
    if ov.exists():
        df = pd.read_csv(ov)
        m["BoxedOverheadN"] = str(len(df))
        m["BoxedRSSMedian"] = f"{np.median(df['rss_mib']):.2f}\\,MiB"
        m["BoxedRSSPNinetyFive"] = f"{np.percentile(df['rss_mib'],95):.2f}\\,MiB"
        m["BoxedRSSMax"] = f"{df['rss_mib'].max():.2f}\\,MiB"
        m["BoxedCPUMedian"] = f"{np.median(df['cpu_pct']):.2f}\\%"

    # ---------------------------------------------------------------- escapes
    esc = sorted(glob.glob(str(run / "escapes_r*.csv")))
    if esc:
        E = pd.concat([pd.read_csv(p).assign(run=i + 1) for i, p in enumerate(esc)], ignore_index=True)
        first = E[E["run"] == 1].set_index("test")
        consistent = E.groupby("test")["postcondition"].nunique().max() == 1
        m["BoxedEscapeRuns"] = str(len(esc))
        m["BoxedEscapeTotal"] = str(len(first))
        m["BoxedEscapeDenied"] = str(int((first["postcondition"] == "DENIED").sum()))
        m["BoxedEscapeUndenied"] = str(int((first["postcondition"] == "UNDENIED").sum()))
        m["BoxedEscapeError"] = str(int((first["postcondition"] == "ERROR").sum()))
        m["BoxedEscapePct"] = f"{100*(first['postcondition']=='DENIED').sum()/len(first):.0f}\\%"
        m["BoxedEscapeSigDenied"] = str(int((first["signature"] == "DENIED").sum()))
        m["BoxedEscapeSigDisagree"] = str(int((first["signature"] != first["postcondition"]).sum()))
        m["BoxedEscapeConsistent"] = "identical" if consistent else "NOT identical"
        S = ["\\begin{table*}[t]", "\\centering",
             "\\caption{Escape probe on the hardened Docker driver, " + m["BoxedEscapeTotal"] + " vectors, " + m["BoxedEscapeRuns"] +
             " independent runs (verdicts " + m["BoxedEscapeConsistent"] + " across runs). \\emph{Signature} is signature matching (grep for a denial string in captured output); "
             "\\emph{Post-condition} checks state after the attempt, on the host where possible. The paper reports the post-condition column.}",
             f"\\label{{tab:escape{suffix}}}", "\\scriptsize", "\\setlength{\\tabcolsep}{4pt}",
             "\\begin{tabular}{@{}lllllp{0.42\\textwidth}@{}}", "\\toprule",
             "Vector & Threat & Exit & Signature & Post-condition & Evidence \\\\", "\\midrule"]
        for t, r in first.iterrows():
            ev = str(r["evidence"]).replace("_", "\\_").replace("%", "\\%").replace("&", "\\&").replace("/", "/\\allowbreak ")
            ex = str(r["attempt_exit"]).replace("-1", "$-$1")
            thr = str(r["threat"]).replace(";", ", ")
            S.append(f"\\texttt{{{t.replace('_','\\_')}}} & {thr} & {ex} & {r['signature'].lower()} & \\textbf{{{r['postcondition'].lower()}}} & \\texttt{{{ev}}} \\\\")
        S += ["\\bottomrule", "\\end{tabular}", "\\end{table*}"]
        (out / f"escapes{suffix}.tex").write_text("\n".join(S) + "\n")

    # ---------------------------------------------------------------- agent trace
    ag = run / "agent_trace.csv"
    if ag.exists():
        reps = [ag] + sorted(glob.glob(str(run / "agent_rep*/agent_trace.csv")))
        frames = [pd.read_csv(p, dtype={"passed": str}) for p in reps]
        df = pd.concat(frames, ignore_index=True)
        valid = df[df["passed"].isin(["true", "false"])].copy()
        m["BoxedAgentReps"] = str(len(reps))
        m["BoxedAgentTasksPerRep"] = str(len(frames[0]))
        m["BoxedAgentN"] = str(len(valid))
        m["BoxedAgentErrors"] = str(len(df) - len(valid))
        def rep_stat(fn):
            vals = [fn(f[f["passed"].isin(["true", "false"])]) for f in frames]
            return min(vals), max(vals)
        lo, hi = rep_stat(lambda f: np.median(f["create_ms"].astype(float)))
        m["BoxedAgentCreateRepRange"] = f"{lo:.0f}--{hi:.0f}\\,ms"
        lo, hi = rep_stat(lambda f: np.median(f["create_ms"].astype(float) + f["exec_ms"].astype(float) + f["destroy_ms"].astype(float)))
        m["BoxedAgentSandboxRepRange"] = f"{lo:.0f}--{hi:.0f}\\,ms"
        lo, hi = rep_stat(lambda f: 100 * f["llm_ms"].astype(float).sum() / (f["llm_ms"].astype(float) + f["create_ms"].astype(float) + f["exec_ms"].astype(float) + f["destroy_ms"].astype(float)).sum())
        m["BoxedAgentLLMShareRepRange"] = f"{lo:.0f}--{hi:.0f}\\%"
        if len(valid):
            p = valid["passed"] == "true"
            llm = valid["llm_ms"].astype(float); cr = valid["create_ms"].astype(float)
            ex = valid["exec_ms"].astype(float); de = valid["destroy_ms"].astype(float)
            tot = llm + cr + ex + de; sb = cr + ex + de
            m["BoxedAgentModel"] = str(valid["model"].iloc[0]).replace("_", "\\_")
            m["BoxedAgentPassCount"] = str(int(p.sum()))
            m["BoxedAgentPass"] = f"{100*p.sum()/len(valid):.0f}\\%"
            m["BoxedAgentLLMShare"] = f"{100*llm.sum()/tot.sum():.0f}\\%"
            m["BoxedAgentSandboxShare"] = f"{100*sb.sum()/tot.sum():.0f}\\%"
            m["BoxedAgentMedianTotal"] = fmt_ms(np.median(tot))
            m["BoxedAgentLLMMedian"] = fmt_ms(np.median(llm))
            m["BoxedAgentSandboxMedian"] = fmt_ms(np.median(sb))
            m["BoxedAgentCreateMedian"] = fmt_ms(np.median(cr))
            m["BoxedAgentExecMedian"] = fmt_ms(np.median(ex))
            m["BoxedAgentOutTokMedian"] = f"{np.median(valid['output_tokens'].astype(float)):.0f}"
            m["BoxedAgentStopReasons"] = ", ".join(f"{k}: {v}" for k, v in valid["stop_reason"].value_counts().items()).replace("_", "\\_")

    # ---------------------------------------------------------------- legacy
    if legacy and (legacy / "coldstart.csv").exists():
        l = pd.read_csv(legacy / "coldstart.csv")
        m["LegacyColdMedian"] = fmt_ms(np.median(l["total_ms"]))
        m["LegacyColdN"] = str(len(l))
        if (legacy / "escapes.csv").exists():
            le = pd.read_csv(legacy / "escapes.csv")
            m["LegacyEscapeDenied"] = str(int((le["result"] == "DENIED").sum()))
        if (legacy / "throughput.csv").exists():
            lt = pd.read_csv(legacy / "throughput.csv")
            m["LegacyPeakRPS"] = f"{lt['rps'].max():.1f}"

    lines = ["% AUTO-GENERATED by bench/analyze/stats.py -- do not edit by hand.",
             f"% source: {run}"]
    for k, v in m.items():
        lines.append(f"\\newcommand{{\\{prefix}{k}}}{{{v}\\xspace}}")
    (out / f"numbers{suffix}.tex").write_text("\n".join(lines) + "\n")
    print(f"wrote {out}/numbers{suffix}.tex ({len(m)} macros) + lifecycle/throughput/escapes tables", file=sys.stderr)


if __name__ == "__main__":
    main()
