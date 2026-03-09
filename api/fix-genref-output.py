#!/usr/bin/env python3
"""
Post-processes genref markdown output to fix two upstream issues:
1. Trailing whitespace on lines (causes CI diffs when compared across platforms)
2. '<td>(Members of' on a single line, which MDX parses as broken JSX
"""
import glob
import re
import sys

output_dir = sys.argv[1] if len(sys.argv) > 1 else "."

for path in glob.glob(f"{output_dir}/*.mdx"):
    content = open(path).read()
    # Strip trailing whitespace from every line
    fixed = re.sub(r"[^\S\n]+\n", "\n", content)
    # Put the embedded-struct note on its own line so MDX doesn't choke on <td>(
    fixed = fixed.replace("<td>(Members of", "<td>\n(Members of")
    if fixed != content:
        open(path, "w").write(fixed)
