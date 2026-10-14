#!/usr/bin/env bash
#
# rpm-build.sh - build the nsdyndns Fedora RPM from the current branch.
#
# Usage:
#   ./rpm-build.sh        # generate sources, build the RPMs, show result
#   ./rpm-build.sh lint    # rpmlint the built SRPM and RPMs
#
# Conventions:
#   * version = latest git tag (override with NSDYNDNS_VERSION;
#     falls back to the version.go Version variable when the repo has
#     no tags)
#   * release = commits since the last tag + 1 (1 when there are no tags)
#   * the spec's Version: and License: must be up to date; verified here
#   * topdir = ./rpmbuild (local to this checkout)
#
# Requires: go, python3, rpmbuild, xz, git. rpmlint only for lint mode.

set -euo pipefail
cd "$(dirname "$0")"

NAME=nsdyndns
SPEC=nsdyndns.spec

# Version: env override, else latest git tag (v-prefix stripped), else
# the version.go Version variable (this repo currently has no tags).
if [ -n "${NSDYNDNS_VERSION:-}" ]; then
    VERSION="$NSDYNDNS_VERSION"
elif last_tag=$(git describe --tags --abbrev=0 2>/dev/null); then
    VERSION="${last_tag#v}"
else
    VERSION="$(sed -n 's/^var Version = "\(.*\)$/\1/p' version.go)"
fi

RPM_TOP="$(pwd)/rpmbuild"
mkdir -p "${RPM_TOP}"/{SOURCES,SPECS,BUILD,BUILDROOT,SRPMS,RPMS,TMP}

# Release number: commits since the last tag + 1 (1 if no tags).
release_number() {
    if last_tag=$(git describe --tags --abbrev=0 2>/dev/null); then
        echo $(( $(git rev-list --count "${last_tag}..HEAD") + 1 ))
    else
        echo 1
    fi
}

# Print the cumulative SPDX expression the spec's License: must declare
# (project + all vendored modules): every atomic license of each LICENSE
# file (OR/AND flattened), deduplicated, project license first, rest
# sorted, joined with AND. Runs with vendor/ present.
license_expression() {
    python3 - <<'PY'
import re
from pathlib import Path

def atoms(l):
    l = l.replace("(", " ").replace(")", " ")
    return [t.strip() for t in re.split(r"\s+(?:OR|AND)\s+", l) if t.strip()]

def spdx(path: Path) -> str:
    text = path.read_text(errors="replace")
    if "Apache License, Version 2.0" in text:
        return "Apache-2.0"
    if "Redistribution and use in source and binary forms" in text:
        return "BSD-2-Clause"
    if "Permission is hereby granted, free of charge, to any person" in text:
        return "BSD-3-Clause"
    if "MIT License" in text and "Permission is hereby granted" in text:
        return "MIT"
    if re.search(r"GNU\s+GENERAL\s+PUBLIC\s+LICENSE", text) and "Version 3" in text:
        return "GPL-3.0-only"
    raise SystemExit(f"ERROR: unrecognized license file {path}")

files = [Path("LICENSE")] + sorted(Path("vendor").glob("*/LICENSE*"))
out = []
for f in files:
    for t in atoms(spdx(f)):
        if t not in out:
            out.append(t)

# Project license first, rest sorted.
order = [t for t in atoms(spdx(files[0])) if t in out]
order += sorted(t for t in out if t not in order)
print(" AND ".join(order))
PY
}

# Verify the spec's Version:, version.go Version variable, and License:
# are up to date.
check_spec() {
    local current reported gover
    current="$(sed -n 's/^Version:[[:space:]]*//p' "$SPEC")"
    if [[ -z "$current" || "$current" != "$VERSION" ]]; then
        echo "ERROR: spec Version ($current) does not match $VERSION."
        echo "Update 'Version:' in $SPEC."
        return 1
    fi
    gover="$(sed -n 's/^var Version = "\(.*\)$/\1/p' version.go)"
    if [[ "$gover" != "$VERSION" ]]; then
        echo "ERROR: version.go Version ($gover) does not match $VERSION."
        echo "Update 'Version' in version.go."
        return 1
    fi
    reported="$(license_expression)"
    current="$(sed -n 's/^License:[[:space:]]*//p' "$SPEC")"
    if [[ -z "$current" || "$current" != "$reported" ]]; then
        echo "ERROR: License tag out of date."
        echo "  spec:     $current"
        echo "  reported: $reported"
        echo "Update 'License:' in $SPEC."
        return 1
    fi
    echo "OK: spec Version ($VERSION) and License are up to date"
}

do_sources() {
    # Drop artifacts of previous builds (topdir is persistent and shared).
    rm -f "${RPM_TOP}/RPMS"/*/${NAME}-*.rpm \
         "${RPM_TOP}/SRPMS/${NAME}"-*.src.rpm
    cp "$SPEC" "${RPM_TOP}/SPECS/"
    # Source0: archive from HEAD (Fedora-preferred .tar.xz compression).
    # vendor/ is committed, so the archive is fully self-contained.
    git archive --prefix="${NAME}-${VERSION}/" HEAD | xz -f \
        > "${RPM_TOP}/SOURCES/${NAME}-${VERSION}.tar.xz"
    check_spec
    echo "Sources ready in ${RPM_TOP}/SOURCES:"
    ls -l "${RPM_TOP}/SOURCES"
}

do_build() {
    do_sources
    # release_number() must run from the repo (git describe); the rpmbuild
    # subshell has SPECS as its cwd.
    local release
    release="$(release_number)"
    echo "==> Release number: $release (commits since last tag + 1)"
    ( cd "${RPM_TOP}/SPECS" && rpmbuild --define "_topdir ${RPM_TOP}" --define "release $release" -ba "$SPEC" )
}

# Lint the spec (from SPECS) and the built binary RPMs.
do_lint() {
    ( cd "${RPM_TOP}/SPECS" && rpmlint "$SPEC" )
    rpmlint "${RPM_TOP}/RPMS"/*/${NAME}*.rpm
}

echo "==> Building RPM for $NAME v$VERSION"
do_build
echo "==> Done"
ls -lh "${RPM_TOP}"/RPMS/*/${NAME}*.rpm

case "${1:-}" in
    lint) do_lint ;;
    "") ;;
    *) echo "Usage: $0 [lint]" >&2; exit 2 ;;
esac
