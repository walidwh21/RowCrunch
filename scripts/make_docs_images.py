#!/usr/bin/env python3
"""Renders the images used by README.md into docs/img/.

  architecture.png    pipeline overview
  chunking.png        how the file is cut into line-aligned ranges
  term-input.png      terminal capture: the input format (real testdata)
  term-report.png     terminal capture: the report format (real testdata)
  term-layout.png     terminal capture: the project tree (real file listing)

Requires matplotlib and Pillow.  Run from anywhere:  python3 scripts/make_docs_images.py
"""
import os

import matplotlib

matplotlib.use("Agg")
import matplotlib.pyplot as plt
from matplotlib.patches import FancyArrowPatch, FancyBboxPatch, Rectangle
from PIL import Image, ImageDraw, ImageFont

ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), ".."))
IMG = os.path.join(ROOT, "docs", "img")
os.makedirs(IMG, exist_ok=True)

INK = "#1f2933"
BLUE, BLUE_BG = "#2563eb", "#dbeafe"
GREEN, GREEN_BG = "#15803d", "#dcfce7"
AMBER, AMBER_BG = "#b45309", "#fef3c7"
GREY, GREY_BG = "#475569", "#f1f5f9"


def box(ax, x, y, w, h, text, fc, ec, size=10, weight="normal"):
    ax.add_patch(FancyBboxPatch((x, y), w, h, boxstyle="round,pad=0.02,rounding_size=0.12",
                                fc=fc, ec=ec, lw=1.6))
    ax.text(x + w / 2, y + h / 2, text, ha="center", va="center", fontsize=size,
            color=INK, weight=weight, linespacing=1.35)


def arrow(ax, p, q, color=GREY, style="-|>", ls="-"):
    ax.add_patch(FancyArrowPatch(p, q, arrowstyle=style, mutation_scale=14, lw=1.6,
                                 color=color, linestyle=ls, shrinkA=0, shrinkB=0))


# --------------------------------------------------------------------------- architecture
def architecture():
    fig, ax = plt.subplots(figsize=(12, 5.6), dpi=160)
    ax.set_xlim(0, 12)
    ax.set_ylim(-0.3, 5.6)
    ax.axis("off")

    # probe band
    box(ax, 0.3, 4.75, 11.4, 0.6,
        "probe: wall time, allocations, peak memory  -  cancels the context on timeout / memory limit",
        AMBER_BG, AMBER, size=10)

    box(ax, 0.3, 1.9, 1.9, 1.5, "measurements.csv\n\nstation;value\nstation;value\n...", GREY_BG, GREY, size=9)
    box(ax, 2.75, 2.15, 1.75, 1.0, "SplitPoints\nline-aligned\nbyte ranges", BLUE_BG, BLUE, size=9)

    ys = [3.45, 2.3, 1.15, 0.0]
    labels = ["Worker 1", "Worker 2", "...", "Worker N"]
    for y, lab in zip(ys, labels):
        if lab == "...":
            ax.text(7.05, y + 0.42, "...", ha="center", va="center", fontsize=16, color=GREY)
            continue
        box(ax, 5.3, y, 3.5, 0.85,
            f"{lab}\nReadSlice > ParseLine > Table.Observe\n(private table, no locks)",
            GREEN_BG, GREEN, size=8.5)
        arrow(ax, (4.5, 2.65), (5.3, y + 0.42), color=BLUE)
        arrow(ax, (8.8, y + 0.42), (9.4, 2.65), color=GREEN)

    box(ax, 9.4, 2.15, 1.2, 1.0, "Merge\ntables", BLUE_BG, BLUE, size=9)
    arrow(ax, (2.2, 2.65), (2.75, 2.65), color=GREY)
    box(ax, 10.8, 3.35, 0.95, 0.75, "Sort\nby name", BLUE_BG, BLUE, size=8.5)
    box(ax, 10.8, 1.2, 0.95, 0.95, "report\n.csv", GREY_BG, GREY, size=9)
    arrow(ax, (10.6, 2.9), (10.8, 3.6), color=BLUE)
    arrow(ax, (11.27, 3.35), (11.27, 2.15), color=BLUE)

    ax.set_title("rowcrunch pipeline (parallel mode)", fontsize=13, color=INK, pad=6, weight="bold")
    fig.savefig(os.path.join(IMG, "architecture.png"), bbox_inches="tight", facecolor="white")
    plt.close(fig)


# --------------------------------------------------------------------------- chunking
def chunking():
    import random

    rng = random.Random(3)
    fig, ax = plt.subplots(figsize=(12, 3.9), dpi=160)
    ax.set_xlim(-0.3, 12.3)
    ax.set_ylim(-1.9, 3.1)
    ax.axis("off")

    # a file drawn as a row of variable-length "lines"
    x = 0.0
    lines = []
    while x < 12.0:
        w = rng.uniform(0.16, 0.42)
        lines.append((x, min(w, 12.0 - x)))
        x += w
    for lx, lw in lines:
        ax.add_patch(Rectangle((lx, 1.0), lw, 0.8, fc="#e2e8f0", ec="white", lw=0.8))

    parts = 4
    colors = [BLUE, GREEN, AMBER, "#7c3aed"]
    aligned = [0.0]
    for i in range(1, parts):
        guess = 12.0 * i / parts
        nxt = next(lx + lw for lx, lw in lines if lx + lw > guess)
        aligned.append(nxt)
        ax.plot([guess, guess], [0.8, 2.15], ls=(0, (3, 3)), color=GREY, lw=1.3)
        ax.text(guess, 2.35, f"size x {i}/{parts}", ha="center", fontsize=8.5, color=GREY)
        ax.annotate("", xy=(nxt, 1.0), xytext=(guess, 1.0),
                    arrowprops=dict(arrowstyle="-|>", color=INK, lw=1.4))
        ax.plot([nxt, nxt], [0.75, 1.85], color=INK, lw=2.2)
    aligned.append(12.0)

    for i, (a, b) in enumerate(zip(aligned, aligned[1:])):
        c = colors[i]
        ax.add_patch(Rectangle((a, 0.15), b - a, 0.38, fc=c, ec="white", lw=1.2, alpha=0.9))
        ax.text((a + b) / 2, 0.34, f"Worker {i + 1}", ha="center", va="center",
                fontsize=9.5, color="white", weight="bold")

    ax.text(0, 2.9, "Each guess is moved forward to the first byte after the next '\\n'",
            fontsize=11, color=INK, weight="bold", ha="left")
    ax.text(0, -0.55, "dashed = equal split of the file size      solid = start of the next full line (chosen boundary)",
            fontsize=9, color=GREY, ha="left")
    ax.text(0, -1.1, "Consecutive workers share a boundary, so no line is read twice and none is lost.",
            fontsize=9, color=GREY, ha="left")
    fig.savefig(os.path.join(IMG, "chunking.png"), bbox_inches="tight", facecolor="white")
    plt.close(fig)


# --------------------------------------------------------------------------- terminal captures
FONT = "/usr/share/fonts/truetype/dejavu/DejaVuSansMono.ttf"
FONT_B = "/usr/share/fonts/truetype/dejavu/DejaVuSansMono-Bold.ttf"


def terminal(name, title, blocks, width=980):
    """blocks: list of (kind, text) with kind in {'cmd', 'out', 'dim'}."""
    f = ImageFont.truetype(FONT, 17)
    fb = ImageFont.truetype(FONT_B, 17)
    ft = ImageFont.truetype(FONT, 14)
    line_h, pad, bar = 27, 22, 40
    height = bar + pad * 2 + line_h * len(blocks)
    im = Image.new("RGB", (width, height), "#0f172a")
    d = ImageDraw.Draw(im)
    d.rectangle([0, 0, width, bar], fill="#1e293b")
    for i, c in enumerate(["#ef4444", "#f59e0b", "#22c55e"]):
        d.ellipse([16 + i * 24, 13, 28 + i * 24, 25], fill=c)
    d.text((width / 2, bar / 2), title, font=ft, fill="#94a3b8", anchor="mm")
    y = bar + pad
    for kind, text in blocks:
        if kind == "cmd":
            d.text((pad, y), "$ ", font=fb, fill="#22c55e")
            d.text((pad + 28, y), text, font=fb, fill="#e2e8f0")
        elif kind == "dim":
            d.text((pad, y), text, font=f, fill="#64748b")
        else:
            d.text((pad, y), text, font=f, fill="#cbd5e1")
        y += line_h
    im.save(os.path.join(IMG, name))


def captures():
    def head(path, n):
        with open(os.path.join(ROOT, path), encoding="utf-8", newline="") as fh:
            return [ln.rstrip("\r\n") for ln in fh.readlines()[:n]]

    terminal("term-input.png", "testdata/sample.csv",
             [("cmd", "head -n 8 testdata/sample.csv")] + [("out", ln) for ln in head("testdata/sample.csv", 8)],
             width=760)
    terminal("term-report.png", "testdata/expected.csv",
             [("dim", "# columns: station;min;max;mean;count"),
              ("cmd", "head -n 8 testdata/expected.csv")]
             + [("out", ln) for ln in head("testdata/expected.csv", 8)],
             width=900)

    # project tree built from the real directory
    skip = {".git", "bin", "__pycache__"}
    tree = []

    def walk(path, prefix=""):
        entries = sorted(e for e in os.listdir(path) if e not in skip)
        entries.sort(key=lambda e: (not os.path.isdir(os.path.join(path, e)), e))
        for i, e in enumerate(entries):
            last = i == len(entries) - 1
            tree.append(prefix + ("└── " if last else "├── ") + e)
            if os.path.isdir(os.path.join(path, e)):
                walk(os.path.join(path, e), prefix + ("    " if last else "│   "))

    walk(ROOT)
    terminal("term-layout.png", "rowcrunch",
             [("cmd", "tree -I bin"), ("out", ".")] + [("out", t) for t in tree], width=760)


if __name__ == "__main__":
    architecture()
    chunking()
    captures()
    print("images written to", IMG)
