#!/usr/bin/env bash
# End-to-end smoke test of the TRDL HTTP contract against a running instance.
#
#   scripts/smoke-test.sh BASE_URL [EXPECTED_VERSION]
#
# Examples:
#   scripts/smoke-test.sh http://localhost:8080                     # k3d
#   scripts/smoke-test.sh https://136-82-211-146.sslip.io ec2e9c4   # GKE
#
# Used by CI against the freshly built container. Exits non-zero on any failure.
set -uo pipefail

base="${1:?usage: $0 BASE_URL [EXPECTED_VERSION]}"
base="${base%/}"
expected_version="${2:-}"
failures=0

check() { # description, expected, actual
  if [ "$2" = "$3" ]; then
    printf 'ok    %s\n' "$1"
  else
    printf 'FAIL  %s: expected "%s", got "%s"\n' "$1" "$2" "$3"
    failures=$((failures + 1))
  fi
}

status() { curl -s -o /dev/null -w '%{http_code}' --max-time 10 "$@"; }
body()   { curl -s --max-time 10 "$@"; }
header() { # header name, curl args...
  local name="$1"; shift
  curl -s -o /dev/null -D - --max-time 10 "$@" \
    | tr -d '\r' | awk -v n="$(echo "$name" | tr 'A-Z' 'a-z')" \
      'BEGIN{FS=": "} tolower($1)==n {print $2}'
}

check "GET / status"            200          "$(status "$base/")"
check "GET / body"              42           "$(body "$base/")"
check "GET /healthz"            ok           "$(body "$base/healthz")"
check "GET /readyz"             ready        "$(body "$base/readyz")"
check "HEAD / status"           200          "$(status -I "$base/")"
check "GET unknown path"        404          "$(status "$base/does-not-exist")"
check "POST / status"           405          "$(status -X POST "$base/")"
check "POST / Allow header"     "GET, HEAD"  "$(header Allow -X POST "$base/")"
if [ -n "$expected_version" ]; then
  check "GET /version"          "$expected_version" "$(body "$base/version")"
fi

if [ "$failures" -gt 0 ]; then
  echo "$failures check(s) failed against $base"
  exit 1
fi
echo "All checks passed against $base"
