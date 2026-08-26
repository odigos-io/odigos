#!/usr/bin/env python3
"""Regression tests for the odiglet security-profile behaviour of the odigos helm chart.

Runs `helm template` over many permutations of values and asserts on the rendered
odiglet DaemonSet (and the odigos-configuration ConfigMap for the wasp flag).

What it protects
----------------
* Both `odiglet.securityProfile` values (legacy / unprivileged) render the
  expected pod and per-container securityContext, and `unprivileged` is gated
  behind an Odigos Enterprise token.
* BACKWARD COMPATIBILITY INVARIANT: with no values set, and with the legacy
  `odiglet.unPrivileged=true` set, the rendered DaemonSet is byte-identical to the
  render of the same values from BASELINE_REF (the commit before security profiles
  were introduced), except for a small, explicitly whitelisted set of intentional
  differences.  Rendering the baseline uses a throwaway `git worktree`.
* Every pre-existing individual switch still takes effect and still overrides the
  profile defaults.
* The guards (`fail` calls and values.schema.json enums) fire with a useful message.

Usage
-----
    make helm-test
    python3 scripts/helm-odiglet-security-test.py [-v] [--filter SUBSTR] [--skip-baseline]

Requirements: helm >= 3.9 and PyYAML.
"""

from __future__ import annotations

import argparse
import difflib
import json
import re
import shutil
import subprocess
import sys
import tempfile
from pathlib import Path

try:
    import yaml
except ImportError:  # pragma: no cover
    sys.exit("error: PyYAML is required (pip install pyyaml)")

REPO = Path(__file__).resolve().parents[1]
CHART = REPO / "helm" / "odigos"
DS_TEMPLATE = "templates/odiglet/daemonset.yaml"
CM_TEMPLATE = "templates/odigos-configuration-cm.yaml"

# The commit right before `odiglet.securityProfile` was introduced.  The default
# render and the legacy `odiglet.unPrivileged=true` render must not drift from it.
BASELINE_REF = "b605830bfb1ab17fc1691fd11c5bf71d3268ff3f"

DEFAULT_KUBE = "1.30.0"

TRACES_ONLY = ["--set-json", 'signals=["traces"]']
MM_INIT = ["--set", "instrumentor.mountMethod=k8s-init-container"]

PROFILE_LEGACY = ["--set", "odiglet.securityProfile=legacy"]
PROFILE_UNPRIV = ["--set", "odiglet.securityProfile=unprivileged"]

# The unprivileged profile is an enterprise capability.  `helm template` cannot
# see a pre-existing odigos-pro secret (that is a cluster lookup), so these are
# the two ways to satisfy the gate from values alone.
ENTERPRISE = ["--set", "onPremToken=test-token"]
# externalOnpremTokenSecret satisfies the profile gate on its own, but the chart
# then has no pull secret for registry.odigos.io, so a full render needs both.
ENTERPRISE_EXTERNAL_SECRET = ["--set", "externalOnpremTokenSecret=true",
                              "--set", "externalOnpremPullSecret=true"]

# The unprivileged profile refuses to collect logs/metrics, refuses every mount
# method but k8s-init-container and requires an enterprise token, so this is the
# smallest set of values that lets it render at all.
UNPRIV_BASE = TRACES_ONLY + MM_INIT + ENTERPRISE

LEGACY_ODIGLET_CAPS = [
    "SYS_ADMIN", "BPF", "PERFMON", "SYS_PTRACE", "DAC_READ_SEARCH", "IPC_LOCK", "SYS_RESOURCE",
]
LEGACY_DATA_COLLECTION_CAPS = ["SYS_ADMIN", "BPF", "PERFMON", "IPC_LOCK"]
# CAP_DAC_OVERRIDE joins the set only when the host mounts are present: the
# agent directory and the device plugin socket are root-owned hostPaths that a
# non-root uid has no other way to write. noHostPathMounts takes it away again.
PROFILE_ODIGLET_CAPS_NO_HOSTPATH = ["BPF", "PERFMON", "SYS_PTRACE", "DAC_READ_SEARCH", "FOWNER"]
PROFILE_ODIGLET_CAPS = PROFILE_ODIGLET_CAPS_NO_HOSTPATH + ["DAC_OVERRIDE"]

RUN_AS = 1000
TRACEFS_DEFAULT = "/sys/kernel"

# ---------------------------------------------------------------------------
# rendering
# ---------------------------------------------------------------------------


class Render:
    def __init__(self, cmd, proc):
        self.cmd = cmd
        self.ok = proc.returncode == 0
        self.stdout = proc.stdout
        self.stderr = proc.stderr

    @property
    def error(self) -> str:
        return (self.stderr or self.stdout).strip()

    def docs(self):
        return [d for d in yaml.safe_load_all(self.stdout) if d]

    def kind(self, kind, name):
        for d in self.docs():
            if d.get("kind") == kind and d.get("metadata", {}).get("name") == name:
                return d
        return None

    def pretty_cmd(self) -> str:
        out = []
        for part in self.cmd:
            out.append(part if re.fullmatch(r"[\w./=:-]*", part) else f"'{part}'")
        return " ".join(out)


class _EmptyProc:
    returncode = 0
    stdout = ""
    stderr = ""


_EMPTY_RENDER = Render(["helm", "template", "<skipped>"], _EmptyProc())

_render_cache: dict = {}


def render(sets=(), kube=DEFAULT_KUBE, chart=None, show=DS_TEMPLATE) -> Render:
    chart = Path(chart or CHART)
    key = (str(chart), tuple(sets), kube, show)
    if key not in _render_cache:
        cmd = [
            "helm", "template", "odigos", str(chart),
            "--namespace", "odigos-system",
            "--kube-version", kube,
        ]
        if show:
            cmd += ["-s", show]
        cmd += list(sets)
        proc = subprocess.run(cmd, capture_output=True, text=True)
        _render_cache[key] = Render(cmd, proc)
    return _render_cache[key]


# ---------------------------------------------------------------------------
# reporting
# ---------------------------------------------------------------------------


class _Missing:
    def __repr__(self):
        return "<absent>"


MISSING = _Missing()


def fmt(value) -> str:
    if value is MISSING:
        return "<absent>"
    return json.dumps(value, sort_keys=True, default=str)


def explain(expected, actual) -> list:
    """Point at the exact key(s) that differ, so failures are readable."""
    lines = []
    if isinstance(expected, dict) and isinstance(actual, dict):
        for k in sorted(set(expected) | set(actual)):
            e, a = expected.get(k, MISSING), actual.get(k, MISSING)
            if e != a:
                lines.append(f"        key {k!r}: expected {fmt(e)}, got {fmt(a)}")
    elif isinstance(expected, list) and isinstance(actual, list):
        for line in difflib.unified_diff(
            [fmt(x) for x in expected], [fmt(x) for x in actual],
            fromfile="expected", tofile="actual", lineterm="", n=1,
        ):
            lines.append(f"        {line}")
    return lines


class Reporter:
    def __init__(self, verbose=False, filt=None):
        self.verbose = verbose
        self.filter = filt
        self.checks = 0
        self.failures = []
        self.known_failures = []      # xfail that failed as expected
        self.unexpected_passes = []   # xfail that unexpectedly passed
        self.cases = 0

    def selected(self, group, name) -> bool:
        if not self.filter:
            return True
        return self.filter.lower() in f"{group}/{name}".lower()

    def record(self, case, what, message_lines):
        block = [f"  [{case.group}] {case.name}", f"    check: {what}"]
        block += [f"    {ln}" for ln in message_lines]
        block.append(f"    render: {case.render.pretty_cmd()}")
        text = "\n".join(block)
        if case.xfail:
            self.known_failures.append((case, what, case.xfail, text))
        else:
            self.failures.append(text)

    def summary(self) -> int:
        print()
        if self.known_failures:
            print(f"KNOWN BUGS (expected failures, not fatal) - {len(self.known_failures)}:")
            for _case, _what, why, text in self.known_failures:
                print(text)
                print(f"    known bug: {why}")
            print()
        if self.unexpected_passes:
            print(f"UNEXPECTED PASSES - {len(self.unexpected_passes)}:")
            for case, what, why in self.unexpected_passes:
                print(f"  [{case.group}] {case.name}: {what}")
                print(f"    this was marked as a known bug ({why}) but now passes.")
                print("    the bug looks fixed - drop the xfail marker so it stays fixed.")
            print()
        if self.failures:
            print(f"FAILURES - {len(self.failures)}:")
            for text in self.failures:
                print(text)
                print()
        total_bad = len(self.failures) + len(self.unexpected_passes)
        print(
            f"{self.cases} permutations, {self.checks} assertions, "
            f"{len(self.failures)} failed, {len(self.unexpected_passes)} unexpected passes, "
            f"{len(self.known_failures)} known bugs."
        )
        if total_bad:
            print("RESULT: FAIL")
            return 1
        print("RESULT: PASS")
        return 0


REPORT = Reporter()


class Case:
    """One permutation of chart values, plus assertions against its render."""

    def __init__(self, group, name, sets=(), kube=DEFAULT_KUBE, show=DS_TEMPLATE,
                 chart=None, xfail=None):
        self.group = group
        self.name = name
        self.sets = list(sets)
        self.kube = kube
        self.xfail = xfail
        self.skipped = not REPORT.selected(group, name)
        self.render = (_EMPTY_RENDER if self.skipped
                       else render(self.sets, kube=kube, chart=chart, show=show))
        self._passed = True
        if not self.skipped:
            REPORT.cases += 1
            if REPORT.verbose:
                print(f"  . [{group}] {name}")

    # -- assertions ---------------------------------------------------------

    def _fail(self, what, lines):
        if self.skipped:
            return
        self._passed = False
        REPORT.record(self, what, lines)

    def _ok(self):
        if not self.skipped:
            REPORT.checks += 1

    def eq(self, what, actual, expected):
        self._ok()
        if actual != expected:
            self._fail(what, [f"expected: {fmt(expected)}", f"actual:   {fmt(actual)}"]
                       + explain(expected, actual))
            return False
        return True

    def truthy(self, what, cond, detail=""):
        self._ok()
        if not cond:
            self._fail(what, [detail or "condition not met"])
            return False
        return True

    def rendered(self) -> bool:
        self._ok()
        if self.skipped:
            return False
        if not self.render.ok:
            self._fail("template must render", ["helm failed:", f"  {self.render.error}"])
            return False
        return True

    def render_fails_with(self, what, *needles):
        """Assert the render is rejected and the message is actually useful."""
        self._ok()
        if self.skipped:
            return
        if self.render.ok:
            self._fail(what, ["expected the render to be rejected, but it succeeded"])
            return
        msg = self.render.error
        missing = [n for n in needles if n.lower() not in msg.lower()]
        if missing:
            self._fail(what, [
                "the render was rejected (good) but the message is not the expected one",
                f"missing from message: {missing}",
                f"actual message: {msg}",
            ])

    # -- accessors ----------------------------------------------------------

    def daemonset(self):
        if not self.rendered():
            return None
        ds = self.render.kind("DaemonSet", "odiglet")
        if ds is None:
            self._fail("odiglet DaemonSet must be rendered", ["no DaemonSet named 'odiglet' in the output"])
        return ds

    def pod_spec(self):
        ds = self.daemonset()
        return None if ds is None else ds["spec"]["template"]["spec"]

    def all_containers(self) -> dict:
        spec = self.pod_spec()
        if spec is None:
            return {}
        out = {}
        for c in list(spec.get("initContainers") or []) + list(spec.get("containers") or []):
            out[c["name"]] = c
        return out

    def container(self, name, required=True):
        containers = self.all_containers()
        c = containers.get(name)
        if c is None and required:
            self._ok()
            self._fail(f"container {name!r} must exist",
                       [f"containers rendered: {sorted(containers)}"])
        return c

    def security_context(self, container_name, expected):
        c = self.container(container_name, required=expected is not MISSING)
        if c is None:
            return
        actual = c.get("securityContext", MISSING)
        self.eq(f"{container_name} container securityContext", actual, expected)

    def pod_security_context(self, expected):
        spec = self.pod_spec()
        if spec is None:
            return
        self.eq("pod securityContext", spec.get("securityContext", MISSING), expected)

    def container_absent(self, name):
        containers = self.all_containers()
        self._ok()
        if name in containers:
            self._fail(f"container {name!r} must NOT be rendered",
                       [f"containers rendered: {sorted(containers)}"])

    def mount(self, container_name, mount_path):
        c = self.container(container_name)
        if c is None:
            return MISSING
        for m in c.get("volumeMounts") or []:
            if m["mountPath"] == mount_path:
                return m
        return MISSING

    def mount_read_only(self, container_name, mount_path):
        m = self.mount(container_name, mount_path)
        return MISSING if m is MISSING else m.get("readOnly", MISSING)

    def volume(self, name):
        spec = self.pod_spec()
        for v in (spec.get("volumes") or []) if spec else []:
            if v["name"] == name:
                return v
        return MISSING

    def odigos_config(self) -> dict:
        """The parsed config.yaml of the odigos-configuration ConfigMap."""
        if not self.rendered():
            return {}
        cm = self.render.kind("ConfigMap", "odigos-configuration")
        if cm is None:
            self._fail("odigos-configuration ConfigMap must be rendered",
                       ["no ConfigMap named 'odigos-configuration' in the output"])
            return {}
        return yaml.safe_load(cm.get("data", {}).get("config.yaml") or "{}") or {}

    def volume_names(self):
        spec = self.pod_spec()
        return sorted(v["name"] for v in (spec.get("volumes") or [])) if spec else []

    def all_security_contexts(self):
        """Every container's securityContext, init containers included."""
        spec = self.pod_spec() or {}
        return [c.get("securityContext") or {}
                for key in ("initContainers", "containers")
                for c in (spec.get(key) or [])]

    def env(self, container_name, key):
        c = self.container(container_name)
        if c is None:
            return MISSING
        for e in c.get("env") or []:
            if e["name"] == key:
                return e.get("value", MISSING)
        return MISSING

    def args(self, container_name):
        c = self.container(container_name)
        return list(c.get("args") or []) if c else []

    def finish(self):
        """Flip an xfail case that unexpectedly passed into a failure."""
        if self.xfail and self._passed and not self.skipped:
            REPORT.unexpected_passes.append((self, "all assertions", self.xfail))
        return self


# ---------------------------------------------------------------------------
# expected security contexts
# ---------------------------------------------------------------------------

SC_PRIVILEGED = {"privileged": True}
SC_DROP_ALL = {
    "privileged": False,
    "allowPrivilegeEscalation": False,
    "capabilities": {"drop": ["ALL"]},
}

# Same, plus the uid a profile names so that nothing in the daemonset runs as
# root - even a container that runs no code of its own.
SC_DROP_ALL_AS_USER = dict(SC_DROP_ALL, runAsUser=RUN_AS)


def sc_deviceplugin_profile():
    """The device plugin under a profile: non-root, and able to bind its socket
    under the kubelet's root-owned directory."""
    return {
        "privileged": False,
        "allowPrivilegeEscalation": False,
        "runAsUser": RUN_AS,
        "capabilities": {"drop": ["ALL"], "add": ["DAC_OVERRIDE"]},
    }


def sc_odiglet_profile(caps=None, apparmor=True):
    """Expected odiglet securityContext under an unprivileged profile.

    apparmor=False for k8s < 1.30, where the profile is set through a pod
    annotation instead of a securityContext field.
    """
    sc = {
        "privileged": False,
        "runAsUser": RUN_AS,
        "runAsGroup": RUN_AS,
        # Deliberate in the chart: a uid change at exec clears the permitted set,
        # so the binary's file capabilities are the only source of privilege and
        # no_new_privs must stay off for the kernel to honour them.
        "allowPrivilegeEscalation": True,
        "capabilities": {"add": list(caps or PROFILE_ODIGLET_CAPS), "drop": ["ALL"]},
    }
    if apparmor:
        sc["appArmorProfile"] = {"type": "Unconfined"}
    return sc


def sc_data_collection_profile(caps=None):
    sc = {
        "privileged": False,
        "runAsUser": RUN_AS,
        "runAsGroup": RUN_AS,
        "allowPrivilegeEscalation": False,
        "capabilities": {"drop": ["ALL"]},
    }
    if caps:
        sc["capabilities"] = {"add": list(caps), "drop": ["ALL"]}
    return sc


def sc_unprivileged_legacy(caps=None, apparmor=True):
    """odiglet.unPrivileged=true: no uid change, no allowPrivilegeEscalation."""
    sc = {
        "privileged": False,
        "capabilities": {"add": list(caps or LEGACY_ODIGLET_CAPS), "drop": ["ALL"]},
    }
    if apparmor:
        sc["appArmorProfile"] = {"type": "Unconfined"}
    return sc


# ---------------------------------------------------------------------------
# tests
# ---------------------------------------------------------------------------

TESTS = []


def test(fn):
    TESTS.append(fn)
    return fn


@test
def profiles():
    """Each profile renders the expected securityContext for every container."""
    g = "profiles"

    # ---- legacy (the chart default, default signals -> logs+metrics) ----
    c = Case(g, "legacy (default values)")
    c.pod_security_context(MISSING)
    c.security_context("init", SC_PRIVILEGED)
    c.security_context("odiglet", SC_PRIVILEGED)
    c.security_context("data-collection", SC_PRIVILEGED)
    c.security_context("deviceplugin", SC_DROP_ALL)

    # legacy profile, traces only: data-collection no longer needs privileged
    c = Case(g, "legacy + traces only", TRACES_ONLY)
    c.pod_security_context(MISSING)
    c.security_context("init", SC_PRIVILEGED)
    c.security_context("odiglet", SC_PRIVILEGED)
    c.security_context("data-collection", {
        "privileged": False,
        "capabilities": {"add": LEGACY_DATA_COLLECTION_CAPS, "drop": ["ALL"]},
    })
    c.security_context("deviceplugin", SC_DROP_ALL)

    # explicit legacy is the same as the default
    c = Case(g, "securityProfile=legacy explicitly", PROFILE_LEGACY)
    c.security_context("odiglet", SC_PRIVILEGED)
    c.security_context("data-collection", SC_PRIVILEGED)
    c.security_context("deviceplugin", SC_DROP_ALL)
    c.pod_security_context(MISSING)

    # legacy needs no enterprise token, with or without the value spelled out
    c = Case(g, "securityProfile=legacy needs no enterprise token", PROFILE_LEGACY)
    c.rendered()

    # ---- unprivileged ----
    c = Case(g, "securityProfile=unprivileged", UNPRIV_BASE + PROFILE_UNPRIV)
    c.pod_security_context({"fsGroup": RUN_AS})
    # the standard init container is replaced by the image-pull one
    c.container_absent("init")
    c.security_context("odigos-agents-image-pull", SC_DROP_ALL_AS_USER)
    c.security_context("odiglet", sc_odiglet_profile())
    c.security_context("data-collection", sc_data_collection_profile())
    # this case pins mountMethod=k8s-init-container, which renders neither
    c.container_absent("deviceplugin")
    c.container_absent("csi-driver")

    # unprivileged with the host mounts kept: every container still non-root,
    # nothing privileged, and the device plugin renders as it always has.
    c = Case(g, "securityProfile=unprivileged with host mounts",
             TRACES_ONLY + ENTERPRISE + PROFILE_UNPRIV)
    c.pod_security_context({"fsGroup": RUN_AS})
    c.security_context("init", {
        "runAsUser": RUN_AS,
        "allowPrivilegeEscalation": False,
        "capabilities": {"drop": ["ALL"], "add": ["DAC_OVERRIDE"]},
    })
    c.security_context("odiglet", sc_odiglet_profile())
    c.security_context("data-collection", sc_data_collection_profile())
    c.security_context("deviceplugin", sc_deviceplugin_profile())
    c.truthy("nothing is privileged", "privileged: true" not in c.render.stdout,
             "'privileged: true' appears in the rendered daemonset")
    c.truthy("no container runs as root",
             all(sc.get("runAsUser") == RUN_AS for sc in c.all_security_contexts()),
             f"security contexts: {c.all_security_contexts()}")
    c.truthy("the agent directory is still mounted", "odigos" in c.volume_names(),
             f"volumes: {c.volume_names()}")

    # noHostPathMounts is the addon that narrows it further: with no root-owned
    # hostPath to write, CAP_DAC_OVERRIDE is not granted.
    c = Case(g, "securityProfile=unprivileged + noHostPathMounts drops DAC_OVERRIDE",
             UNPRIV_BASE + PROFILE_UNPRIV + ["--set", "odiglet.noHostPathMounts=true"])
    c.security_context("odiglet", sc_odiglet_profile(caps=PROFILE_ODIGLET_CAPS_NO_HOSTPATH))
    c.truthy("the agent directory is not mounted", "odigos" not in c.volume_names(),
             f"volumes: {c.volume_names()}")
    c.truthy("odiglet container must not be privileged anywhere",
             "privileged: true" not in c.render.stdout,
             "'privileged: true' appears in the rendered daemonset")

    # the same, reached through the external-secret form of the enterprise gate
    c = Case(g, "securityProfile=unprivileged (externalOnpremTokenSecret)",
             TRACES_ONLY + MM_INIT + ENTERPRISE_EXTERNAL_SECRET + PROFILE_UNPRIV)
    c.pod_security_context({"fsGroup": RUN_AS})
    c.security_context("odiglet", sc_odiglet_profile())
    c.security_context("data-collection", sc_data_collection_profile())
    c.truthy("nothing is privileged", "privileged: true" not in c.render.stdout,
             "'privileged: true' appears in the rendered daemonset")


@test
def tracefs_host_path():
    """odiglet.tracefsHostPath drives both the hostPath and every mountPath."""
    g = "tracefs"
    override = "/run/tracing"

    profiles = (
        ("legacy", []),
        ("unprivileged", UNPRIV_BASE + PROFILE_UNPRIV),
        ("legacy + unPrivileged=true", TRACES_ONLY + ["--set", "odiglet.unPrivileged=true"]),
    )

    # The parent directory rather than /sys/kernel/tracing, because it is a
    # recursive bind: the tracefs submount comes with it whether the node
    # mounts tracefs at /sys/kernel/tracing or only under
    # /sys/kernel/debug/tracing, so one value works on both layouts.
    for label, sets in profiles:
        c = Case(g, f"sys-kernel is mounted ({label})", sets)
        c.eq("sys-kernel hostPath", c.volume("sys-kernel"),
             {"name": "sys-kernel", "hostPath": {"path": TRACEFS_DEFAULT}})
        c.eq("odiglet mount", c.mount("odiglet", TRACEFS_DEFAULT),
             {"name": "sys-kernel", "mountPath": TRACEFS_DEFAULT})
        c.eq("data-collection mount", c.mount("data-collection", TRACEFS_DEFAULT),
             {"name": "sys-kernel", "mountPath": TRACEFS_DEFAULT, "readOnly": True})

        # A node that keeps its tracing filesystem somewhere neither of the two
        # usual layouts covers has to be able to say so.
        c = Case(g, f"tracefsHostPath={override} ({label})",
                 sets + ["--set", f"odiglet.tracefsHostPath={override}"])
        c.eq("sys-kernel hostPath", c.volume("sys-kernel"),
             {"name": "sys-kernel", "hostPath": {"path": override}})
        c.eq("odiglet mount", c.mount("odiglet", override),
             {"name": "sys-kernel", "mountPath": override})
        c.eq("data-collection mount", c.mount("data-collection", override),
             {"name": "sys-kernel", "mountPath": override, "readOnly": True})
        c.eq("the default path must not be mounted any more",
             c.mount("odiglet", TRACEFS_DEFAULT), MISSING)

    # an empty value must fall back to the parent directory, not render an empty
    # hostPath, which the API server rejects
    c = Case(g, "empty tracefsHostPath falls back to /sys/kernel",
             ["--set", "odiglet.tracefsHostPath="])
    c.eq("sys-kernel hostPath", c.volume("sys-kernel"),
         {"name": "sys-kernel", "hostPath": {"path": TRACEFS_DEFAULT}})

    # the odiglet needs to write to tracefs: the mount is never read-only,
    # in either profile.  (unprivileged-strict, which mounted it read-only,
    # is gone.)
    for label, sets in profiles:
        c = Case(g, f"odiglet tracefs mount is writable ({label})", sets)
        c.eq("odiglet tracefs readOnly", c.mount_read_only("odiglet", TRACEFS_DEFAULT), MISSING)


@test
def legacy_unprivileged_switch():
    """odiglet.unPrivileged=true keeps its historical meaning, profile or not."""
    g = "legacy-unPrivileged"

    # REGRESSION 1: this permutation once stopped rendering at all.
    # REGRESSION 3: it once started adding runAsUser/allowPrivilegeEscalation,
    # which breaks images without file capabilities.
    c = Case(g, "unPrivileged=true alone", TRACES_ONLY + ["--set", "odiglet.unPrivileged=true"])
    c.pod_security_context(MISSING)
    c.security_context("odiglet", sc_unprivileged_legacy())
    c.security_context("data-collection", {
        "privileged": False,
        "capabilities": {"add": LEGACY_DATA_COLLECTION_CAPS, "drop": ["ALL"]},
    })
    c.security_context("deviceplugin", SC_DROP_ALL)
    c.truthy("hostPath mounts are kept (unPrivileged never dropped them)",
             "odigos" in c.volume_names() and "run-dir" in c.volume_names(),
             f"volumes: {c.volume_names()}")
    for key in ("runAsUser", "runAsGroup", "allowPrivilegeEscalation"):
        c.eq(f"odiglet securityContext must not gain {key}",
             (c.container('odiglet') or {}).get("securityContext", {}).get(key, MISSING), MISSING)

    # unPrivileged=true on top of the profile must not re-privilege anything
    c = Case(g, "unPrivileged=true + securityProfile=unprivileged",
             UNPRIV_BASE + PROFILE_UNPRIV + ["--set", "odiglet.unPrivileged=true"])
    c.security_context("odiglet", sc_odiglet_profile())
    c.security_context("data-collection", sc_data_collection_profile())
    c.truthy("nothing is privileged", "privileged: true" not in c.render.stdout,
             "'privileged: true' appears in the rendered daemonset")

    # unPrivileged=true is not the enterprise-gated profile: it needs no token
    c = Case(g, "unPrivileged=true needs no enterprise token",
             TRACES_ONLY + ["--set", "odiglet.unPrivileged=true"])
    c.rendered()


@test
def capability_overrides():
    """An explicit capabilities list always wins over the profile default."""
    g = "capabilities"
    odiglet_override = ["--set-json", 'odiglet.odiglet.capabilities=["NET_ADMIN","SYS_NICE"]']
    dc_override = ["--set-json", 'odiglet.dataCollection.capabilities=["SYS_CHROOT"]']

    # REGRESSION 2: an explicit capabilities list was silently ignored under a profile.
    c = Case(g, "odiglet.odiglet.capabilities overrides securityProfile=unprivileged",
             UNPRIV_BASE + PROFILE_UNPRIV + odiglet_override)
    c.security_context("odiglet", sc_odiglet_profile(caps=["NET_ADMIN", "SYS_NICE"]))

    c = Case(g, "odiglet.dataCollection.capabilities overrides securityProfile=unprivileged",
             UNPRIV_BASE + PROFILE_UNPRIV + dc_override)
    c.security_context("data-collection", sc_data_collection_profile(caps=["SYS_CHROOT"]))

    c = Case(g, "both capability lists override securityProfile=unprivileged",
             UNPRIV_BASE + PROFILE_UNPRIV + odiglet_override + dc_override)
    c.security_context("odiglet", sc_odiglet_profile(caps=["NET_ADMIN", "SYS_NICE"]))
    c.security_context("data-collection", sc_data_collection_profile(caps=["SYS_CHROOT"]))

    # and over the legacy switch
    c = Case(g, "odiglet.odiglet.capabilities overrides unPrivileged=true",
             TRACES_ONLY + ["--set", "odiglet.unPrivileged=true"] + odiglet_override)
    c.security_context("odiglet", sc_unprivileged_legacy(caps=["NET_ADMIN", "SYS_NICE"]))

    c = Case(g, "odiglet.dataCollection.capabilities overrides unPrivileged=true",
             TRACES_ONLY + ["--set", "odiglet.unPrivileged=true"] + dc_override)
    c.security_context("data-collection", {
        "privileged": False,
        "capabilities": {"add": ["SYS_CHROOT"], "drop": ["ALL"]},
    })


@test
def apparmor():
    """odiglet.odiglet.appArmorProfile applies on both k8s API shapes."""
    g = "apparmor"
    override = ["--set", "odiglet.odiglet.appArmorProfile.type=RuntimeDefault"]

    for label, sets in (
        ("unPrivileged=true", TRACES_ONLY + ["--set", "odiglet.unPrivileged=true"]),
        ("securityProfile=unprivileged", UNPRIV_BASE + PROFILE_UNPRIV),
    ):
        c = Case(g, f"default Unconfined profile ({label}) on k8s >= 1.30", sets)
        sc = (c.container("odiglet") or {}).get("securityContext", {})
        c.eq("odiglet appArmorProfile", sc.get("appArmorProfile", MISSING), {"type": "Unconfined"})

        c = Case(g, f"appArmorProfile override ({label}) on k8s >= 1.30", sets + override)
        sc = (c.container("odiglet") or {}).get("securityContext", {})
        c.eq("odiglet appArmorProfile", sc.get("appArmorProfile", MISSING), {"type": "RuntimeDefault"})

        c = Case(g, f"annotation form ({label}) on k8s < 1.30", sets, kube="1.29.0")
        ds = c.daemonset()
        ann = (ds or {})["spec"]["template"]["metadata"].get("annotations", {}) if ds else {}
        c.eq("apparmor annotation",
             ann.get("container.apparmor.security.beta.kubernetes.io/odiglet", MISSING), "unconfined")
        sc = (c.container("odiglet") or {}).get("securityContext", {})
        c.eq("no appArmorProfile field before k8s 1.30", sc.get("appArmorProfile", MISSING), MISSING)

    # legacy mode: no apparmor relaxation at all
    c = Case(g, "legacy has no apparmor profile / annotation")
    sc = (c.container("odiglet") or {}).get("securityContext", {})
    c.eq("odiglet appArmorProfile", sc.get("appArmorProfile", MISSING), MISSING)
    ds = c.daemonset()
    ann = (ds or {})["spec"]["template"]["metadata"].get("annotations", {}) if ds else {}
    c.eq("no apparmor annotation",
         ann.get("container.apparmor.security.beta.kubernetes.io/odiglet", MISSING), MISSING)


@test
def host_switches():
    """noHostPathMounts / noHostPid / noHostNetwork still do what they did."""
    g = "host-switches"

    c = Case(g, "noHostPathMounts=true drops the optional hostPath volumes",
             MM_INIT + ["--set", "odiglet.noHostPathMounts=true"])
    vols = c.volume_names()
    for name in ("odigos", "run-dir", "host-cgroup", "odigos-opamp-exchange", "device-plugins-dir"):
        c.truthy(f"volume {name!r} must be gone", name not in vols, f"volumes: {vols}")
    c.truthy("sys-kernel is still mounted", "sys-kernel" in vols, f"volumes: {vols}")
    c.eq("ODIGOS_CGROUP_ROOT env must be gone", c.env("odiglet", "ODIGOS_CGROUP_ROOT"), MISSING)
    c.security_context("odiglet", SC_PRIVILEGED)  # unrelated to privilege

    c = Case(g, "default keeps the hostPath volumes")
    vols = c.volume_names()
    for name in ("odigos", "run-dir", "host-cgroup"):
        c.truthy(f"volume {name!r} must be present", name in vols, f"volumes: {vols}")
    c.eq("ODIGOS_CGROUP_ROOT env", c.env("odiglet", "ODIGOS_CGROUP_ROOT"), "/host/sys/fs/cgroup")

    c = Case(g, "noHostPid=true drops hostPID and mounts /proc",
             ["--set", "odiglet.noHostPid=true"])
    spec = c.pod_spec() or {}
    c.eq("hostPID", spec.get("hostPID", MISSING), MISSING)
    c.eq("ODIGOS_PROC_DIR env", c.env("odiglet", "ODIGOS_PROC_DIR"), "/hostproc")
    c.eq("/hostproc mount", c.mount("odiglet", "/hostproc"), {"name": "hostproc", "mountPath": "/hostproc"})
    c.truthy("hostproc volume", "hostproc" in c.volume_names(), f"volumes: {c.volume_names()}")

    c = Case(g, "hostPID is on by default")
    c.eq("hostPID", (c.pod_spec() or {}).get("hostPID", MISSING), True)
    c.eq("ODIGOS_PROC_DIR env", c.env("odiglet", "ODIGOS_PROC_DIR"), MISSING)

    # noHostPid also has to work under a profile
    c = Case(g, "noHostPid=true under securityProfile=unprivileged",
             UNPRIV_BASE + PROFILE_UNPRIV + ["--set", "odiglet.noHostPid=true"])
    c.eq("hostPID", (c.pod_spec() or {}).get("hostPID", MISSING), MISSING)
    c.eq("ODIGOS_PROC_DIR env", c.env("odiglet", "ODIGOS_PROC_DIR"), "/hostproc")

    # hostNetwork is only added before k8s 1.26 (or for network metrics)
    c = Case(g, "hostNetwork on k8s 1.25 by default", kube="1.25.0")
    c.eq("hostNetwork", (c.pod_spec() or {}).get("hostNetwork", MISSING), True)

    c = Case(g, "noHostNetwork=true removes hostNetwork on k8s 1.25",
             ["--set", "odiglet.noHostNetwork=true"], kube="1.25.0")
    c.eq("hostNetwork", (c.pod_spec() or {}).get("hostNetwork", MISSING), MISSING)

    c = Case(g, "no hostNetwork on k8s 1.30")
    c.eq("hostNetwork", (c.pod_spec() or {}).get("hostNetwork", MISSING), MISSING)

    c = Case(g, "networkMetrics forces hostNetwork on k8s 1.30",
             ["--set", "metricsSources.networkMetrics.enabled=true"])
    c.eq("hostNetwork", (c.pod_spec() or {}).get("hostNetwork", MISSING), True)

    c = Case(g, "noHostNetwork=true under securityProfile=unprivileged (k8s 1.25)",
             UNPRIV_BASE + PROFILE_UNPRIV + ["--set", "odiglet.noHostNetwork=true"], kube="1.25.0")
    c.eq("hostNetwork", (c.pod_spec() or {}).get("hostNetwork", MISSING), MISSING)
    # k8s < 1.30 carries the apparmor profile in an annotation, not the securityContext
    c.security_context("odiglet", sc_odiglet_profile(apparmor=False))


@test
def mount_methods():
    """Every instrumentor.mountMethod value keeps its container layout."""
    g = "mountMethod"
    expected_containers = {
        "": {"init", "odiglet", "data-collection", "deviceplugin"},
        "k8s-virtual-device": {"init", "odiglet", "data-collection", "deviceplugin"},
        "k8s-host-path": {"init", "odiglet", "data-collection"},
        "k8s-init-container": {"odigos-agents-image-pull", "odiglet", "data-collection"},
        "k8s-csi-driver": {"init", "odiglet", "data-collection", "csi-driver"},
    }
    for method, expected in expected_containers.items():
        sets = ["--set", f"instrumentor.mountMethod={method}"] if method else []
        c = Case(g, f"mountMethod={method or '<empty>'} (legacy)", sets)
        c.eq("containers", sorted(c.all_containers()), sorted(expected))
        c.security_context("odiglet", SC_PRIVILEGED)
        if "deviceplugin" in expected:
            c.security_context("deviceplugin", SC_DROP_ALL)
        if "csi-driver" in expected:
            c.security_context("csi-driver", SC_PRIVILEGED)
        if method == "k8s-init-container":
            # privileged mode leaves the image-pull container without a securityContext
            c.security_context("odigos-agents-image-pull", MISSING)

    # unprivileged legacy switch keeps working with every host-mount method
    for method in ("", "k8s-virtual-device", "k8s-host-path", "k8s-init-container"):
        sets = TRACES_ONLY + ["--set", "odiglet.unPrivileged=true"]
        if method:
            sets += ["--set", f"instrumentor.mountMethod={method}"]
        c = Case(g, f"unPrivileged=true + mountMethod={method or '<empty>'}", sets)
        c.security_context("odiglet", sc_unprivileged_legacy())

    # the profiles only accept k8s-init-container
    c = Case(g, "securityProfile=unprivileged + mountMethod=k8s-init-container",
             UNPRIV_BASE + PROFILE_UNPRIV)
    c.eq("containers", sorted(c.all_containers()),
         ["data-collection", "odiglet", "odigos-agents-image-pull"])


@test
def signals():
    """logs / metrics keep forcing a privileged data-collection container."""
    g = "signals"
    matrix = {
        '["traces"]': (False, set()),
        '["traces","logs"]': (True, {"varlog", "varlibdockercontainers"}),
        '["traces","metrics"]': (True, {"hostfs"}),
        '["traces","logs","metrics"]': (True, {"varlog", "varlibdockercontainers", "hostfs"}),
    }
    for sig, (privileged, vols) in matrix.items():
        c = Case(g, f"signals={sig}", ["--set-json", f"signals={sig}"])
        if privileged:
            c.security_context("data-collection", SC_PRIVILEGED)
        else:
            c.security_context("data-collection", {
                "privileged": False,
                "capabilities": {"add": LEGACY_DATA_COLLECTION_CAPS, "drop": ["ALL"]},
            })
        got = set(c.volume_names())
        for v in vols:
            c.truthy(f"volume {v!r} present for {sig}", v in got, f"volumes: {sorted(got)}")

    c = Case(g, "metrics with hostMetrics disabled drops /hostfs",
             ["--set-json", 'signals=["traces","metrics"]',
              "--set", "metricsSources.hostMetrics.disabled=true"])
    c.truthy("hostfs volume must be gone", "hostfs" not in c.volume_names(),
             f"volumes: {c.volume_names()}")
    c.eq("/hostfs mount must be gone", c.mount("data-collection", "/hostfs"), MISSING)


@test
def platform_switches():
    """openshift.enabled and gke.autopilot still take effect, profile or not."""
    g = "platform"

    c = Case(g, "openshift.enabled=true adds host + selinux mounts",
             ["--set", "openshift.enabled=true"])
    vols = c.volume_names()
    for v in ("host", "selinux"):
        c.truthy(f"volume {v!r} present", v in vols, f"volumes: {vols}")
    c.eq("odiglet /host mount", c.mount("odiglet", "/host"),
         {"name": "host", "mountPath": "/host", "readOnly": True})
    c.eq("odiglet /host/etc/selinux mount", c.mount("odiglet", "/host/etc/selinux"),
         {"name": "selinux", "mountPath": "/host/etc/selinux", "mountPropagation": "Bidirectional"})
    c.eq("init /host mount", c.mount("init", "/host"),
         {"name": "host", "mountPath": "/host", "readOnly": True})

    c = Case(g, "openshift off by default")
    c.truthy("no host/selinux volumes",
             "host" not in c.volume_names() and "selinux" not in c.volume_names(),
             f"volumes: {c.volume_names()}")

    # openshift under a profile: the profile is about privilege, not about host
    # mounts, so openshift keeps the mounts it has always had.
    c = Case(g, "openshift.enabled=true + securityProfile=unprivileged",
             UNPRIV_BASE + PROFILE_UNPRIV + ["--set", "openshift.enabled=true"])
    c.security_context("odiglet", sc_odiglet_profile())
    c.truthy("host/selinux volumes are kept",
             "host" in c.volume_names() and "selinux" in c.volume_names(),
             f"volumes: {c.volume_names()}")

    # ...and noHostPathMounts is what takes them away, profile or not.
    c = Case(g, "openshift.enabled=true + securityProfile=unprivileged + noHostPathMounts",
             UNPRIV_BASE + PROFILE_UNPRIV
             + ["--set", "openshift.enabled=true", "--set", "odiglet.noHostPathMounts=true"])
    c.truthy("no host/selinux volumes",
             "host" not in c.volume_names() and "selinux" not in c.volume_names(),
             f"volumes: {c.volume_names()}")

    c = Case(g, "gke.autopilot=true sets GKE_AUTOPILOT on the init container",
             ["--set", "gke.autopilot=true"])
    c.eq("init GKE_AUTOPILOT env", c.env("init", "GKE_AUTOPILOT"), "true")

    c = Case(g, "gke.autopilot off by default")
    c.eq("init GKE_AUTOPILOT env", c.env("init", "GKE_AUTOPILOT"), MISSING)

    c = Case(g, "gke.autopilot=true + unPrivileged=true",
             TRACES_ONLY + ["--set", "gke.autopilot=true", "--set", "odiglet.unPrivileged=true"])
    c.eq("init GKE_AUTOPILOT env", c.env("init", "GKE_AUTOPILOT"), "true")
    c.security_context("odiglet", sc_unprivileged_legacy())


@test
def wasp():
    """wasp turns on automatically for the unprivileged profile only."""
    g = "wasp"
    c = Case(g, "--wasp-enabled arg for securityProfile=unprivileged", UNPRIV_BASE + PROFILE_UNPRIV)
    c.truthy("odiglet args contain --wasp-enabled", "--wasp-enabled" in c.args("odiglet"),
             f"args: {c.args('odiglet')}")

    c = Case(g, "waspEnabled config key for securityProfile=unprivileged",
             UNPRIV_BASE + PROFILE_UNPRIV, show=CM_TEMPLATE)
    c.eq("waspEnabled in odigos-configuration", c.odigos_config().get("waspEnabled", MISSING), True)

    c = Case(g, "no --wasp-enabled under the legacy profile")
    c.truthy("odiglet args have no --wasp-enabled", "--wasp-enabled" not in c.args("odiglet"),
             f"args: {c.args('odiglet')}")

    c = Case(g, "no --wasp-enabled for the legacy unPrivileged=true",
             TRACES_ONLY + ["--set", "odiglet.unPrivileged=true"])
    c.truthy("odiglet args have no --wasp-enabled", "--wasp-enabled" not in c.args("odiglet"),
             f"args: {c.args('odiglet')}")

    c = Case(g, "no waspEnabled key under the legacy profile", show=CM_TEMPLATE)
    c.eq("waspEnabled in odigos-configuration", c.odigos_config().get("waspEnabled", MISSING), MISSING)

    # an enterprise token on its own must not turn wasp on
    c = Case(g, "no --wasp-enabled for legacy with an enterprise token", ENTERPRISE)
    c.truthy("odiglet args have no --wasp-enabled", "--wasp-enabled" not in c.args("odiglet"),
             f"args: {c.args('odiglet')}")

    c = Case(g, "no waspEnabled key for legacy with an enterprise token",
             ENTERPRISE, show=CM_TEMPLATE)
    c.eq("waspEnabled in odigos-configuration", c.odigos_config().get("waspEnabled", MISSING), MISSING)

    c = Case(g, "wasp.enabled=true works on its own under legacy",
             ["--set", "wasp.enabled=true"])
    c.truthy("odiglet args contain --wasp-enabled", "--wasp-enabled" in c.args("odiglet"),
             f"args: {c.args('odiglet')}")

    c = Case(g, "wasp.enabled=true config key under legacy",
             ["--set", "wasp.enabled=true"], show=CM_TEMPLATE)
    c.eq("waspEnabled in odigos-configuration", c.odigos_config().get("waspEnabled", MISSING), True)


@test
def guards():
    """Every guard must fire, and say something actionable."""
    g = "guards"
    logs_msg = "privileged mode is required when collecting logs or metrics"
    hostpath_msg = "mount method must be k8s-init-container"
    enterprise_msg = "requires Odigos Enterprise"

    # ---- the enterprise gate ----
    c = Case(g, "securityProfile=unprivileged without a token", TRACES_ONLY + MM_INIT + PROFILE_UNPRIV)
    c.render_fails_with("enterprise gate", "odiglet.securityProfile=unprivileged",
                        enterprise_msg, "onPremToken", "securityProfile=legacy")

    c = Case(g, "securityProfile=unprivileged without a token (default values otherwise)",
             PROFILE_UNPRIV)
    c.render_fails_with("enterprise gate fires before the other guards", enterprise_msg)

    c = Case(g, "empty onPremToken does not satisfy the enterprise gate",
             TRACES_ONLY + MM_INIT + PROFILE_UNPRIV + ["--set", "onPremToken="])
    c.render_fails_with("enterprise gate", enterprise_msg)

    c = Case(g, "externalOnpremTokenSecret=false does not satisfy the enterprise gate",
             TRACES_ONLY + MM_INIT + PROFILE_UNPRIV + ["--set", "externalOnpremTokenSecret=false"])
    c.render_fails_with("enterprise gate", enterprise_msg)

    # the gate must not fire for legacy, with or without a token
    for label, sets in (("default values", []), ("explicit", PROFILE_LEGACY),
                        ("with unPrivileged=true", TRACES_ONLY + ["--set", "odiglet.unPrivileged=true"]),
                        ("with noHostPathMounts", MM_INIT + ["--set", "odiglet.noHostPathMounts=true"])):
        c = Case(g, f"legacy needs no enterprise token ({label})", sets)
        c.rendered()
        c.truthy("no enterprise message in the output",
                 enterprise_msg not in c.render.stdout, "output mentions the enterprise gate")

    # and it is satisfied by either form of the token
    for label, token in (("onPremToken", ENTERPRISE),
                         ("externalOnpremTokenSecret=true", ENTERPRISE_EXTERNAL_SECRET)):
        c = Case(g, f"{label} satisfies the enterprise gate",
                 TRACES_ONLY + MM_INIT + token + PROFILE_UNPRIV)
        c.rendered()
        c.security_context("odiglet", sc_odiglet_profile())

    # externalOnpremTokenSecret on its own gets past the profile gate; what stops
    # it is the pre-existing pull-secret validation, which is a different message.
    c = Case(g, "externalOnpremTokenSecret=true alone passes the profile gate",
             TRACES_ONLY + MM_INIT + PROFILE_UNPRIV + ["--set", "externalOnpremTokenSecret=true"])
    c.render_fails_with("pull secret validation, not the profile gate",
                        "no on-prem token is available to the chart")
    c.truthy("the enterprise profile gate must not be what fired",
             enterprise_msg not in c.render.error,
             f"message was: {c.render.error}")

    # ---- logs / metrics ----
    c = Case(g, "securityProfile=unprivileged + default signals (logs+metrics)",
             MM_INIT + ENTERPRISE + PROFILE_UNPRIV)
    c.render_fails_with("logs/metrics guard", logs_msg)

    c = Case(g, "securityProfile=unprivileged + logs",
             MM_INIT + ENTERPRISE + PROFILE_UNPRIV + ["--set-json", 'signals=["traces","logs"]'])
    c.render_fails_with("logs guard", logs_msg)

    c = Case(g, "securityProfile=unprivileged + metrics",
             MM_INIT + ENTERPRISE + PROFILE_UNPRIV + ["--set-json", 'signals=["traces","metrics"]'])
    c.render_fails_with("metrics guard", logs_msg)

    # ---- mount methods ----
    # The profile is about privilege, not about host mounts, so it constrains
    # the mount method only where the method itself needs privilege: the CSI
    # driver, which a pre-existing guard refuses in unprivileged mode.
    for method in ("", "k8s-virtual-device", "k8s-host-path", "k8s-init-container"):
        sets = TRACES_ONLY + ENTERPRISE + PROFILE_UNPRIV
        if method:
            sets += ["--set", f"instrumentor.mountMethod={method}"]
        c = Case(g, f"securityProfile=unprivileged + mountMethod={method or '<empty>'}", sets)
        c.rendered()

    c = Case(g, "securityProfile=unprivileged + mountMethod=k8s-csi-driver",
             TRACES_ONLY + ENTERPRISE + PROFILE_UNPRIV
             + ["--set", "instrumentor.mountMethod=k8s-csi-driver"])
    c.render_fails_with("csi guard", "CSI driver mount method requires privileged mode")

    # noHostPathMounts is what withholds the host mounts, profile or not, and
    # it still requires the mount method that needs none of them.
    c = Case(g, "securityProfile=unprivileged + noHostPathMounts without k8s-init-container",
             TRACES_ONLY + ENTERPRISE + PROFILE_UNPRIV
             + ["--set", "odiglet.noHostPathMounts=true"])
    c.render_fails_with("hostPath guard", hostpath_msg)

    c = Case(g, "legacy unPrivileged=true + logs", ["--set", "odiglet.unPrivileged=true"])
    c.render_fails_with("logs/metrics guard", logs_msg)

    c = Case(g, "noHostPathMounts=true without k8s-init-container",
             ["--set", "odiglet.noHostPathMounts=true"])
    c.render_fails_with("hostPath guard", hostpath_msg)

    c = Case(g, "k8s-csi-driver + unPrivileged=true",
             TRACES_ONLY + ["--set", "instrumentor.mountMethod=k8s-csi-driver",
                            "--set", "odiglet.unPrivileged=true"])
    c.render_fails_with("csi guard", "CSI driver mount method requires privileged mode")

    c = Case(g, "networkMetrics + noHostNetwork",
             ["--set", "metricsSources.networkMetrics.enabled=true",
              "--set", "odiglet.noHostNetwork=true"])
    c.render_fails_with("network metrics guard", "noHostNetwork cannot be true")

    # ---- the schema enum ----
    # "privileged" and "unprivileged-strict" were valid while this feature was in
    # flight; both must now be rejected by name rather than silently ignored.
    for bad in ("privileged", "unprivileged-strict", "unpriviledged", "Unprivileged", "true", ""):
        c = Case(g, f"invalid securityProfile={bad!r}",
                 ENTERPRISE + ["--set", f"odiglet.securityProfile={bad}"])
        c.render_fails_with("securityProfile enum guard", "securityProfile", "legacy", "unprivileged")


@test
def image_paths():
    """Every binary the daemonset runs must be installed by odiglet/Dockerfile.

    The odiglet binary moved from /root to /usr/local/bin so it can be executed
    by a non-root uid; this keeps the chart and the image from drifting apart.
    """
    g = "image-paths"
    dockerfile = REPO / "odiglet" / "Dockerfile"
    if not dockerfile.exists():
        return

    # A uid change at exec clears the permitted set, so the binary's own file
    # capabilities are the only source of privilege for a non-root odiglet.
    # Two properties have to hold, and neither shows up in a rendered manifest.
    setcap = [l for l in dockerfile.read_text().splitlines() if l.startswith("RUN setcap")]
    c = Case(g, "the odiglet file capabilities are usable in every configuration")
    if c.truthy("odiglet/Dockerfile sets file capabilities", len(setcap) == 1,
                f"expected exactly one 'RUN setcap' line, found {len(setcap)}"):
        spec = setcap[0].split()[2]
        caps, _, flags = spec.rpartition("=")
        # Permitted-only. bprm_caps_from_vfs_caps returns -EPERM when the file's
        # effective bit is set and the bounding set grants less than the file
        # asks for, so '=ep' turns any narrowed capability list into a container
        # that cannot start at all.
        c.eq("file capabilities are permitted-only", flags, "p")
        on_binary = {x.strip().upper().removeprefix("CAP_") for x in caps.split(",")}
        # ...and they must cover everything any values file can grant, or the
        # manifest would add a capability the process can never actually hold.
        for label, granted in (
            ("legacy", LEGACY_ODIGLET_CAPS),
            ("profile", PROFILE_ODIGLET_CAPS),
            ("profile + noHostPathMounts", PROFILE_ODIGLET_CAPS_NO_HOSTPATH),
        ):
            c.eq(f"{label} capabilities are all on the binary",
                 sorted(set(granted) - on_binary), [])
    installed = set()
    for line in dockerfile.read_text().splitlines():
        if line.startswith("COPY"):
            dest = line.split()[-1]
            if dest.startswith("/") and not dest.endswith("/"):
                installed.add(dest)

    for label, sets in (
        ("legacy", []),
        ("mountMethod=k8s-csi-driver", ["--set", "instrumentor.mountMethod=k8s-csi-driver"]),
        ("securityProfile=unprivileged", UNPRIV_BASE + PROFILE_UNPRIV),
    ):
        c = Case(g, f"binaries exist in the image ({label})", sets)
        for name, container in sorted(c.all_containers().items()):
            # only the containers that run the odiglet image; data-collection runs
            # the collector image and the pull container runs the agents image.
            if "odiglet" not in container.get("image", ""):
                continue
            paths = list(container.get("command") or [])[:1]
            for probe in ("livenessProbe", "readinessProbe", "startupProbe"):
                exec_cmd = ((container.get(probe) or {}).get("exec") or {}).get("command") or []
                paths += exec_cmd[:1]
            for path in paths:
                if not path.startswith("/"):
                    continue
                c.truthy(
                    f"{name}: {path} must be installed by odiglet/Dockerfile",
                    path in installed,
                    f"odiglet/Dockerfile does not COPY anything to {path}; "
                    f"it installs: {sorted(installed)}")


@test
def schema_guards():
    """values.schema.json must stay usable: helm-schema is easy to mis-drive."""
    g = "schema"

    # Prose placed between the closing `# @schema` fence and the key is consumed
    # by helm-schema as part of the annotation, which once typed this value as
    # `null` and made it unsettable.  Keep descriptions inside `# description: |-`.
    c = Case(g, "odiglet.nodeSelector can still be set",
             ["--set", "odiglet.nodeSelector.kubernetes\\.io/os=linux"])
    if c.rendered():
        c.eq("pod nodeSelector", (c.pod_spec() or {}).get("nodeSelector", MISSING),
             {"kubernetes.io/os": "linux"})

    # the same mistake once invented a property named after a comment line
    schema = json.loads((CHART / "values.schema.json").read_text())
    odiglet = schema["properties"]["odiglet"]["properties"]
    c = Case(g, "no properties invented from comment prose")
    for name, node in (("odiglet.dataCollection", odiglet["dataCollection"]),
                       ("odiglet.odiglet", odiglet["odiglet"]),
                       ("odiglet", schema["properties"]["odiglet"])):
        for key in node.get("properties", {}):
            c.truthy(f"{name}.{key!r} looks like a real property name",
                     " " not in key,
                     f"{name} has a property named {key!r}, which came from a "
                     "comment line that helm-schema read as part of the annotation")
    c.eq("odiglet.dataCollection properties", sorted(odiglet["dataCollection"]["properties"]),
         ["capabilities"])
    c.truthy("odiglet.nodeSelector is not typed as null",
             odiglet["nodeSelector"].get("type") != "null",
             f"odiglet.nodeSelector schema: {json.dumps(odiglet['nodeSelector'])}")
    c.eq("securityProfile enum", odiglet["securityProfile"].get("enum"),
         ["legacy", "unprivileged"])


# ---------------------------------------------------------------------------
# backward-compatibility invariant against the pre-change commit
# ---------------------------------------------------------------------------

# Intentional differences between BASELINE_REF and HEAD.  Anything else is a
# regression.  Each entry rewrites the NEW render back into the OLD shape; the
# rewrite is anchored on the exact rendered text so that any additional change
# inside the same block shows up as a diff instead of being swallowed.
ALLOWED_DIFFS = {
    "odiglet-command-path": (
        # the binary moved out of /root so it can run as a non-root uid
        "            - /usr/local/bin/odiglet\n",
        "            - /root/odiglet\n",
    ),
    "aux-binary-paths": (
        # grpc_health_probe, deviceplugin and the csi driver moved out of /root
        # for the same reason the odiglet binary did: ubi-micro ships /root as
        # 0550, so a non-root container cannot execute anything under it
        "                - /usr/local/bin/grpc_health_probe\n",
        "                - /root/grpc_health_probe\n",
    ),
    "csi-driver-path": (
        "            - /usr/local/bin/odigos-csi-driver\n",
        "            - /root/odigos-csi-driver\n",
    ),
    "deviceplugin-securitycontext": (
        # the deviceplugin container had no securityContext at all before
        "            - /usr/local/bin/deviceplugin\n"
        "          securityContext:\n"
        "            privileged: false\n"
        "            allowPrivilegeEscalation: false\n"
        "            capabilities:\n"
        "              drop:\n"
        "                - ALL\n",
        "            - /root/deviceplugin\n",
    ),
    "image-pull-securitycontext": (
        # only reached with unPrivileged=true + mountMethod=k8s-init-container
        "          imagePullPolicy: IfNotPresent\n"
        "          securityContext:\n"
        "            privileged: false\n"
        "            allowPrivilegeEscalation: false\n"
        "            capabilities:\n"
        "              drop:\n"
        "                - ALL\n"
        "          # # it does not run any actual code, just pulls the image\n",
        "          imagePullPolicy: IfNotPresent\n"
        "          # # it does not run any actual code, just pulls the image\n",
    ),
}

BASELINE_CASES = [
    # (name, sets, kube, allowed diff keys)
    ("no values set", [], "1.30.0", ["odiglet-command-path", "deviceplugin-securitycontext", "aux-binary-paths", "csi-driver-path"]),
    ("no values set (k8s 1.25)", [], "1.25.0", ["odiglet-command-path", "deviceplugin-securitycontext", "aux-binary-paths", "csi-driver-path"]),
    ("unPrivileged=true", TRACES_ONLY + ["--set", "odiglet.unPrivileged=true"], "1.30.0",
     ["odiglet-command-path", "deviceplugin-securitycontext", "aux-binary-paths", "csi-driver-path"]),
    ("unPrivileged=true (k8s 1.29)", TRACES_ONLY + ["--set", "odiglet.unPrivileged=true"], "1.29.0",
     ["odiglet-command-path", "deviceplugin-securitycontext", "aux-binary-paths", "csi-driver-path"]),
    ("unPrivileged=true (k8s 1.25)", TRACES_ONLY + ["--set", "odiglet.unPrivileged=true"], "1.25.0",
     ["odiglet-command-path", "deviceplugin-securitycontext", "aux-binary-paths", "csi-driver-path"]),
    ("noHostPid=true", ["--set", "odiglet.noHostPid=true"], "1.30.0",
     ["odiglet-command-path", "deviceplugin-securitycontext", "aux-binary-paths", "csi-driver-path"]),
    ("openshift.enabled=true", ["--set", "openshift.enabled=true"], "1.30.0",
     ["odiglet-command-path", "deviceplugin-securitycontext", "aux-binary-paths", "csi-driver-path"]),
    ("gke.autopilot=true", ["--set", "gke.autopilot=true"], "1.30.0",
     ["odiglet-command-path", "deviceplugin-securitycontext", "aux-binary-paths", "csi-driver-path"]),
    ("mountMethod=k8s-host-path", ["--set", "instrumentor.mountMethod=k8s-host-path"], "1.30.0",
     ["odiglet-command-path"]),
    ("mountMethod=k8s-csi-driver", ["--set", "instrumentor.mountMethod=k8s-csi-driver"], "1.30.0",
     ["odiglet-command-path", "csi-driver-path", "aux-binary-paths"]),
    ("mountMethod=k8s-init-container", MM_INIT, "1.30.0", ["odiglet-command-path"]),
    ("noHostPathMounts=true", MM_INIT + ["--set", "odiglet.noHostPathMounts=true"], "1.30.0",
     ["odiglet-command-path"]),
    ("signals=[traces,logs]", ["--set-json", 'signals=["traces","logs"]'], "1.30.0",
     ["odiglet-command-path", "deviceplugin-securitycontext", "aux-binary-paths", "csi-driver-path"]),
    ("unPrivileged=true + explicit capabilities",
     TRACES_ONLY + ["--set", "odiglet.unPrivileged=true",
                    "--set-json", 'odiglet.odiglet.capabilities=["NET_ADMIN"]'], "1.30.0",
     ["odiglet-command-path", "deviceplugin-securitycontext", "aux-binary-paths", "csi-driver-path"]),
    ("unPrivileged=true + mountMethod=k8s-init-container",
     TRACES_ONLY + ["--set", "odiglet.unPrivileged=true"] + MM_INIT, "1.30.0",
     ["odiglet-command-path", "image-pull-securitycontext"]),
]


def ensure_baseline_commit() -> bool:
    have = subprocess.run(["git", "-C", str(REPO), "cat-file", "-e", BASELINE_REF + "^{commit}"],
                          capture_output=True)
    if have.returncode == 0:
        return True
    print(f"  fetching baseline commit {BASELINE_REF[:9]} ...")
    subprocess.run(["git", "-C", str(REPO), "fetch", "--depth", "1", "origin", BASELINE_REF],
                   capture_output=True)
    have = subprocess.run(["git", "-C", str(REPO), "cat-file", "-e", BASELINE_REF + "^{commit}"],
                          capture_output=True)
    return have.returncode == 0


def normalize_new(text: str, allowed: list) -> str:
    for key in allowed:
        new_shape, old_shape = ALLOWED_DIFFS[key]
        text = text.replace(new_shape, old_shape)
    return text


def baseline_invariant():
    """The pre-change render must not drift, except where we say it may."""
    g = "baseline"
    if not ensure_baseline_commit():
        REPORT.failures.append(
            f"  [{g}] baseline commit {BASELINE_REF} is not available in this clone.\n"
            "    check out the full history (actions/checkout with fetch-depth: 0) or run\n"
            f"    'git fetch origin {BASELINE_REF}', then re-run.  Do not skip this check.")
        return

    with tempfile.TemporaryDirectory(prefix="odigos-helm-baseline-") as tmp:
        wt = Path(tmp) / "baseline"
        add = subprocess.run(
            ["git", "-C", str(REPO), "worktree", "add", "--detach", str(wt), BASELINE_REF],
            capture_output=True, text=True)
        if add.returncode != 0:
            REPORT.failures.append(
                f"  [{g}] could not create a git worktree for {BASELINE_REF[:9]}:\n"
                f"    {add.stderr.strip()}")
            return
        try:
            old_chart = wt / "helm" / "odigos"
            for name, sets, kube, allowed in BASELINE_CASES:
                if not REPORT.selected(g, name):
                    continue
                REPORT.cases += 1
                REPORT.checks += 1
                if REPORT.verbose:
                    print(f"  . [{g}] {name}")
                old = render(sets, kube=kube, chart=old_chart)
                new = render(sets, kube=kube)
                if not old.ok or not new.ok:
                    REPORT.failures.append(
                        f"  [{g}] {name}\n"
                        f"    check: both {BASELINE_REF[:9]} and HEAD must render\n"
                        f"    {BASELINE_REF[:9]}: {'ok' if old.ok else old.error}\n"
                        f"    HEAD:      {'ok' if new.ok else new.error}\n"
                        f"    render: {new.pretty_cmd()}")
                    continue
                normalized = normalize_new(new.stdout, allowed)
                if normalized == old.stdout:
                    continue
                diff = list(difflib.unified_diff(
                    old.stdout.splitlines(), normalized.splitlines(),
                    fromfile=f"{BASELINE_REF[:9]} (before security profiles)",
                    tofile="HEAD (after whitelisting the intentional differences)",
                    lineterm="", n=2))
                if len(diff) > 80:
                    diff = diff[:80] + [f"... ({len(diff) - 80} more diff lines)"]
                REPORT.failures.append(
                    f"  [{g}] {name}\n"
                    f"    check: rendered odiglet daemonset must be unchanged since "
                    f"{BASELINE_REF[:9]}\n"
                    f"    whitelisted differences: {allowed}\n"
                    "    the diff below is what changed on top of the whitelist - "
                    "either it is a regression,\n"
                    "    or it is intentional and belongs in ALLOWED_DIFFS in "
                    f"{Path(__file__).name}\n"
                    + "\n".join("      " + ln for ln in diff)
                    + f"\n    render: {new.pretty_cmd()}")
        finally:
            subprocess.run(["git", "-C", str(REPO), "worktree", "remove", "--force", str(wt)],
                           capture_output=True)


# ---------------------------------------------------------------------------
# main
# ---------------------------------------------------------------------------


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__,
                                 formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("-v", "--verbose", action="store_true",
                    help="print every permutation as it runs")
    ap.add_argument("-f", "--filter",
                    help="only run cases whose 'group/name' contains this substring")
    ap.add_argument("--skip-baseline", action="store_true",
                    help="skip the diff against the pre-change commit (needs git history)")
    ap.add_argument("--chart", default=None, help="chart directory to test (default: helm/odigos)")
    args = ap.parse_args()

    global CHART
    if args.chart:
        CHART = Path(args.chart).resolve()

    REPORT.verbose = args.verbose
    REPORT.filter = args.filter

    if shutil.which("helm") is None:
        print("error: helm is not installed", file=sys.stderr)
        return 2

    print(f"chart:    {CHART}")
    print(f"baseline: {BASELINE_REF[:9]}")
    if args.filter:
        print(f"filter:   {args.filter!r}")
    for fn in TESTS:
        print(f"* {fn.__name__}: {(fn.__doc__ or '').strip().splitlines()[0]}")
        fn()
    if args.skip_baseline:
        print("* baseline: SKIPPED (--skip-baseline)")
    else:
        print("* baseline: rendered daemonset must not drift from "
              f"{BASELINE_REF[:9]} (except whitelisted differences)")
        baseline_invariant()
    return REPORT.summary()


if __name__ == "__main__":
    sys.exit(main())
