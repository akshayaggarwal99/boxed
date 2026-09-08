# Figures for the trade-article packet

Seven figures, two or three per article. Each ships as an `.svg` source and a
`@2x.png` render. **Insert the PNG into Google Docs**; Docs cannot import SVG.
Send publishers the SVG when they ask for vector, the PNG otherwise.

| Figure | Article | Form | File |
|---|---|---|---|
| 1a | 1 | Mechanism diagram: runc vs Kata, where the Yama protection lives | `fig-1a-two-kernels` |
| 1b | 1 | Emphasis bars, two small multiples, shared scale | `fig-1b-hardening-faster` |
| 2a | 2 | Columns, axis from zero | `fig-2a-cores-vs-throughput` |
| 2b | 2 | Single stacked bar, two segments | `fig-2b-agent-step-split` |
| 2c | 2 | Columns, cumulative flags | `fig-2c-kata-flags` |
| 3a | 3 | Pipeline diagram: the 600-row split | `fig-3a-six-hundred-row-split` |
| 3b | 3 | Dumbbell, before and after per row | `fig-3b-kappa-dumbbell` |

Insertion points are marked in each article file as a blockquote beginning
`[Figure Nx: ...]`, followed by the caption in italics. Paste the PNG where the
blockquote sits and keep the caption under it.

## Regenerating

```
python3 make_charts.py                       # rewrites the five chart SVGs
for f in fig-*.svg; do rsvg-convert --zoom 2 -o "${f%.svg}@2x.png" "$f"; done
```

The two diagrams (1a, 3a) are hand-authored SVG and are not touched by the
script. `rsvg-convert` is from librsvg (`brew install librsvg`).

## Provenance and style

Every number is copied from `paper-v2/tables/numbers*.tex` (Boxed) or
`paper-judge/sections/05_results.tex` (AMP); nothing is rounded for effect.

Colors are the dataviz skill's reference palette, validated with its script:
blue `#2a78d6` and orange `#eb6834` pass every check (adjacent CVD Delta E 24.7,
normal-vision 33.6, both above 3:1 on the surface). The de-emphasis gray for the
emphasis form is `#c3c2b7`; the dumbbell's "before" shade is sequential step 250,
`#86b6ef`. Ink is `#0b0b0b` / `#52514e` / `#898781`, grid `#e1e0d9`, surface
`#fcfcfb`. Marks are 20 to 24 px thick with a 4 px rounded data-end square at the
baseline, gridlines are solid hairlines, values sit at bar tips, and text never
wears a series color. Typeface is Helvetica, pinned so the renderer does not fall
back to Verdana.
