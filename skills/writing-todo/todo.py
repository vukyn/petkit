#!/usr/bin/env python3
"""Mechanical checks and listings for a TODO.md written with the writing-todo skill.

    python3 todo.py audit  [TODO.md]   # rule violations, orphans, drift; exit 1 if any error
    python3 todo.py status [TODO.md]   # open items grouped by area, as JSON

Only what can be decided from the text lives here. Judgment (a claim with no
number, an entry that is stale against the code, what to do next) belongs to the
skill that calls this script.
"""

import json
import re
import sys
from pathlib import Path

CODE = r"[A-Z]{3,4}-\d{3}"
CODE_RE = re.compile(rf"`({CODE})`")
BARE_CODE_RE = re.compile(rf"\b({CODE})\b")
INDEX_ROW_RE = re.compile(rf"^\|\s*`({CODE})`\s*\|\s*([a-z]+)\s*\|\s*(.*?)\s*\|\s*$")
AREA_ROW_RE = re.compile(r"^\|\s*`([A-Z]{3,4})`\s*\|\s*(.*?)\s*\|\s*(.*?)\s*\|\s*$")
ENTRY_RE = re.compile(rf"^- (\[( |x)\] )?`({CODE})`")
RAISED_RE = re.compile(r"^\s+Raised \d{4}-\d{2}-\d{2}")
LINE_REF_RE = re.compile(r"\b(?:at |on )?line \d+\b|\bdòng \d+\b", re.IGNORECASE)
STATUSES = {"open", "done", "refused"}
SECTIONS = ["The codes", "Done", "Not done", "Decided against"]
REOPEN_HINT = re.compile(r"reopen|re-raise|bring it back|come back|would have to change|mở lại", re.IGNORECASE)


def section_of(title):
    for name in SECTIONS:
        if title.startswith(name):
            return name
    return None


def parse(path):
    lines = Path(path).read_text(encoding="utf-8").splitlines()
    doc = {"lines": lines, "headings": [], "index": {}, "index_dupes": [], "areas": {},
           "entries": [], "section_at": []}
    current = None
    for i, line in enumerate(lines, 1):
        if line.startswith("## "):
            current = section_of(line[3:].strip())
            doc["headings"].append((i, line[3:].strip(), current))
        doc["section_at"].append(current)
        m = INDEX_ROW_RE.match(line)
        if m and current == "The codes":
            code, status, summary = m.groups()
            if code in doc["index"]:
                doc["index_dupes"].append((i, code))
            doc["index"][code] = {"line": i, "status": status, "summary": summary}
            continue
        m = AREA_ROW_RE.match(line)
        if m and current == "The codes" and not INDEX_ROW_RE.match(line):
            doc["areas"][m.group(1)] = {"line": i, "short": m.group(2).replace("*", ""),
                                        "covers": m.group(3)}
        m = ENTRY_RE.match(line)
        if m and current in ("Done", "Not done", "Decided against"):
            doc["entries"].append({"line": i, "code": m.group(3), "box": m.group(2),
                                   "section": current, "title": line})
    return doc


def audit(path):
    doc = parse(path)
    lines, index, entries = doc["lines"], doc["index"], doc["entries"]
    errors, warnings, notes = [], [], []

    def err(line, msg):
        errors.append({"line": line, "msg": msg})

    def warn(line, msg):
        warnings.append({"line": line, "msg": msg})

    # Sections: present once each, in order.
    found = [h[2] for h in doc["headings"] if h[2]]
    for name in SECTIONS:
        if found.count(name) != 1:
            err(0, f"section '## {name}' appears {found.count(name)} times (want exactly 1)")
    order = [n for n in found if n in SECTIONS]
    if order != sorted(order, key=SECTIONS.index):
        err(0, f"sections out of order: {order} (want {SECTIONS})")

    # Index rows.
    for line, code in doc["index_dupes"]:
        err(line, f"{code} has two index rows")
    if not index:
        err(0, "no index rows found under '## The codes'")
    for code, row in index.items():
        if row["status"] not in STATUSES:
            err(row["line"], f"{code} status '{row['status']}' is not one of {sorted(STATUSES)}")
        area = code.split("-")[0]
        if doc["areas"] and area not in doc["areas"]:
            err(row["line"], f"{code} uses area {area}, which the area table does not define")

    # Entries against the index.
    by_code = {}
    for e in entries:
        by_code.setdefault(e["code"], []).append(e)
    for code, es in by_code.items():
        secs = [e["section"] for e in es]
        if len(es) > 1:
            err(es[1]["line"], f"{code} has {len(es)} entries ({', '.join(secs)}); a code is one entry")
        if code not in index:
            err(es[0]["line"], f"{code} has an entry but no index row")
    for code, row in index.items():
        es = by_code.get(code, [])
        st = row["status"]
        want = {"open": "Not done", "done": "Done", "refused": "Decided against"}.get(st)
        if st == "open" and not any(e["section"] == "Not done" for e in es):
            err(row["line"], f"{code} is open but has no entry under '## Not done' (lost title line?)")
        if st == "done" and any(e["section"] == "Not done" for e in es):
            err(row["line"], f"{code} is done but its entry is still under '## Not done'")
        if st == "refused" and not any(e["section"] == "Decided against" for e in es):
            err(row["line"], f"{code} is refused but has no entry under '## Decided against'")
        if st == "done" and not es:
            notes.append({"line": row["line"], "msg": f"{code} is done with no entry (fine for a capability paragraph that names it; check it is mentioned in Done)"})
        for e in es:
            if want and e["section"] != want and not (st == "done" and e["section"] == "Done"):
                if not (st == "open" and e["section"] == "Not done"):
                    err(e["line"], f"{code} entry sits in '{e['section']}' but the index says {st}")
            if e["section"] == "Not done" and e["box"] == "x":
                err(e["line"], f"{code} is ticked [x] but still under '## Not done' (closing moves it)")
            if e["section"] == "Not done" and e["box"] is None:
                warn(e["line"], f"{code} entry under '## Not done' has no '[ ]' checkbox")
            if e["section"] == "Done" and e["box"] == " ":
                err(e["line"], f"{code} is under '## Done' but unticked")
    if "refused" in {r["status"] for r in index.values()}:
        for e in entries:
            if e["section"] != "Decided against":
                continue
            body = entry_body(lines, e["line"])
            if not REOPEN_HINT.search(body):
                warn(e["line"], f"{e['code']} is refused but does not say what would bring it back")

    # Orphan bodies: an entry whose title line was lost leaves its 'Raised …' body
    # behind, where it reads as part of the entry above. It shows as a 'Raised' line
    # that starts a new block (blank line above), or a second 'Raised' in one entry.
    raised_seen = {}
    for i, line in enumerate(lines, 1):
        if not RAISED_RE.match(line):
            continue
        prev = lines[i - 2] if i >= 2 else ""
        if ENTRY_RE.match(prev):
            raised_seen[i - 1] = True
            continue
        owner_line, owner = owner_entry(lines, i)
        sec = doc["section_at"][i - 1]
        hint = f" (it reads as part of {owner})" if owner else ""
        if prev.strip() == "":
            err(i, f"orphan entry body in '{sec}': a 'Raised …' block with no title line above{hint}")
        elif owner_line in raised_seen:
            err(i, f"second 'Raised …' line inside {owner}: probably another entry's body whose title was lost")
        else:
            raised_seen[owner_line] = True

    # References: line numbers, and codes that are not in the index.
    for i, line in enumerate(lines, 1):
        if LINE_REF_RE.search(line) and not line.lstrip().startswith("⚠️ Refer"):
            if "by code" not in line.lower():
                warn(i, "refers to a line number; refer by code instead")
        if doc["section_at"][i - 1] == "The codes" and INDEX_ROW_RE.match(line):
            continue
        for code in BARE_CODE_RE.findall(line):
            if code not in index and code.split("-")[0] in doc["areas"]:
                warn(i, f"mentions {code}, which has no index row")

    # Housekeeping facts for the caller.
    counts = {s: sum(1 for r in index.values() if r["status"] == s) for s in sorted(STATUSES)}
    next_free = {}
    for area in doc["areas"]:
        nums = [int(c.split("-")[1]) for c in index if c.startswith(area + "-")]
        next_free[area] = f"{area}-{(max(nums) + 1 if nums else 1):03d}"
    sizes = []
    for e in entries:
        body = entry_body(lines, e["line"])
        sizes.append((len(body), e["code"], e["section"], e["line"]))
    sizes.sort(reverse=True)
    return {
        "file": str(path),
        "errors": errors,
        "warnings": warnings,
        "notes": notes,
        "counts": counts,
        "areas": {a: v["short"] for a, v in doc["areas"].items()},
        "next_free": next_free,
        "largest_entries": [{"code": c, "section": s, "line": l, "chars": n} for n, c, s, l in sizes[:8]],
        "done_section_entries": sum(1 for e in entries if e["section"] == "Done"),
        "file_lines": len(lines),
    }


def entry_body(lines, title_line):
    """Text from the title line to the next entry title or section heading."""
    out = []
    for line in lines[title_line - 1:]:
        if out and (ENTRY_RE.match(line) or line.startswith("## ")):
            break
        out.append(line)
    return "\n".join(out)


def owner_entry(lines, at):
    """(1-based title line, code) of the entry a line falls under, or (None, None)."""
    for j in range(at - 2, -1, -1):
        m = ENTRY_RE.match(lines[j])
        if m:
            return j + 1, m.group(3)
        if lines[j].startswith("## "):
            return None, None
    return None, None


def status(path):
    doc = parse(path)
    groups = {}
    for code, row in doc["index"].items():
        if row["status"] != "open":
            continue
        area = code.split("-")[0]
        e = next((x for x in doc["entries"] if x["code"] == code and x["section"] == "Not done"), None)
        body = entry_body(doc["lines"], e["line"]) if e else ""
        groups.setdefault(area, []).append({
            "code": code,
            "summary": row["summary"],
            "line": e["line"] if e else None,
            "mentions": sorted(set(BARE_CODE_RE.findall(body)) - {code}),
        })
    areas = doc["areas"]
    order = list(areas) + sorted(a for a in groups if a not in areas)
    return {
        "file": str(path),
        "open": sum(len(v) for v in groups.values()),
        "groups": [{"area": a, "short": areas.get(a, {}).get("short", ""),
                    "covers": areas.get(a, {}).get("covers", ""), "items": groups[a]}
                   for a in order if a in groups],
    }


def find_todo(arg):
    if arg:
        return Path(arg)
    here = Path.cwd()
    for d in [here, *here.parents]:
        if (d / "TODO.md").is_file():
            return d / "TODO.md"
        if (d / ".git").exists():
            break
    sys.exit("todo.py: no TODO.md in this directory or up to the repository root")


def main():
    if len(sys.argv) < 2 or sys.argv[1] not in ("audit", "status"):
        sys.exit(__doc__)
    path = find_todo(sys.argv[2] if len(sys.argv) > 2 else None)
    if sys.argv[1] == "audit":
        out = audit(path)
        print(json.dumps(out, ensure_ascii=False, indent=1))
        sys.exit(1 if out["errors"] else 0)
    print(json.dumps(status(path), ensure_ascii=False, indent=1))


if __name__ == "__main__":
    main()
