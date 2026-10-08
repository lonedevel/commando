#!/usr/bin/env python3
"""Measure how many options commando reads from the manuals on this system.

    scripts/survey.py run BIN OUT.json    parse every man page with BIN
    scripts/survey.py run BIN OUT.json --help-tools
                                          also every command on PATH that has
                                          no man page but answers --help
    scripts/survey.py compare OLD.json NEW.json
                                          options and names gained and lost

Build two binaries (before and after a parser change), run both, and
compare: a change should gain names without losing any.

Reading man pages runs nothing but man. --help-tools runs every program on
PATH with --help to find the ones without a manual; some programs ignore
--help and start anyway (on a Mac, some open windows), so use it only on
a machine set up for it, such as a container.
"""
import concurrent.futures as cf
import json
import os
import re
import subprocess
import sys
import tempfile


def man_dirs():
    """The man page search path. Linux's man-db prints it for "man -w";
    macOS has manpath(1) instead, and "man -w" there prints nothing."""
    for cmd in (["manpath"], ["man", "--path"], ["man", "-w"]):
        try:
            out = subprocess.run(cmd, capture_output=True, text=True, timeout=10).stdout
        except (OSError, subprocess.TimeoutExpired):
            continue
        dirs = [d for d in out.strip().split(":") if d.startswith("/")]
        if dirs:
            return dirs
    if os.environ.get("MANPATH"):
        return [d for d in os.environ["MANPATH"].split(":") if d]
    return ["/usr/share/man", "/usr/local/share/man", "/opt/homebrew/share/man"]


def man_pages():
    pages = set()
    for d in man_dirs():
        for sect in ("man1", "man8"):
            try:
                for f in os.listdir(os.path.join(d, sect)):
                    m = re.match(r"(.+)\.[18][a-z]*(\.gz)?$", f)
                    if m:
                        pages.add(m.group(1))
            except OSError:
                pass
    if not pages:
        sys.exit("survey: found no man pages; set MANPATH to where they are")
    return sorted(pages)


def help_tools():
    """Commands on PATH with no man page whose --help lists options."""
    seen, cmds = set(), []
    for d in os.environ["PATH"].split(":"):
        try:
            names = os.listdir(d)
        except OSError:
            continue
        for n in names:
            p = os.path.join(d, n)
            if n not in seen and os.access(p, os.X_OK) and not os.path.isdir(p):
                seen.add(n)
                cmds.append(n)

    def ok(n):
        if subprocess.run(["man", "-w", n], capture_output=True).returncode == 0:
            return False
        try:
            h = subprocess.run([n, "--help"], capture_output=True, text=True, timeout=3,
                               stdin=subprocess.DEVNULL, cwd=scratch).stdout
        except Exception:
            return False
        return len(re.findall(r"(?m)^\s{1,10}-{1,2}[A-Za-z]", h)) >= 3

    # Some tools write files when run; keep them out of the current folder.
    with tempfile.TemporaryDirectory() as scratch, cf.ThreadPoolExecutor(16) as ex:
        return [n for n, k in zip(cmds, ex.map(ok, cmds)) if k]


def dump(binary, cache, name, kind):
    env = dict(os.environ, COMMANDO_CACHE_DIR=cache, COMMANDO_NO_COMPLETIONS="1")
    if kind == "man":
        env["COMMANDO_NO_HELP"] = "1"  # never run the command itself
    try:
        out = subprocess.run([binary, "--dump", "--no-cache", name], capture_output=True,
                             text=True, timeout=30, env=env, cwd=cache).stdout
        return [o["names"] for o in json.loads(out).get("options") or []]
    except Exception:
        return None


def run(binary, out, with_help):
    pages = {"man": man_pages()}
    if with_help:
        pages["help"] = help_tools()
    result = {}
    with tempfile.TemporaryDirectory() as cache, cf.ThreadPoolExecutor(8) as ex:
        for kind, names in pages.items():
            opts = ex.map(lambda n, k=kind: dump(binary, cache, n, k), names)
            result[kind] = {n: o for n, o in zip(names, opts) if o is not None}
    with open(out, "w") as f:
        json.dump(result, f)
    for kind, r in result.items():
        print(f"{kind}: {len(r)} pages, {sum(len(o) for o in r.values())} options, "
              f"{sum(len(n) for o in r.values() for n in o)} names")


def compare(old_path, new_path):
    old, new = json.load(open(old_path)), json.load(open(new_path))
    for kind in old:
        if kind not in new:
            continue
        o, n = old[kind], new[kind]
        common = sorted(set(o) & set(n))
        gained = lost = opts_before = opts_after = 0
        losses = []
        for p in common:
            a = {x for names in o[p] for x in names}
            b = {x for names in n[p] for x in names}
            gained += len(b - a)
            lost += len(a - b)
            opts_before += len(o[p])
            opts_after += len(n[p])
            if a - b:
                losses.append((p, sorted(a - b)))
        print(f"{kind}: {len(common)} pages; options {opts_before} -> {opts_after}; "
              f"names gained {gained}, lost {lost}")
        for p, names in losses[:40]:
            print(f"  lost in {p}: {' '.join(names[:12])}")


if __name__ == "__main__":
    if len(sys.argv) in (4, 5) and sys.argv[1] == "run" and sys.argv[4:] in ([], ["--help-tools"]):
        run(sys.argv[2], sys.argv[3], sys.argv[4:] == ["--help-tools"])
    elif len(sys.argv) == 4 and sys.argv[1] == "compare":
        compare(sys.argv[2], sys.argv[3])
    else:
        sys.exit(__doc__)
