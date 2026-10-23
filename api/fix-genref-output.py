#!/usr/bin/env python3
"""
Post-processes genref markdown output to fix known upstream issues:
1. Trailing whitespace on lines (causes CI diffs when compared across platforms)
2. '<td>(Members of' on a single line, which MDX parses as broken JSX
3. The SourcesScopes alias is documented as a local type on some Go versions
   and as k8sconsts.SourcesScopes on others. Always emit the external link.
"""
import glob
import re
import sys

output_dir = sys.argv[1] if len(sys.argv) > 1 else "."

# type SourcesScopes = k8sconsts.SourcesScopes. genref follows the alias on the
# CI Go toolchain and inlines a local section on newer local toolchains.
SOURCES_SCOPES_LOCAL = (
    '<a href="#odigos-io-v1alpha1-SourcesScopes"><code>SourcesScopes</code></a>'
)
SOURCES_SCOPES_EXTERNAL = (
    '<a href="https://pkg.go.dev/github.com/odigos-io/odigos/api/k8sconsts#SourcesScopes">'
    "<code>k8sconsts.SourcesScopes</code></a>"
)
SOURCES_SCOPES_SECTION = re.compile(
    r'\n## `SourcesScopes` <a id="odigos-io-v1alpha1-SourcesScopes"></a>.*\Z',
    re.DOTALL,
)

for path in glob.glob(f"{output_dir}/*.mdx"):
    content = open(path).read()
    # Strip trailing whitespace from every line
    fixed = re.sub(r"[^\S\n]+\n", "\n", content)
    # Put the embedded-struct note on its own line so MDX doesn't choke on <td>(
    fixed = fixed.replace("<td>(Members of", "<td>\n(Members of")
    fixed = fixed.replace(SOURCES_SCOPES_LOCAL, SOURCES_SCOPES_EXTERNAL)
    fixed = SOURCES_SCOPES_SECTION.sub("", fixed)
    if fixed != content:
        open(path, "w").write(fixed)
