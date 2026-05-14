#!/usr/bin/env bash
# audit-presence.sh — verify the sheets/ and detail/ tree is structurally sound.
#
# Two checks:
#   (a) Orphan detail files: every `detail/<cat>/<name>.md` must have a
#       matching `sheets/<cat>/<name>.md` (strict, same category). Pure
#       theory-only details (e.g. cs-theory) are exempted via
#       `.ci/presence-allowlist.txt` entries.
#   (b) Unknown categories: every dir under `sheets/` and `detail/` must
#       appear in `.ci/presence-categories.txt`.
#
# A `## Prerequisites` block check was considered but rejected — most
# prereq bullets are aspirational concept names (`linear-algebra`,
# `queuing-theory`) not backtick-wrapped sheet refs. The convention
# differs from `## See Also`, which `scripts/audit-see-also.sh` already
# validates. Adding noise here would obscure the orphan + category signals.
#
# Exit 0 = clean (or only allowlisted). Exit 1 = real failures.
#
# Usage:
#   scripts/audit-presence.sh
#   scripts/audit-presence.sh --quiet
#   scripts/audit-presence.sh --allowlist=.ci/presence-allowlist.txt
#   scripts/audit-presence.sh --categories=.ci/presence-categories.txt
#
# Allowlist format: one entry per line, "<check>|<path>" — where check is
# one of {orphan, category}.
#
#   orphan|detail/cs-theory/binary-numbering.md
#   category|sheets/wasm

set -uo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

QUIET=0
ALLOWLIST=""
CATEGORIES=""

while [ $# -gt 0 ]; do
  case "$1" in
    --quiet|-q)        QUIET=1 ;;
    --allowlist)       shift; ALLOWLIST="${1:-}" ;;
    --allowlist=*)     ALLOWLIST="${1#--allowlist=}" ;;
    --categories)      shift; CATEGORIES="${1:-}" ;;
    --categories=*)    CATEGORIES="${1#--categories=}" ;;
    -h|--help)
      sed -n '2,30p' "$0" | sed 's/^# \{0,1\}//'
      exit 0
      ;;
    *) echo "✗ unknown arg: $1" >&2; exit 2 ;;
  esac
  shift
done

is_allowlisted() {
  [ -z "$ALLOWLIST" ] && return 1
  [ -f "$ALLOWLIST" ] || return 1
  grep -Fxq "$1" "$ALLOWLIST" 2>/dev/null
}

failures_file=$(mktemp -t audit-presence.XXXXXX)
trap 'rm -f "$failures_file"' EXIT

# ----- Check (a): orphan detail files -----

n_orphan=0
while IFS= read -r d; do
  cat=$(echo "$d" | sed -E 's|^detail/([^/]+)/.*|\1|')
  name=$(basename "$d" .md)
  if [ ! -f "sheets/$cat/$name.md" ]; then
    if is_allowlisted "orphan|$d"; then continue; fi
    printf 'orphan\t%s\tno sheets/%s/%s.md\n' "$d" "$cat" "$name" >> "$failures_file"
    n_orphan=$((n_orphan + 1))
  fi
done < <(find detail -type f -name '*.md' -not -name '_index.md')

# ----- Check (b): unknown categories -----

n_unknown_cat=0
if [ -n "$CATEGORIES" ] && [ -f "$CATEGORIES" ]; then
  known=$(grep -v '^#' "$CATEGORIES" | grep -v '^$' | sort -u)
  for d in sheets/* detail/*; do
    [ -d "$d" ] || continue
    cat=$(basename "$d")
    if ! grep -Fxq "$cat" <<<"$known"; then
      if is_allowlisted "category|$d"; then continue; fi
      printf 'category\t%s\tnot in %s\n' "$d" "$CATEGORIES" >> "$failures_file"
      n_unknown_cat=$((n_unknown_cat + 1))
    fi
  done
fi

# ----- Report -----

n_total=$((n_orphan + n_unknown_cat))

if [ "$n_total" -eq 0 ]; then
  [ "$QUIET" -eq 0 ] && echo "✓ Presence audit clean — all detail/* match sheets/*; categories known."
  exit 0
fi

echo "✗ Presence audit found $n_total issue(s):" >&2
echo "  orphan details:    $n_orphan" >&2
echo "  unknown categories: $n_unknown_cat" >&2
echo >&2
head -30 "$failures_file" | awk -F'\t' '{ printf "  [%s] %s — %s\n", $1, $2, $3 }' >&2
if [ "$n_total" -gt 30 ]; then
  echo "  ... and $((n_total - 30)) more" >&2
fi
exit 1
