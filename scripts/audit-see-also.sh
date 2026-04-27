#!/usr/bin/env bash
# audit-see-also.sh — verify every `## See Also` reference resolves to an existing sheet.
#
# A reference is `category/topic` style (e.g. `networking/bgp`). It resolves if either
# `sheets/<cat>/<topic>.md` OR `detail/<cat>/<topic>.md` exists. References to the
# same-named ramp-up sheet (`ramp-up/X-eli5`) are also resolved against the ramp-up dir.
#
# Exit 0 = no broken refs (or only known-allowlisted ones). Exit 1 = broken refs found.
#
# Usage:
#   scripts/audit-see-also.sh                 # report broken refs, exit non-zero on any
#   scripts/audit-see-also.sh --quiet         # only print broken refs, no headers
#   scripts/audit-see-also.sh --allowlist .ci/see-also-allowlist.txt
#                                             # don't fail on refs listed in the allowlist
#                                             # (one ref per line, format: source.md|target/path)

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

QUIET=0
ALLOWLIST=""
for arg in "$@"; do
  case "$arg" in
    --quiet|-q) QUIET=1 ;;
    --allowlist=*) ALLOWLIST="${arg#--allowlist=}" ;;
    --allowlist) shift; ALLOWLIST="${1:-}" ;;
  esac
done

# Build the set of valid sheet paths once.
# Format: "<category>/<topic>" — strip .md, strip leading sheets/ or detail/.
valid_set="$(
  { find sheets -type f -name '*.md' -not -name '_index.md' 2>/dev/null
    find detail -type f -name '*.md' 2>/dev/null
  } | sed -E 's|^sheets/||; s|^detail/||; s|\.md$||' | sort -u
)"

# Walk every sheet/detail file. Extract content between `## See Also` and the next `## ` heading.
# Within that block, find lines matching the `category/topic` pattern (also tolerate backticks).
broken=0
broken_lines=""

while IFS= read -r f; do
  # Extract candidate refs from the See Also block. Tolerate:
  #   `category/topic`             ← bare
  #   `category/topic.md`           ← with extension
  #   `sheets/category/topic`      ← prefixed
  #   `sheets/category/topic.md`    ← prefixed + extension
  #   `detail/category/topic[.md]`  ← detail variants
  # Only extract refs that are wrapped in backticks. This eliminates inline-prose
  # collisions like `pub/sub`, `read/write`, `proxy/registrar` (which are concept
  # descriptions, not path refs). The project convention is backtick-wrapped refs.
  refs=$(awk '
    /^## See Also/                       { in_see=1; next }
    in_see && /^## /                     { in_see=0 }
    in_see                               { print }
  ' "$f" | grep -oE '`(sheets/|detail/)?[a-z][a-z0-9-]*/[a-z0-9-]+(\.md)?`' 2>/dev/null \
        | tr -d '`' \
        | sed -E 's|^sheets/||; s|^detail/||; s|\.md$||' \
        | sort -u || true)
  [ -z "$refs" ] && continue

  while IFS= read -r ref; do
    [ -z "$ref" ] && continue
    case "$ref" in */*) ;; *) continue ;; esac

    if ! grep -Fxq "$ref" <<<"$valid_set"; then
      # Allowlist check
      if [ -n "$ALLOWLIST" ] && [ -f "$ALLOWLIST" ]; then
        if grep -Fxq "$f|$ref" "$ALLOWLIST" 2>/dev/null; then
          continue
        fi
      fi
      printf '%s\t%s\n' "$f" "$ref"
    fi
  done <<<"$refs"
done < <(find sheets detail -type f -name '*.md' -not -name '_index.md' 2>/dev/null) > /tmp/.see-also-broken.tmp

broken_count=$(wc -l < /tmp/.see-also-broken.tmp | tr -d ' ')

if [ "$broken_count" -eq 0 ]; then
  [ "$QUIET" -eq 0 ] && echo "✓ See Also audit clean — all references resolve."
  rm -f /tmp/.see-also-broken.tmp
  exit 0
fi

if [ "$QUIET" -eq 0 ]; then
  echo "✗ See Also audit found $broken_count broken reference(s):" >&2
  echo >&2
fi

awk -F'\t' '
  { src=$1; tgt=$2; counts[tgt]++; sources[tgt]=sources[tgt]" "src }
  END {
    PROCINFO["sorted_in"] = "@val_num_desc"
    for (t in counts) printf "  %4d × %s   ← %s\n", counts[t], t, sources[t]
  }
' /tmp/.see-also-broken.tmp >&2 || cat /tmp/.see-also-broken.tmp >&2

rm -f /tmp/.see-also-broken.tmp
exit 1
