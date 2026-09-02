#!/usr/bin/env python3
"""Runtime-swap comparison: the same Boxed driver, control plane, agent, and
harness under three OCI runtimes on one host.

Usage: runtimes.py <results_dir> <out_tables_dir> <out_figures_dir> RUN1 RUN2 ...
  e.g. runtimes.py results ../paper-v2/tables ../paper-v2/figures \
         linux-runc-2026-09 linux-runsc-2026-09 linux-kata-2026-09

Emits tables/runtimes.tex (lifecycle + throughput + escape summary per
runtime), tables/runtimes-escapes.tex (12x3 verdict matrix), figures/runtimes.pdf,
and macros tables/numbers-runtimes.tex.
"""
from __future__ import annotations
import glob, sys
from pathlib import Path
import numpy as np, pandas as pd
import matplotlib; matplotlib.use("Agg"); import matplotlib.pyplot as plt
plt.rcParams.update({"font.family": "serif", "font.size": 8, "axes.labelsize": 8, "legend.fontsize": 7,
                     "figure.dpi": 200, "savefig.bbox": "tight", "savefig.pad_inches": 0.02})

LABEL = {"runc": "runc (container)", "runsc": "gVisor runsc", "kata": "Kata (QEMU microVM)",
         "kata-aware": "Kata, runtime-aware config", "runc-aware": "runc, runtime-aware config"}
def rtkey(run):  # linux-<key>-YYYY-MM -> <key>
    return "-".join(run.split("-")[1:-2])
RNG = np.random.default_rng(20260902)

def pooled(run, name):
    fs = sorted(glob.glob(str(run / f"r*/{name}")))
    return pd.concat([pd.read_csv(f) for f in fs], ignore_index=True) if fs else None

def ci_diff(a, b, n=10000):
    d = (np.median(RNG.choice(a, (n, len(a))), axis=1) - np.median(RNG.choice(b, (n, len(b))), axis=1))
    return np.percentile(d, 2.5), np.percentile(d, 97.5)

def main():
    res, tout, fout = Path(sys.argv[1]), Path(sys.argv[2]), Path(sys.argv[3])
    runs = sys.argv[4:]
    rows, esc, macros = [], {}, {}
    for r in runs:
        rt = rtkey(r)
        run = res / r
        boxed, hard, dflt = pooled(run, "coldstart.csv"), pooled(run, "baseline_hardened.csv"), pooled(run, "baseline_default.csv")
        if boxed is None: continue
        tps = sorted(glob.glob(str(run / "tp*/throughput.csv")))
        tp = pd.concat([pd.read_csv(p) for p in tps]) if tps else None
        peak = tp.groupby("conc")["rps"].mean().max() if tp is not None else float("nan")
        escs = sorted(glob.glob(str(run / "escapes_r*.csv")))
        E = pd.read_csv(escs[0]).set_index("test") if escs else None
        denied = int((E["postcondition"] == "DENIED").sum()) if E is not None else -1
        consistent = all((pd.read_csv(e).set_index("test")["postcondition"] == E["postcondition"]).all() for e in escs) if escs else False
        lo, hi = ci_diff(boxed["total_ms"].values, hard["total_ms"].values)
        row = dict(rt=rt, label=LABEL.get(rt, rt), boxed=np.median(boxed["total_ms"]), boxed_p95=np.percentile(boxed["total_ms"], 95),
                   hard=np.median(hard["total_ms"]), dflt=np.median(dflt["total_ms"]) if dflt is not None else float("nan"),
                   create=np.median(boxed["create_ms"]), exec=np.median(boxed["first_exec_ms"]), destroy=np.median(boxed["destroy_ms"]),
                   over=np.median(boxed["total_ms"]) - np.median(hard["total_ms"]), lo=lo, hi=hi, peak=peak, denied=denied,
                   n=len(boxed), runs=len(sorted(glob.glob(str(run / "r*/coldstart.csv")))), consistent=consistent)
        rows.append(row); esc[rt] = E
        P = "".join(w.capitalize() for w in rt.split("-"))
        macros[f"{P}BoxedMedian"] = f"{row['boxed']:.0f}\\,ms"; macros[f"{P}HardMedian"] = f"{row['hard']:.0f}\\,ms"
        macros[f"{P}Overhead"] = f"{row['over']:.0f}\\,ms"; macros[f"{P}OverheadCI"] = f"{lo:.0f}--{hi:.0f}\\,ms"
        macros[f"{P}Peak"] = f"{peak:.1f}"; macros[f"{P}Denied"] = str(denied)
        macros[f"{P}Create"] = f"{row['create']:.0f}\\,ms"
        macros[f"{P}CostShare"] = f"{100*row['over']/row['boxed']:.0f}\\%"
    fixed = {}
    for r in runs:
        rt = rtkey(r)
        fx = sorted(glob.glob(str(res / f"{r}-agentfix" / "escapes_r*.csv")))
        if fx:
            F = [pd.read_csv(f).set_index("test")["postcondition"] for f in fx]
            if all(f.equals(F[0]) for f in F):
                fixed[rt] = int((F[0] == "DENIED").sum())
    base = rows[0]
    for row in rows[1:]:
        P = "".join(w.capitalize() for w in row["rt"].split("-"))
        macros[f"{P}VsRuncBoxed"] = f"{row['boxed']/base['boxed']:.1f}$\\times$"
        macros[f"{P}VsRuncHard"] = f"{row['hard']/base['hard']:.1f}$\\times$"
    macros["RuntimeN"] = str(base["n"]); macros["RuntimeRuns"] = str(base["runs"])
    by = {r["rt"]: r for r in rows}
    if "kata" in by and "kata-aware" in by:
        macros["KataAwareSpeedup"] = f"{by['kata']['boxed']/by['kata-aware']['boxed']:.1f}$\\times$"

    # ---- summary table
    L = ["\\begin{table}[t]", "\\centering",
         "\\caption{One driver, three OCI runtimes on one host (GCE \\texttt{n2-standard-4} with nested virtualization). Medians in ms over "
         + ", ".join(f"$n{{=}}{r['n']}$ ({r['rt']})" for r in rows)
         + " sequential lifecycles; substrate cost is Boxed minus raw hardened Docker under the same runtime, with a bootstrap 95\\% CI. Peak is the best mean sandboxes/s over the sweeps. Denied is the post-condition escape verdict, identical across 3 probe runs for every runtime; $\\to$ gives the count after the agent fix of Section~\\ref{sec:runtimes}, also 3 runs. \\emph{+ cfg}: the microVM settings (internal ICC-off network, no CPU quota).}",
         "\\label{tab:runtimes}", "\\scriptsize", "\\setlength{\\tabcolsep}{2pt}", "\\begin{tabular}{@{}lrrrrrr@{}}", "\\toprule",
         "Runtime & stock & hardened & \\textbf{Boxed} & cost (95\\% CI) & peak/s & denied \\\\", "\\midrule"]
    SHORT = {"runc": "runc", "runsc": "gVisor", "kata": "Kata/QEMU", "kata-aware": "Kata + cfg", "runc-aware": "runc + cfg"}
    for r in rows:
        L.append(f"{SHORT.get(r['rt'], r['rt'])} & {r['dflt']:.0f} & {r['hard']:.0f} & \\textbf{{{r['boxed']:.0f}}} & "
                 f"{r['over']:.0f} ({r['lo']:.0f}--{r['hi']:.0f}) & {r['peak']:.1f} & {r['denied']}/12{'' if r['consistent'] else '$^{\\dagger}$'}"
                 + (f"$\\to${fixed[r['rt']]}" if r['rt'] in fixed and fixed[r['rt']] != r['denied'] else "") + " \\\\")
    L += ["\\bottomrule", "\\end{tabular}", "\\end{table}"]
    (tout / "runtimes.tex").write_text("\n".join(L) + "\n")

    # ---- verdict matrix
    vectors = list(esc[rows[0]["rt"]].index)
    M = ["\\begin{table}[t]", "\\centering", "\\caption{Escape-probe verdicts by runtime (post-condition method, run 1). D = denied, U = undenied, E = harness error.}",
         "\\label{tab:runtimes-escapes}", "\\scriptsize", "\\setlength{\\tabcolsep}{4pt}",
         "\\begin{tabular}{@{}l" + "c" * len(rows) + "l@{}}", "\\toprule",
         "Vector & " + " & ".join(r["rt"] for r in rows) + " & note \\\\", "\\midrule"]
    for v in vectors:
        cells = []
        for r in rows:
            e = esc[r["rt"]]
            cells.append({"DENIED": "D", "UNDENIED": "U"}.get(e.loc[v, "postcondition"], "E") if v in e.index else "--")
        note = ""
        # a short evidence note where runtimes differ
        vals = [esc[r["rt"]].loc[v, "postcondition"] if v in esc[r["rt"]].index else "" for r in rows]
        if len(set(vals)) > 1:
            note = "; ".join(f"{r['rt']}: {str(esc[r['rt']].loc[v,'evidence'])[:38]}" for r in rows if v in esc[r["rt"]].index)
            note = note.replace("_", "\\_").replace("%", "\\%").replace("&", "\\&")
        M.append(f"\\texttt{{{v.replace('_','\\_')}}} & " + " & ".join(cells) + f" & \\texttt{{{note}}} \\\\")
    M += ["\\bottomrule", "\\end{tabular}", "\\end{table}"]
    (tout / "runtimes-escapes.tex").write_text("\n".join(M) + "\n")

    # ---- figure: stacked phase bars per runtime, Boxed vs raw hardened
    fig, ax = plt.subplots(figsize=(3.4, 2.0))
    x = np.arange(len(rows)); w = 0.36
    for i, (key, lab, col) in enumerate([("hard", "raw Docker, hardened", "#e07b39"), ("boxed", "Boxed", "#1f5fbf")]):
        ax.bar(x + (i - 0.5) * w, [r[key] for r in rows], w, label=lab, color=col)
    for xi, r in zip(x, rows):
        ax.text(xi + 0.5 * w, r["boxed"], f"+{r['over']:.0f}", ha="center", va="bottom", fontsize=6)
    ax.set_xticks(x); ax.set_xticklabels([r["rt"] for r in rows]); ax.set_ylabel("median lifecycle (ms)")
    ax.legend(frameon=False, loc="upper left"); ax.grid(True, axis="y", alpha=0.3)
    fig.savefig(fout / "runtimes.pdf"); plt.close(fig)

    kns = [res / r / "knobs.csv" for r in runs if (res / r / "knobs.csv").exists()]
    kn = kns[0] if kns else None
    if kn is not None:
        k = pd.read_csv(kn, comment="#").groupby("flag")["ms"].median()
        for flag, name in (("stock", "Stock"), ("netnone", "NetNone"), ("cpus1", "CpusOne"), ("all", "All"), ("tmpfs", "Tmpfs"), ("pids", "Pids")):
            if flag in k: macros[f"KataKnob{name}"] = f"{k[flag]/1000:.1f}\\,s"
    for r in rows:
        tps = sorted(glob.glob(str(res / [x for x in runs if rtkey(x)==r['rt']][0] / "tp*/throughput.csv")))
        if tps:
            tp = pd.concat([pd.read_csv(p) for p in tps])
            PP = "".join(w.capitalize() for w in r['rt'].split("-"))
            macros[f"{PP}TPErrors"] = str(int(tp["errors"].sum())); macros[f"{PP}TPTotal"] = str(int(tp["n"].sum()))
            bad = tp[tp["errors"] > 0]["conc"].min()
            macros[f"{PP}TPFirstErrConc"] = str(int(bad)) if pd.notna(bad) else "none"
    # Post-fix probe runs, if present: <run>-agentfix/escapes_r*.csv
    for r in runs:
        rt = rtkey(r); P = "".join(w.capitalize() for w in rt.split("-"))
        fx = sorted(glob.glob(str(res / f"{r}-agentfix" / "escapes_r*.csv")))
        if not fx: continue
        F = [pd.read_csv(f).set_index("test") for f in fx]
        macros[f"{P}DeniedFixed"] = str(int((F[0]["postcondition"] == "DENIED").sum()))
        macros[f"{P}FixedRuns"] = str(len(F))
        macros[f"{P}FixedConsistent"] = "identical" if all(f["postcondition"].equals(F[0]["postcondition"]) for f in F) else "NOT identical"
        if "ptrace_agent" in F[0].index:
            ev = str(F[0].loc["ptrace_agent", "evidence"]).replace("_", "\\_").replace("%", "\\%")
            macros[f"{P}PtraceFixedVerdict"] = F[0].loc["ptrace_agent", "postcondition"].lower()
            macros[f"{P}PtraceFixedEvidence"] = ev
    lines = ["% AUTO-GENERATED by bench/analyze/runtimes.py -- do not edit by hand."]
    lines += [f"\\newcommand{{\\{k}}}{{{v}\\xspace}}" for k, v in macros.items()]
    (tout / "numbers-runtimes.tex").write_text("\n".join(lines) + "\n")
    print(f"runtimes: {[r['rt'] for r in rows]} -> tables/runtimes*.tex, figures/runtimes.pdf, {len(macros)} macros", file=sys.stderr)

if __name__ == "__main__":
    main()
