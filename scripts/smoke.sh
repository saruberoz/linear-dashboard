#!/bin/sh
# End-to-end smoke test against a running stack (docker compose up).
# Checks every endpoint answers, the core counting invariants hold on real data,
# and the sync guard replies before doing work. Does NOT start a Linear sync
# unless the cooldown allows one (then it only checks the reply).
#   BASE=http://localhost:3300 FROM=2026-11-01 TO=2026-11-30 sh scripts/smoke.sh
set -eu

BASE=${BASE:-http://localhost:3300}
FROM=${FROM:-2026-11-01}
TO=${TO:-2026-11-30}
R="from=$FROM&to=$TO"
API="$BASE/api/v1"
fail=0

check() { # name, condition result (true/false)
  if [ "$2" = "true" ]; then echo "ok   $1"; else echo "FAIL $1"; fail=1; fi
}
get() { curl -sf "$API/$1"; }
near() { jq -n --argjson a "$1" --argjson b "$2" '(($a - $b) | fabs) <= 0.1'; }

for p in "menu?$R" "overview?$R" "segments/B2B?$R" "missing-estimates?$R" "missing-estimates?all=true&$R" \
  "warnings/segment-labels?$R" "untagged-projects?$R" "projects-without-initiative?$R" "initiatives-without-projects" "sync"; do
  code=$(curl -s -o /dev/null -w '%{http_code}' "$API/$p")
  check "GET /v1/$p -> $code" "$([ "$code" = 200 ] && echo true || echo false)"
done
code=$(curl -s -o /dev/null -w '%{http_code}' "$BASE/?$R")
check "page / -> $code" "$([ "$code" = 200 ] && echo true || echo false)"

ov=$(get "overview?$R")
seg_sum=$(echo "$ov" | jq '[.segments[].pts] | add')
pct_sum=$(echo "$ov" | jq '[.segments[].pct] | add')
tagged=$(echo "$ov" | jq '.taggedTotal')
summary=$(echo "$ov" | jq '.summary.total')
check "segment points add up to the tagged total ($seg_sum vs $tagged)" "$(near "$seg_sum" "$tagged")"
check "tagged total equals the summary total ($tagged vs $summary)" "$(near "$tagged" "$summary")"
check "segment percentages add up to 100 ($pct_sum)" "$(near "$pct_sum" 100)"

# Drill-down equals the clicked cell, on a segment view where shared projects are split.
cell=$(get "segments/B2B?$R" | jq -c '[.summary.contributors[] | select(.dim == "priority") | .rows[] | . as $r | .cells | to_entries[] | select(.value.pts > 0) | {a: $r.assigneeId, c: .key, pts: .value.pts}][0]')
if [ "$cell" != "null" ]; then
  a=$(echo "$cell" | jq -r .a); c=$(echo "$cell" | jq -r .c); pts=$(echo "$cell" | jq .pts)
  total=$(get "tasks?scope=segment:B2B&dim=priority&column=$c&assignee_id=$a&$R" | jq .total)
  check "segment drill-down adds up to its cell ($total vs $pts)" "$(near "$total" "$pts")"
fi

# Custom dashboards: an insight of effort by segment must match the overview's segment shares,
# and a bar's drill-down must equal the bar.
spec='{"measure":"effort","slice":"segment","segment":"","view":"bar","filters":{}}'
ins=$(curl -sf -X POST -d "{\"spec\":$spec,\"from\":\"$FROM\",\"to\":\"$TO\"}" "$API/insights/run")
ins_total=$(echo "$ins" | jq '[.result.groups[].value] | add // 0')
check "insight effort by segment adds up to the tagged total ($ins_total vs $tagged)" "$(near "$ins_total" "$tagged")"
bar=$(echo "$ins" | jq -c '.result.groups[0] // empty')
if [ -n "$bar" ]; then
  key=$(echo "$bar" | jq .key); val=$(echo "$bar" | jq .value)
  drill=$(curl -sf -X POST -d "{\"spec\":$spec,\"from\":\"$FROM\",\"to\":\"$TO\",\"sliceKey\":$key}" "$API/insights/tasks" | jq .total)
  check "insight bar drill-down adds up to the bar ($drill vs $val)" "$(near "$drill" "$val")"
fi
code=$(curl -s -o /dev/null -w '%{http_code}' -X POST -d '{"spec":{"measure":"velocity","slice":"assignee","view":"bar"}}' "$API/insights/run")
check "unsupported insight measure is rejected with 400 ($code)" "$([ "$code" = 400 ] && echo true || echo false)"
# Time measures: a bar's drill-down median equals the bar.
tspec='{"measure":"lead_time","slice":"priority","segment":"","view":"bar","filters":{"labelMatch":"bug","includeUntagged":true}}'
tdash='{"includeUntagged":true}'
tbar=$(curl -sf -X POST -d "{\"spec\":$tspec,\"dashboardFilters\":$tdash,\"from\":\"$FROM\",\"to\":\"$TO\"}" "$API/insights/run" | jq -c '.result.groups[0] // empty')
if [ -n "$tbar" ]; then
  key=$(echo "$tbar" | jq .key); val=$(echo "$tbar" | jq .value)
  med=$(curl -sf -X POST -d "{\"spec\":$tspec,\"dashboardFilters\":$tdash,\"from\":\"$FROM\",\"to\":\"$TO\",\"sliceKey\":$key}" "$API/insights/tasks" | jq .total)
  check "bug lead-time drill-down median equals the bar ($med vs $val)" "$(near "$med" "$val")"
fi
code=$(curl -s -o /dev/null -w '%{http_code}' "$API/dashboards")
check "GET /v1/dashboards -> $code" "$([ "$code" = 200 ] && echo true || echo false)"

code=$(curl -s -o /dev/null -w '%{http_code}' "$API/overview?from=$TO&to=$FROM")
check "reversed range is rejected with 400 ($code)" "$([ "$code" = 400 ] && echo true || echo false)"
code=$(curl -s -o /dev/null -w '%{http_code}' "$API/segments/NOPE?$R")
check "unknown segment is 404 ($code)" "$([ "$code" = 404 ] && echo true || echo false)"

# Sync guard: the reply comes before any work. 202-style success, 409 running or 429 cooldown are all valid.
code=$(curl -s -o /dev/null -w '%{http_code}' -X POST -d '{}' "$API/sync")
check "POST /v1/sync replies 200, 409 or 429 ($code)" "$(case $code in (200|409|429) echo true;; (*) echo false;; esac)"
if [ "$code" = 200 ]; then
  code=$(curl -s -o /dev/null -w '%{http_code}' -X POST -d '{}' "$API/sync")
  check "second POST /v1/sync is refused with 409 or 429 ($code)" "$(case $code in (409|429) echo true;; (*) echo false;; esac)"
fi

[ $fail = 0 ] && echo "smoke: all checks passed" || echo "smoke: FAILED"
exit $fail
