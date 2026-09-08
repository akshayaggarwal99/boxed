#!/usr/bin/env python3
"""Render an article's markdown as styled HTML for pasting into Google Docs.

Docs keeps fonts, sizes, bold/italic, table borders and shading, and embedded
images when rich HTML is pasted. This converter covers exactly the constructs
the three articles use: one H1, H2s, paragraphs with **bold** *italic* `code`,
bare URLs, bullet and numbered lists, pipe tables, and the figure callouts
(a blockquote whose first line names the PNG and whose second line is the
caption). The Source/Length/Audience/Status/Byline lines become a two-column
metadata table instead of one run-on paragraph.

Usage: build_doc_html.py article-1-hardening-yama-scoring.md > out.html
"""
import base64
import html
import re
import sys
from pathlib import Path

HERE = Path(__file__).parent

# Fonts are all in Google Docs' default font list, so they survive the paste.
BODY = "Lora, Georgia, serif"
HEAD = "Roboto, Arial, sans-serif"
MONO = "'Roboto Mono', 'Courier New', monospace"
INK, INK2, MUTED = "#1f1f1f", "#444444", "#6b6b6b"
RULE, SHADE = "#d6d6d6", "#f1f3f4"

META_KEYS = ("Source", "Length", "Audience", "Status", "Byline")


def esc(s):
    return html.escape(s, quote=False)


def inline(s):
    """bold, italic, code, bare URLs -> HTML. Code first so its contents are left alone."""
    out, i = [], 0
    for m in re.finditer(r"`([^`]+)`", s):
        out.append(_inline_text(s[i:m.start()]))
        out.append(f'<span style="font-family:{MONO};font-size:10pt;background:{SHADE};padding:0 2px;text-decoration:none">{esc(m.group(1))}</span>')
        i = m.end()
    out.append(_inline_text(s[i:]))
    return "".join(out)


def _inline_text(s):
    s = esc(s)
    s = re.sub(r"\*\*(.+?)\*\*", r"<b>\1</b>", s)
    s = re.sub(r"(?<![\w*])\*(?!\s)(.+?)(?<!\s)\*(?![\w*])", r"<i>\1</i>", s)
    s = re.sub(r"(https?://[^\s<]+?)([.,;)]?)(?=\s|$)", r'<a href="\1" style="color:#1a56b0;text-decoration:underline">\1</a>\2', s)
    return s


def p(s, size="11pt", color=INK, extra=""):
    return f'<p style="font-family:{BODY};font-size:{size};line-height:1.5;color:{color};text-decoration:none;margin:0 0 10pt 0;{extra}">{s}</p>'


def h1(s):
    return f'<h1 style="font-family:{HEAD};font-size:22pt;font-weight:700;line-height:1.25;color:{INK};text-decoration:none;margin:0 0 14pt 0">{inline(s)}</h1>'


def h2(s):
    return f'<h2 style="font-family:{HEAD};font-size:14pt;font-weight:700;color:{INK};text-decoration:none;margin:22pt 0 8pt 0">{inline(s)}</h2>'


def meta_table(rows):
    tr = []
    for k, v in rows:
        tr.append(
            f'<tr>'
            f'<td style="font-family:{HEAD};font-size:9.5pt;font-weight:700;color:{INK2};background:{SHADE};'
            f'border:1px solid {RULE};padding:6px 10px;width:110px;vertical-align:top;text-decoration:none">{esc(k)}</td>'
            f'<td style="font-family:{BODY};font-size:10pt;color:{INK};border:1px solid {RULE};padding:6px 10px;'
            f'vertical-align:top;text-decoration:none">{inline(v)}</td></tr>'
        )
    return ('<table cellspacing="0" cellpadding="0" style="border-collapse:collapse;width:100%;margin:0 0 16pt 0">'
            + "".join(tr) + "</table>")


def data_table(header, rows):
    th = "".join(
        f'<th style="font-family:{HEAD};font-size:10pt;font-weight:700;color:{INK};background:{SHADE};'
        f'border:1px solid {RULE};padding:6px 10px;text-align:left;text-decoration:none">{inline(c)}</th>' for c in header)
    body = []
    for r in rows:
        tds = "".join(
            f'<td style="font-family:{BODY};font-size:10.5pt;color:{INK};border:1px solid {RULE};'
            f'padding:6px 10px;vertical-align:top;text-decoration:none">{inline(c)}</td>' for c in r)
        body.append(f"<tr>{tds}</tr>")
    return ('<table cellspacing="0" cellpadding="0" style="border-collapse:collapse;width:100%;margin:6pt 0 14pt 0">'
            f'<thead><tr>{th}</tr></thead><tbody>{"".join(body)}</tbody></table>')


def figure(png_rel, caption):
    path = HERE / png_rel
    b64 = base64.b64encode(path.read_bytes()).decode()
    img = f'<img src="data:image/png;base64,{b64}" width="624" style="width:624px;height:auto;display:block;margin:6pt 0 6pt 0">'
    cap = p(f"<i>{inline(caption)}</i>", size="9.5pt", color=MUTED, extra="margin:0 0 16pt 0")
    return img + cap


def lists(kind, items):
    tag = "ol" if kind == "ol" else "ul"
    li = "".join(f'<li style="font-family:{BODY};font-size:11pt;line-height:1.5;color:{INK};text-decoration:none;margin:0 0 4pt 0">{inline(x)}</li>' for x in items)
    return f'<{tag} style="margin:0 0 12pt 22pt;padding:0">{li}</{tag}>'


def split_row(line):
    return [c.strip() for c in line.strip().strip("|").split("|")]


def convert(md_text):
    lines = md_text.split("\n")
    out, meta, title_opts = [], [], []
    i, n = 0, len(lines)
    saw_rule = False
    while i < n:
        line = lines[i]
        s = line.strip()

        if not s:
            i += 1
            continue

        if s.startswith("# "):
            title = re.sub(r"^Article \d+:\s*", "", s[2:])
            out.append(h1(title))
            i += 1
            continue

        m = re.match(r"^\*\*(\w+):\*\*\s*(.*)$", s)
        if m and m.group(1) in META_KEYS and not saw_rule:
            meta.append((m.group(1), m.group(2)))
            i += 1
            continue

        if s == "---":
            # end of front matter: flush the metadata table and title options
            if meta:
                out.append(meta_table(meta))
                meta = []
            if title_opts:
                out.append(h2("Title options"))
                out.append(lists("ol", title_opts))
                title_opts = []
            saw_rule = True
            out.append(f'<hr style="border:0;border-top:1px solid {RULE};margin:8pt 0 16pt 0">')
            i += 1
            continue

        if s.startswith("## "):
            if s[3:].strip().lower() == "title options" and not saw_rule:
                # collect the numbered options that follow
                i += 1
                while i < n and (not lines[i].strip() or re.match(r"^\d+\.\s", lines[i].strip())):
                    if lines[i].strip():
                        title_opts.append(re.sub(r"^\d+\.\s", "", lines[i].strip()))
                    i += 1
                continue
            out.append(h2(s[3:]))
            i += 1
            continue

        if s.startswith("> "):
            block = []
            while i < n and lines[i].strip().startswith(">"):
                block.append(lines[i].strip()[1:].strip())
                i += 1
            head = block[0] if block else ""
            m = re.search(r"\[Figure [^:]+:\s*`([^`]+)`\]", head)
            if m:
                cap = " ".join(b for b in block[1:] if b).strip("*")
                out.append(figure(m.group(1), cap))
            else:
                out.append(p(inline(" ".join(block)), color=INK2, extra=f"border-left:3px solid {RULE};padding-left:10px"))
            continue

        if s.startswith("|"):
            rows = []
            while i < n and lines[i].strip().startswith("|"):
                rows.append(split_row(lines[i]))
                i += 1
            header, body = rows[0], [r for r in rows[1:] if not all(set(c) <= set("-: ") for c in r)]
            out.append(data_table(header, body))
            continue

        if re.match(r"^[-*]\s", s):
            items = []
            while i < n and re.match(r"^[-*]\s", lines[i].strip()):
                items.append(lines[i].strip()[2:])
                i += 1
            out.append(lists("ul", items))
            continue

        if re.match(r"^\d+\.\s", s):
            items = []
            while i < n and re.match(r"^\d+\.\s", lines[i].strip()):
                items.append(re.sub(r"^\d+\.\s", "", lines[i].strip()))
                i += 1
            out.append(lists("ol", items))
            continue

        # paragraph: consecutive non-blank lines
        para = [s]
        i += 1
        while i < n and lines[i].strip() and not re.match(r"^(#|\||>|[-*]\s|\d+\.\s|---)", lines[i].strip()):
            para.append(lines[i].strip())
            i += 1
        out.append(p(inline(" ".join(para))))

    if meta:
        out.append(meta_table(meta))
    return '<div style="font-family:' + BODY + '">' + "\n".join(out) + "</div>"


if __name__ == "__main__":
    src = Path(sys.argv[1])
    sys.stdout.write(convert(src.read_text()))
