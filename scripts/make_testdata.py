#!/usr/bin/env python3
"""Builds testdata/sample.csv and testdata/expected.csv.

The expected report is computed here, independently of the Go code, from
integer hundredths, so the Go tests compare two separate implementations.

Sample contents (on purpose):
  * mixed number styles: 2 decimals, 1 decimal, integers, '+' prefix
  * one CRLF line, two malformed lines, a non-ASCII station name
  * no trailing newline at the end of the file
"""
import os
import random

HERE = os.path.dirname(os.path.abspath(__file__))
OUT = os.path.join(HERE, "..", "testdata")

STATIONS = [
    "Alpha", "Bravo", "Charlie", "Delta", "Echo", "Foxtrot", "Golf", "Hotel",
    "India", "Juliet", "Kilo", "Lima", "Mike", "November", "Oscar", "Papa",
    "São Tomé", "Quebec", "Romeo", "Sierra",
]
ROWS = 5000
MALFORMED = ["this line has no separator", "Alpha;not-a-number"]


def style(h: int, rng: random.Random) -> str:
    """Render hundredths `h` in one of several accepted spellings."""
    sign = "-" if h < 0 else ""
    a = abs(h)
    whole, frac = divmod(a, 100)
    choice = rng.random()
    if frac == 0 and choice < 0.3:
        return f"{sign}{whole}"                      # integer: "7"
    if frac % 10 == 0 and choice < 0.6:
        return f"{sign}{whole}.{frac // 10}"         # one decimal: "7.5"
    if h >= 0 and choice < 0.65:
        return f"+{whole}.{frac:02d}"                # explicit plus
    return f"{sign}{whole}.{frac:02d}"               # two decimals


def main() -> None:
    rng = random.Random(7)
    stats = {}
    lines = []
    for i in range(ROWS):
        st = rng.choice(STATIONS)
        h = rng.randint(-9999, 9999)
        if i % 97 == 0:
            h = int(h / 100) * 100                   # force some integers
        elif i % 53 == 0:
            h = int(h / 10) * 10                     # force some 1-decimal
        s = stats.setdefault(st, [h, h, 0, 0])
        s[0] = min(s[0], h)
        s[1] = max(s[1], h)
        s[2] += h
        s[3] += 1
        text = f"{st};{style(h, rng)}"
        lines.append(text + ("\r\n" if i == 1234 else "\n"))
    lines.insert(10, MALFORMED[0] + "\n")
    lines.insert(2500, MALFORMED[1] + "\n")
    lines[-1] = lines[-1].rstrip("\n")               # no final newline

    with open(os.path.join(OUT, "sample.csv"), "w", encoding="utf-8", newline="") as f:
        f.writelines(lines)

    def fmt(h: int) -> str:
        sign = "-" if h < 0 else ""
        a = abs(h)
        return f"{sign}{a // 100}.{a % 100:02d}"

    with open(os.path.join(OUT, "expected.csv"), "w", encoding="utf-8", newline="") as f:
        for st in sorted(stats):
            lo, hi, total, n = stats[st]
            mean = total / n / 100.0                 # same IEEE steps as the Go code
            f.write(f"{st};{fmt(lo)};{fmt(hi)};{mean:.2f};{n}\n")

    print(f"stations={len(stats)} rows={ROWS} skipped={len(MALFORMED)}")


if __name__ == "__main__":
    main()
