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

LABEL = {"runc": "runc (container)", "runsc": "gVisor runsc", "kata": "Kata (QEMU microVM)"}
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
        rt = r.split("-")[1]
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
        P = rt.capitalize()
        macros[f"{P}BoxedMedian"] = f"{row['boxed']:.0f}\\,ms"; macros[f"{P}HardMedian"] = f"{row['hard']:.0f}\\,ms"
        macros[f"{P}Overhead"] = f"{row['over']:.0f}\\,ms"; macros[f"{P}OverheadCI"] = f"{lo:.0f}--{hi:.0f}\\,ms"
        macros[f"{P}Peak"] = f"{peak:.1f}"; macros[f"{P}Denied"] = str(denied)
        macros[f"{P}Create"] = f"{row['create']:.0f}\\,ms"
    base = rows[0]
    for row in rows[1:]:
        P = row["rt"].capitalize()
        macros[f"{P}VsRuncBoxed"] = f"{row['boxed']/base['boxed']:.1f}$\\times$"
        macros[f"{P}VsRuncHard"] = f"{row['hard']/base['hard']:.1f}$\\times$"
    macros["RuntimeN"] = str(base["n"]); macros["RuntimeRuns"] = str(base["runs"])

    # ---- summary table
    L = ["\\begin{table}[t]", "\\centering",
         "\\caption{One driver, three OCI runtimes on one host (GCE \\texttt{n2-standard-4}, nested virtualization). Medians over "
         f"{base['runs']} runs $\\times$ {base['n']//base['runs']} lifecycles; the substrate cost is Boxed minus raw hardened Docker under the same runtime, with a bootstrap 95\\% CI. Peak throughput is the best mean over 3 sweeps. Escape verdicts are post-condition, identical across 3 runs where marked.}}",
         "\\label{tab:runtimes}", "\\scriptsize", "\\setlength{\\tabcolsep}{3.5pt}", "\\begin{tabular}{@{}lrrrrrrr@{}}", "\\toprule",
         "Runtime & raw stock & raw hard. & \\textbf{Boxed} & p95 & substrate cost & peak/s & denied \\\\", "\\midrule"]
    for r in rows:
        L.append(f"{r['label']} & {r['dflt']:.0f} & {r['hard']:.0f} & \\textbf{{{r['boxed']:.0f}}} & {r['boxed_p95']:.0f} & "
                 f"{r['over']:.0f} ({r['lo']:.0f}--{r['hi']:.0f}) & {r['peak']:.1f} & {r['denied']}/12{'' if r['consistent'] else '$^{\\dagger}$'} \\\\")
    L += ["\\bottomrule", "\\end{tabular}", "\\\\[2pt]{\\scriptsize Latencies in ms. $^{\\dagger}$verdicts differed between runs; see text.}", "\\end{table}"]
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

    lines = ["% AUTO-GENERATED by bench/analyze/runtimes.py -- do not edit by hand."]
    lines += [f"\\newcommand{{\\{k}}}{{{v}\\xspace}}" for k, v in macros.items()]
    (tout / "numbers-runtimes.tex").write_text("\n".join(lines) + "\n")
    print(f"runtimes: {[r['rt'] for r in rows]} -> tables/runtimes*.tex, figures/runtimes.pdf, {len(macros)} macros", file=sys.stderr)

if __name__ == "__main__":
    main()
