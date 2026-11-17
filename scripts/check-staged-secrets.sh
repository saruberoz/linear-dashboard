#!/bin/sh
# Pre-commit guard: refuse commits that stage a Linear API key, an env file,
# or a snapshot of real Linear data.
set -eu

fail=0

# Staged paths that must never be committed.
bad_paths=$(git diff --cached --name-only --diff-filter=ACMR | grep -E '(^|/)\.env($|\.)|(^|/)data/.*\.json$' | grep -v '\.env\.example$' || true)
if [ -n "$bad_paths" ]; then
  echo "pre-commit: refusing to commit env files or Linear data snapshots:" >&2
  echo "$bad_paths" | sed 's/^/  /' >&2
  fail=1
fi

# Linear personal API keys look like lin_api_ followed by ~40 alphanumerics.
if git diff --cached -U0 --diff-filter=ACMR | grep -E '^\+' | grep -qE 'lin_api_[A-Za-z0-9]{20,}'; then
  echo "pre-commit: a staged change contains a Linear API key (lin_api_...)." >&2
  echo "  Move it to .env (git-ignored) and rotate the key if it was ever shared." >&2
  fail=1
fi

exit $fail
