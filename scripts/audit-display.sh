#!/usr/bin/env bash
# audit-display.sh — verify every sheet/detail renders correctly via `cs`.
#
# Catches the North Star failure mode where pedagogical content (e.g. detail
# pages with LaTeX math) leaks raw `$\bmod$`, `$$...$$`, `\cdot`, etc. into
# terminal output because glamour has no math extension.
#
# Per file, after stripping ANSI, asserts:
#   (a) `cs` exits 0
#   (b) output is non-empty
#   (c) no runtime-crash markers (`runtime error`, `panic:`, `goroutine N [`)
#   (d) no raw `$$` math-block delimiters
#   (e) no LaTeX canary tokens (\bmod, \cdot, \equiv, \sum, \sqrt, \frac, ...)
#
# Exit 0 = all clean (or only allowlisted failures). Exit 1 = real failures.
# Exit 2 = setup error (no binary).
#
# Usage:
#   scripts/audit-display.sh
#   scripts/audit-display.sh --quiet           # suppress headers, only print failures
#   scripts/audit-display.sh --sample 20       # smoke: first N sheets + N details
#   scripts/audit-display.sh --bin /usr/local/bin/cs   # alternate binary
#   scripts/audit-display.sh --allowlist=.ci/display-allowlist.txt
#                                              # tolerate listed failures
#                                              # (one per line, format: kind|name)
#   scripts/audit-display.sh --canary-only     # skip (d); only check canaries — fewer FPs

set -uo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

QUIET=0
SAMPLE=0
BIN="./cs"
ALLOWLIST=""
CANARY_ONLY=0

while [ $# -gt 0 ]; do
  case "$1" in
    --quiet|-q)        QUIET=1 ;;
    --sample)          shift; SAMPLE="${1:-0}" ;;
    --sample=*)        SAMPLE="${1#--sample=}" ;;
    --bin)             shift; BIN="${1:-./cs}" ;;
    --bin=*)           BIN="${1#--bin=}" ;;
    --allowlist)       shift; ALLOWLIST="${1:-}" ;;
    --allowlist=*)     ALLOWLIST="${1#--allowlist=}" ;;
    --canary-only)     CANARY_ONLY=1 ;;
    -h|--help)
      sed -n '2,30p' "$0" | sed 's/^# \{0,1\}//'
      exit 0
      ;;
    *) echo "✗ unknown arg: $1" >&2; exit 2 ;;
  esac
  shift
done

# Resolve binary: relative path needs to be executable; otherwise check PATH.
if [ -x "$BIN" ]; then
  :
elif command -v "$BIN" >/dev/null 2>&1; then
  :
else
  echo "✗ audit-display: '$BIN' not found or not executable. Run 'make build' first." >&2
  exit 2
fi

# Canary regex (Tier 1 — strong). These never appear in legitimate prose,
# only in unrendered LaTeX. If one shows up after `cs` render, math leaked.
canary='\\bmod|\\cdot|\\equiv|\\sum|\\prod|\\sqrt|\\frac|\\mathbb|\\pmod|\\lfloor|\\rfloor|\\lceil|\\rceil|\\gcd|\\phi|\\varphi|\\sigma|\\oplus|\\Rightarrow|\\rightarrow|\\leftarrow|\\quad|\\cdots|\\ldots|\\leq|\\geq|\\neq|\\approx|\\text\{|\\mathit|\\mathrm|\\bigl|\\bigr'

# Panic regex. Strict — must look like a Go runtime crash, not example
# log lines quoting SIGSEGV/SIGABRT from third-party software. We require
# panic: at line start, a goroutine dump header, or "runtime error:" prefix.
panic='^panic:|^goroutine [0-9]+ \[(running|runnable|sleeping|waiting)\]:|runtime error: (invalid memory|index out of range|nil pointer|divide by zero)'

# Strip ANSI escapes so grep can inspect content, not styling.
strip_ansi() {
  sed $'s/\x1b\\[[0-9;]*[A-Za-z]//g'
}

# Allowlist check: returns 0 if (kind, name) is tolerated.
is_allowlisted() {
  local kind="$1" name="$2"
  [ -z "$ALLOWLIST" ] && return 1
  [ -f "$ALLOWLIST" ] || return 1
  grep -Fxq "$kind|$name" "$ALLOWLIST" 2>/dev/null
}

# Check one topic. Writes failure line to stdout on failure; returns rc.
check_one() {
  local kind="$1" name="$2"
  local out rc clean line

  if [ "$kind" = "detail" ]; then
    out=$("$BIN" -d "$name" 2>&1); rc=$?
  else
    out=$("$BIN" "$name" 2>&1); rc=$?
  fi

  if [ "$rc" -ne 0 ]; then
    printf '%s\t%s\texit=%d\n' "$kind" "$name" "$rc"
    return 1
  fi
  if [ -z "$out" ]; then
    printf '%s\t%s\tempty-output\n' "$kind" "$name"
    return 1
  fi

  clean=$(printf '%s' "$out" | strip_ansi)

  if line=$(printf '%s' "$clean" | grep -nE "$panic" | head -1); then
    printf '%s\t%s\tpanic: %s\n' "$kind" "$name" "${line:0:120}"
    return 1
  fi
  if [ "$CANARY_ONLY" -eq 0 ]; then
    if line=$(printf '%s' "$clean" | grep -nE '\$\$' | head -1); then
      printf '%s\t%s\traw-$$: %s\n' "$kind" "$name" "${line:0:120}"
      return 1
    fi
  fi
  if line=$(printf '%s' "$clean" | grep -nE "$canary" | head -1); then
    printf '%s\t%s\tlatex: %s\n' "$kind" "$name" "${line:0:120}"
    return 1
  fi
  return 0
}

# Enumerate basenames (dedup across categories — `cs <name>` resolves a basename).
sheet_names=$(find sheets -type f -name '*.md' -not -name '_index.md' \
  | sed -E 's|^sheets/[^/]+/||; s|\.md$||' | sort -u)
detail_names=$(find detail -type f -name '*.md' -not -name '_index.md' \
  | sed -E 's|^detail/[^/]+/||; s|\.md$||' | sort -u)

if [ "$SAMPLE" -gt 0 ]; then
  sheet_names=$(printf '%s\n' "$sheet_names" | head -n "$SAMPLE")
  detail_names=$(printf '%s\n' "$detail_names" | head -n "$SAMPLE")
fi

n_sheets=$(printf '%s\n' "$sheet_names" | grep -c .)
n_details=$(printf '%s\n' "$detail_names" | grep -c .)

[ "$QUIET" -eq 0 ] && echo "Auditing $n_sheets sheets and $n_details details via $BIN ..."

failures_file=$(mktemp -t audit-display.XXXXXX)
trap 'rm -f "$failures_file"' EXIT

run_check() {
  local kind="$1" name="$2" line
  if ! line=$(check_one "$kind" "$name"); then
    if is_allowlisted "$kind" "$name"; then return 0; fi
    printf '%s\n' "$line" >> "$failures_file"
  fi
}

while IFS= read -r name; do
  [ -z "$name" ] && continue
  run_check sheet "$name"
done <<< "$sheet_names"

while IFS= read -r name; do
  [ -z "$name" ] && continue
  run_check detail "$name"
done <<< "$detail_names"

n_fail=$(grep -c . "$failures_file" 2>/dev/null || echo 0)

if [ "$n_fail" -eq 0 ]; then
  [ "$QUIET" -eq 0 ] && echo "✓ Display audit clean — all $((n_sheets + n_details)) topics render without leaked LaTeX or panics."
  exit 0
fi

echo "✗ Display audit found $n_fail failure(s):" >&2
echo >&2
head -30 "$failures_file" | awk -F'\t' '{ printf "  [%s] %s — %s\n", $1, $2, $3 }' >&2
if [ "$n_fail" -gt 30 ]; then
  echo "  ... and $((n_fail - 30)) more (head shown — full list in $failures_file before exit)" >&2
fi

# Summarize by failure category
echo >&2
echo "Failure breakdown:" >&2
awk -F'\t' '{ split($3, a, ":"); print a[1] }' "$failures_file" \
  | sort | uniq -c | sort -rn \
  | awk '{ printf "  %5d × %s\n", $1, $2 }' >&2

exit 1
