#!/usr/bin/env bash
# audit-secrets.sh — defense-in-depth scan for accidentally-committed secrets.
#
# Greps the repo (excluding .git/, bin/, vendor-style dirs, and the audit
# script itself) for common credential-shaped strings. Exits 1 on hit and
# prints offending file:line so a human can review.
#
# This is INTENTIONALLY noisy in test files only when a real-looking value
# is present — test files use sentinel constants (e.g. "TESTKEY-...-do-not-leak")
# that are explicitly NOT secrets.
#
# Usage:
#   scripts/audit-secrets.sh             # report + exit 1 on hit
#   scripts/audit-secrets.sh --quiet     # only print hits, no headers

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

QUIET=0
for arg in "$@"; do
  case "$arg" in
    --quiet|-q) QUIET=1 ;;
  esac
done

[[ $QUIET -eq 0 ]] && echo "→ Scanning repo for accidentally-committed secrets..."

# Patterns we flag. Each is conservative — tuned to flag actual leaks while
# minimizing false positives. The audit script and ramp-up educational sheets
# can legitimately discuss these patterns; we exclude scripts/ + ramp-up + tests.
PATTERNS=(
  # Concrete env-var assignments with non-empty values for our known secret name
  '^[^#]*STACK_OVERFLOW_API_KEY[[:space:]]*=[[:space:]]*[^[:space:]"'\''<>]'
  # PEM private keys (any flavor)
  'BEGIN[[:space:]]+(RSA[[:space:]]+|EC[[:space:]]+|DSA[[:space:]]+|OPENSSH[[:space:]]+|PRIVATE)[[:space:]]*KEY'
  # AWS access key id literal pattern (AKIA / ASIA followed by 16 uppercase/digits)
  '\b(AKIA|ASIA)[A-Z0-9]{16}\b'
  # GitHub fine-grained / classic tokens (ghp_ / ghs_ / gho_ / ghu_ / ghr_)
  '\bgh[pousr]_[A-Za-z0-9]{36,}\b'
  # Slack tokens (xoxa, xoxb, xoxp, xoxr, xoxs)
  '\bxox[abprs]-[A-Za-z0-9-]{10,}\b'
  # GitLab personal/project access tokens (glpat-XXXXXX, 20 chars)
  '\bglpat-[A-Za-z0-9_-]{20,}\b'
  # OpenAI API keys (sk-..., sk-proj-..., sk-svcacct-...)
  '\bsk-(proj-|svcacct-)?[A-Za-z0-9_-]{20,}\b'
  # Anthropic API keys (sk-ant-... format, 95+ chars typical)
  '\bsk-ant-(api|admin)[0-9]+-[A-Za-z0-9_-]{40,}\b'
  # Discord bot tokens (3 base64 segments separated by dots, 50+ chars total)
  '\b[MN][A-Za-z0-9_-]{23,25}\.[A-Za-z0-9_-]{6}\.[A-Za-z0-9_-]{27,}\b'
  # Discord webhooks
  'discord(?:app)?\.com/api/webhooks/[0-9]+/[A-Za-z0-9_-]+'
  # Stripe live secret keys (sk_live_... and rk_live_..., 24+ chars)
  '\b(sk|rk)_live_[A-Za-z0-9]{24,}\b'
  # Stripe restricted keys
  '\brk_(live|test)_[A-Za-z0-9]{24,}\b'
  # Google Cloud service-account private-key markers (lowercased)
  '"private_key"[[:space:]]*:[[:space:]]*"-----BEGIN'
  # JWT tokens (3 base64url segments, .-separated, 100+ chars total —
  # conservative to skip the literal "header.payload.signature" docs)
  '\beyJ[A-Za-z0-9_-]{20,}\.eyJ[A-Za-z0-9_-]{20,}\.[A-Za-z0-9_-]{20,}\b'
  # Generic bearer-token-shaped assignments (loose; only matches obvious leaks)
  '(api[_-]?key|access[_-]?token|secret[_-]?key)[[:space:]]*[:=][[:space:]]*["'\''][A-Za-z0-9_+/=-]{32,}["'\'']'
  # Twilio account SID + auth token pairs
  '\bAC[0-9a-f]{32}\b'
  # SendGrid API keys
  'SG\.[A-Za-z0-9_-]{22}\.[A-Za-z0-9_-]{43,}'
  # npm tokens (granular access tokens — lengths vary 36+)
  '\bnpm_[A-Za-z0-9]{36,}'
  # PyPI tokens
  '\bpypi-[A-Za-z0-9_-]{50,}\b'
)

# Files/dirs to exclude.
#
# Threat model: a real leak ends up in source code, scripts, or build configs
# — files that are SOURCE OF TRUTH for the binary. The cheatsheet content
# (sheets/, detail/) is educational documentation that frequently contains
# documented example/sentinel credentials (AWS's `AKIAIOSFODNN7EXAMPLE`, PEM
# header literals, etc.). Those are not secrets and we explicitly skip them.
EXCLUDES=(
  --exclude-dir=.git
  --exclude-dir=bin
  --exclude-dir=node_modules
  --exclude-dir=.idea
  --exclude-dir=.vscode
  --exclude-dir=Cscore.xcframework
  --exclude-dir=sheets            # cheatsheet content — educational, not config
  --exclude-dir=detail            # deep-dive content — educational, not config
  --exclude='*.lock'
  --exclude='*_test.go'           # tests use sentinel keys; their content is safe
  --exclude='audit-secrets.sh'    # this script
  --exclude='*.md'                # any markdown is documentation, not config
)

hits=0
for pat in "${PATTERNS[@]}"; do
  if out=$(grep -RInE "${EXCLUDES[@]}" "$pat" . 2>/dev/null); then
    if [[ -n "$out" ]]; then
      hits=$((hits + $(echo "$out" | wc -l | tr -d ' ')))
      echo "$out"
    fi
  fi
done

if [[ $hits -gt 0 ]]; then
  echo ""
  echo "✗ audit-secrets: $hits potential leak(s) found above."
  echo "  Review each line. If it's a false positive, refine scripts/audit-secrets.sh"
  echo "  patterns. If it's real, rotate the credential and remove from history."
  exit 1
fi

[[ $QUIET -eq 0 ]] && echo "✓ audit-secrets clean — no leaked credential patterns detected."
exit 0
