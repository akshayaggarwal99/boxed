#!/usr/bin/env python3
"""Emit the five chart SVGs for the trade-article packet.

Every number here is copied from paper-v2/tables/numbers*.tex (Boxed) or
paper-judge/sections/05_results.tex (AMP). Style follows the dataviz skill's
reference palette: thin marks, 4px rounded data-end square at the baseline,
hairline solid grid, values at tips, text in ink tokens never in series color.
"""
from pathlib import Path

OUT = Path(__file__).parent

# Reference palette (light mode)
SURFACE = "#fcfcfb"
INK = "#0b0b0b"
INK2 = "#52514e"
MUTED = "#898781"
GRID = "#e1e0d9"
AXIS = "#c3c2b7"
BLUE = "#2a78d6"       # categorical slot 1
ORANGE = "#eb6834"     # categorical slot 2
BLUE_LIGHT = "#86b6ef"  # sequential step 250, the "before" shade for the dumbbell
DEEMPH = "#c3c2b7"     # de-emphasis gray for the emphasis form

FONT = 'font-family="Helvetica, Arial, sans-serif"'
W = 720  # canvas width; rendered at 2x for the doc


def esc(s):
    return s.replace("&", "&amp;").replace("<", "&lt;").replace(">", "&gt;")


def text(x, y, s, size=12, fill=INK, weight="normal", anchor="start", extra=""):
    return (f'<text x="{x:.1f}" y="{y:.1f}" {FONT} font-size="{size}" font-weight="{weight}" '
            f'fill="{fill}" text-anchor="{anchor}" {extra}>{esc(s)}</text>')


def hbar(x0, x1, y, h, fill, r=4):
    """Horizontal bar, square at the baseline (left), 4px rounded data-end (right)."""
    if x1 - x0 < 2 * r:
        return f'<rect x="{x0:.1f}" y="{y:.1f}" width="{x1-x0:.1f}" height="{h}" fill="{fill}"/>'
    return (f'<path d="M{x0:.1f},{y:.1f} H{x1-r:.1f} A{r},{r} 0 0 1 {x1:.1f},{y+r:.1f} '
            f'V{y+h-r:.1f} A{r},{r} 0 0 1 {x1-r:.1f},{y+h:.1f} H{x0:.1f} Z" fill="{fill}"/>')


def vbar(x, w, y0, y1, fill, r=4):
    """Vertical column, square at the baseline (bottom, y0), rounded top (y1)."""
    if y0 - y1 < 2 * r:
        return f'<rect x="{x:.1f}" y="{y1:.1f}" width="{w}" height="{y0-y1:.1f}" fill="{fill}"/>'
    return (f'<path d="M{x:.1f},{y0:.1f} V{y1+r:.1f} A{r},{r} 0 0 1 {x+r:.1f},{y1:.1f} '
            f'H{x+w-r:.1f} A{r},{r} 0 0 1 {x+w:.1f},{y1+r:.1f} V{y0:.1f} Z" fill="{fill}"/>')


def frame(height, title, subtitle, body, source=""):
    parts = [
        f'<svg xmlns="http://www.w3.org/2000/svg" width="{W}" height="{height}" viewBox="0 0 {W} {height}">',
        f'<rect width="{W}" height="{height}" fill="{SURFACE}"/>',
        text(32, 34, title, size=16, weight="600"),
        text(32, 54, subtitle, size=12, fill=INK2),
        body,
    ]
    if source:
        parts.append(text(32, height - 16, source, size=10, fill=MUTED))
    parts.append("</svg>")
    return "\n".join(parts)


# ---------------------------------------------------------------- 1b
def fig_1b():
    """Emphasis bars, two small multiples on one shared scale."""
    panels = [
        ("Laptop, colima VM", [("Raw Docker, stock defaults", 216, DEEMPH, False),
                               ("Raw Docker, hardened", 153, BLUE, True),
                               ("Hardened + control plane", 178, DEEMPH, False)]),
        ("GCE n2-standard-4, idle", [("Raw Docker, stock defaults", 368, DEEMPH, False),
                                     ("Raw Docker, hardened", 298, BLUE, True),
                                     ("Hardened + control plane", 347, DEEMPH, False)]),
    ]
    scale_max = 400
    label_w = 190
    panel_x = [32, 32 + 340]
    plot_w = 340 - label_w - 60
    top = 82
    row_h = 30
    bar_h = 20
    body = []
    for (px, (name, rows)) in zip(panel_x, panels):
        body.append(text(px, top - 8, name, size=12, weight="600", fill=INK2))
        x0 = px + label_w
        base_y = top + len(rows) * row_h + 6
        # hairline grid at 0 / 200 / 400
        for v in (0, 200, 400):
            gx = x0 + plot_w * v / scale_max
            body.append(f'<line x1="{gx:.1f}" y1="{top}" x2="{gx:.1f}" y2="{base_y}" stroke="{GRID}" stroke-width="1"/>')
            body.append(text(gx, base_y + 14, str(v), size=10, fill=MUTED, anchor="middle"))
        body.append(f'<line x1="{x0}" y1="{top}" x2="{x0}" y2="{base_y}" stroke="{AXIS}" stroke-width="1"/>')
        for i, (label, val, color, emph) in enumerate(rows):
            y = top + i * row_h + (row_h - bar_h) / 2
            x1 = x0 + plot_w * val / scale_max
            body.append(text(x0 - 8, y + bar_h / 2 + 4, label, size=11, fill=INK if emph else INK2, anchor="end"))
            body.append(hbar(x0, x1, y, bar_h, color))
            body.append(text(x1 + 6, y + bar_h / 2 + 4, f"{val} ms", size=11,
                             fill=INK, weight="600" if emph else "normal"))
    height = 206
    return frame(height,
                 "Hardening was faster than stock, on both hosts",
                 "Median create, exec, destroy lifecycle, ms, against the Docker Engine API",
                 "\n".join(body))


# ---------------------------------------------------------------- 2a
def fig_2a():
    """Single series columns; axis from zero so the flatness is honest."""
    data = [("4 vCPU", 12.3), ("8 vCPU", 13.6), ("16 vCPU", 15.2)]
    top, base_y = 80, 240
    x0, plot_w = 80, 560
    ymax = 20
    body = []
    for v in (0, 5, 10, 15, 20):
        gy = base_y - (base_y - top) * v / ymax
        body.append(f'<line x1="{x0}" y1="{gy:.1f}" x2="{x0+plot_w}" y2="{gy:.1f}" stroke="{GRID if v else AXIS}" stroke-width="1"/>')
        body.append(text(x0 - 10, gy + 4, str(v), size=10, fill=MUTED, anchor="end"))
    body.append(text(x0 - 10, top - 14, "sandboxes / s", size=10, fill=MUTED, anchor="end"))
    col_w = 24
    slot = plot_w / len(data)
    for i, (label, val) in enumerate(data):
        cx = x0 + slot * (i + 0.5)
        y1 = base_y - (base_y - top) * val / ymax
        body.append(vbar(cx - col_w / 2, col_w, base_y, y1, BLUE))
        body.append(text(cx, y1 - 8, f"{val}", size=12, weight="600", anchor="middle"))
        body.append(text(cx, base_y + 18, label, size=11, fill=INK2, anchor="middle"))
    height = 272
    return frame(height,
                 "Four times the cores bought 1.2 times the sandboxes per second",
                 "Peak create, destroy throughput, native Linux, runc, mean of 10 sweeps",
                 "\n".join(body))


# ---------------------------------------------------------------- 2b
def fig_2b():
    """One stacked bar, two segments, 2px surface gap, legend below."""
    model_s, sandbox_ms, total_s = 3.42, 453, 3.95
    model_share, sandbox_share = 85, 15
    x0, plot_w = 32, W - 64
    y, h = 86, 24
    gap = 2
    split = x0 + plot_w * model_share / 100
    body = []
    # left segment: square both ends. right segment: rounded data-end.
    body.append(f'<rect x="{x0}" y="{y}" width="{split - gap - x0:.1f}" height="{h}" fill="{BLUE}"/>')
    body.append(hbar(split, x0 + plot_w, y, h, ORANGE))
    body.append(text((x0 + split) / 2, y + h / 2 + 4, f"{model_share}%", size=12, weight="600", fill="#ffffff", anchor="middle"))
    body.append(text((split + x0 + plot_w) / 2, y + h / 2 + 4, f"{sandbox_share}%", size=12, weight="600", fill="#ffffff", anchor="middle"))
    # axis ticks 0 / total
    body.append(text(x0, y + h + 16, "0 s", size=10, fill=MUTED))
    body.append(text(x0 + plot_w, y + h + 16, f"{total_s} s end to end", size=10, fill=MUTED, anchor="end"))
    # legend
    ly = y + h + 44
    body.append(f'<rect x="{x0}" y="{ly-9}" width="12" height="12" rx="2" fill="{BLUE}"/>')
    body.append(text(x0 + 18, ly + 1, f"Model call (claude-opus-5, adaptive thinking), {model_s} s median", size=11, fill=INK))
    body.append(f'<rect x="{x0}" y="{ly+13}" width="12" height="12" rx="2" fill="{ORANGE}"/>')
    body.append(text(x0 + 18, ly + 23, f"Sandbox create, exec, destroy, {sandbox_ms} ms median", size=11, fill=INK))
    height = 204
    return frame(height,
                 "Where an agent step goes",
                 "Median wall-clock per HumanEval task, 60 executions, laptop host",
                 "\n".join(body))


# ---------------------------------------------------------------- 2c
def fig_2c():
    """Cumulative flags on Kata: single series columns."""
    data = [("Bare Kata boot,\nstock", 2.8), ("+ network none", 7.7), ("+ one-CPU quota", 10.7)]
    top, base_y = 80, 240
    x0, plot_w = 80, 560
    ymax = 12
    body = []
    for v in (0, 4, 8, 12):
        gy = base_y - (base_y - top) * v / ymax
        body.append(f'<line x1="{x0}" y1="{gy:.1f}" x2="{x0+plot_w}" y2="{gy:.1f}" stroke="{GRID if v else AXIS}" stroke-width="1"/>')
        body.append(text(x0 - 10, gy + 4, str(v), size=10, fill=MUTED, anchor="end"))
    body.append(text(x0 - 10, top - 14, "seconds", size=10, fill=MUTED, anchor="end"))
    col_w = 24
    slot = plot_w / len(data)
    for i, (label, val) in enumerate(data):
        cx = x0 + slot * (i + 0.5)
        y1 = base_y - (base_y - top) * val / ymax
        body.append(vbar(cx - col_w / 2, col_w, base_y, y1, BLUE))
        body.append(text(cx, y1 - 8, f"{val} s", size=12, weight="600", anchor="middle"))
        for j, line in enumerate(label.split("\n")):
            body.append(text(cx, base_y + 18 + j * 14, line, size=11, fill=INK2, anchor="middle"))
    height = 292
    return frame(height,
                 "Two flags that are free on runc cost seconds on a microVM",
                 "Kata Containers lifecycle with hardening flags added one at a time",
                 "\n".join(body))


# ---------------------------------------------------------------- 3b
def fig_3b():
    """Dumbbell: before -> after per row. One hue, two shades."""
    rows = [("All 600 rows", 0.056, 0.714),
            ("461 rows the rule never touched", 0.064, 0.064)]
    label_w = 230
    x0 = 32 + label_w
    plot_w = W - x0 - 60
    top = 92
    row_h = 44
    xmax = 0.8
    body = []
    base_y = top + len(rows) * row_h
    for v in (0, 0.2, 0.4, 0.6, 0.8):
        gx = x0 + plot_w * v / xmax
        body.append(f'<line x1="{gx:.1f}" y1="{top-10}" x2="{gx:.1f}" y2="{base_y}" stroke="{GRID if v else AXIS}" stroke-width="1"/>')
        body.append(text(gx, base_y + 14, f"{v:.1f}", size=10, fill=MUTED, anchor="middle"))
    body.append(text(x0 + plot_w, top - 18, "Cohen's kappa", size=10, fill=MUTED, anchor="end"))
    r = 6
    for i, (label, before, after) in enumerate(rows):
        cy = top + i * row_h + row_h / 2 - 4
        xb = x0 + plot_w * before / xmax
        xa = x0 + plot_w * after / xmax
        body.append(text(x0 - 12, cy + 4, label, size=11, fill=INK2, anchor="end"))
        if abs(xa - xb) > 1:
            body.append(f'<line x1="{xb:.1f}" y1="{cy}" x2="{xa:.1f}" y2="{cy}" stroke="{AXIS}" stroke-width="2" stroke-linecap="round"/>')
            body.append(f'<circle cx="{xb:.1f}" cy="{cy}" r="{r}" fill="{BLUE_LIGHT}" stroke="{SURFACE}" stroke-width="2"/>')
            body.append(f'<circle cx="{xa:.1f}" cy="{cy}" r="{r}" fill="{BLUE}" stroke="{SURFACE}" stroke-width="2"/>')
            body.append(text(xb, cy - 12, f"{before:.3f}", size=11, fill=INK, anchor="middle"))
            body.append(text(xa, cy - 12, f"{after:.3f}", size=11, fill=INK, weight="600", anchor="middle"))
        else:
            # before and after coincide: one dot, half light half dark would be noise; draw after over before with ring
            body.append(f'<circle cx="{xb:.1f}" cy="{cy}" r="{r+3}" fill="{BLUE_LIGHT}" stroke="{SURFACE}" stroke-width="2"/>')
            body.append(f'<circle cx="{xa:.1f}" cy="{cy}" r="{r}" fill="{BLUE}" stroke="{SURFACE}" stroke-width="2"/>')
            body.append(text(xa + 16, cy + 4, f"{before:.3f} before, {after:.3f} after. Unchanged.", size=11, fill=INK, weight="600"))
    # legend (2 shades = 2 series)
    ly = base_y + 40
    body.append(f'<circle cx="{32+6}" cy="{ly}" r="6" fill="{BLUE_LIGHT}"/>')
    body.append(text(32 + 18, ly + 4, "Before the refusal rule", size=11, fill=INK))
    body.append(f'<circle cx="{32+186}" cy="{ly}" r="6" fill="{BLUE}"/>')
    body.append(text(32 + 198, ly + 4, "After the refusal rule", size=11, fill=INK))
    height = 240
    return frame(height,
                 "The kappa gain lived entirely on the rows the rule wrote",
                 "Cohen's kappa between qwen3:8b and deepseek-r1:8b, 600 LoCoMo judgments",
                 "\n".join(body))


FIGS = {
    "fig-1b-hardening-faster.svg": fig_1b,
    "fig-2a-cores-vs-throughput.svg": fig_2a,
    "fig-2b-agent-step-split.svg": fig_2b,
    "fig-2c-kata-flags.svg": fig_2c,
    "fig-3b-kappa-dumbbell.svg": fig_3b,
}

if __name__ == "__main__":
    for name, fn in FIGS.items():
        (OUT / name).write_text(fn() + "\n")
        print("wrote", name)
