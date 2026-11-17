# Stage 8: Intermediate Zoom Out Review

*   **Target:** `docs/designs/effort-dashboard.md`, the full file, lines
    1–1201. It has not changed since Stage 5.
*   **Reviewer:** `egm-zoom-out`, in a fresh context. It read the target, the
    Stage 5, 6 and 7 reports, the earlier spec and the mockup files.
*   **Focus:** whether the mitigations the earlier stages proposed would keep
    the design simple, and what earlier stages missed.
*   **Date:** 2026-11-17
*   **Verdict:** approve with changes. There is no new Blocker, and B1 has a
    small fix.

## Headline

About half of the proposed mitigations are needed and small, and the rest are
heavier than the problem. **Adopting the minimal set makes the design smaller.**
It drops:

*   gRPC;
*   the partial unique index;
*   the `deleted` flag;
*   the derive-after-failure path and its separate context;
*   the separate derived-column write pass;
*   the SQL copy of the range rule, its equivalence test and three date indexes;
*   `INVALID_SCOPE`;
*   the initiative-label fetch.

## Elephant checks (verified by the orchestrator against Linear, the snapshot and Kratos source)

| Check | Result | Effect |
|---|---|---|
| Do the golden November numbers survive the switch to "include archived, exclude trashed"? | **No.** Tagged projects have 1,110 tasks including archived ones; the mockup snapshot has 1,052 (it was fetched without archived tasks). Of the **58 archived tasks missing**, **18 count in November**. The snapshot has no trashed tasks. | The golden values in item 24 must be **re-derived from a fresh snapshot taken the way the real sync will take it**, not only corrected for B2G 16.3 (Stage 6 C1). This is stronger than F6 below. |
| What JSON field names does Kratos use? | **v2.9.2** (latest v2) uses `protojson` with `EmitUnpopulated: true` and no `UseProtoNames`, so **lowerCamelCase with zero values included**. The document's "camelCase" is right for v2, and Stage 5 N1's snake_case concern was wrong. **But Kratos v3.0.0 was released on 2026-06-26**, and its `encoding/json` codec uses Go's standard `encoding/json`. | The design says "Kratos v2" without a pinned version. Either pin `v2.9.x` or decide on v3 on purpose. The JSON naming rule must then be stated for whichever version is picked. **Stage 9 should check this.** |
| F2's gap: can `StartSync` return 409/429? | **Confirmed by reading items 27 and 16.** `StartSync` starts `Run` in a goroutine and returns at once, and the lock and cooldown checks happen inside `Run`. So the 409 and 429 that the API table, CUJ 6 and the smoke check promise can never reach the caller. | This must be fixed alongside B1. |

--------------------------------------------------------------------------------

## Findings

| ID | Severity | Finding | Recommendation |
|---|---|---|---|
| **F1** | Major | **Sync consistency.** One change covers Stage 5 M1, Stage 7 CR-1 and CR-2, Stage 6 C10 and Stage 5 N3: | **Adopt Z-1: fetch everything, then write.** Reject the snapshot-generation alternative (I-1 / `AS OF SYSTEM TIME`): a `run_id` filter on every read, cleanup of old generations, and GC tuning is too much for a nightly internal cache. **This replaces reconciliation decision 3, so the author must re-decide it.** |
| | | 1. Fetch everything into memory (~32k narrow rows, tens of MB). | |
| | | 2. Run `Derive` on exactly the rows seen in this run. | |
| | | 3. Upsert each row with its derived columns. | |
| | | 4. Hard-delete rows that weren't seen. | |
| | | A failed fetch writes nothing. The remaining risk is a few seconds of half-written pages during the local write, which is accepted in Risks. | |
| **F2** | Blocker until fixed; the fix is small | **Sync lock** (Stage 7 B1, Stage 5 M2, Stage 5 N8, Stage 6 C8). Use an in-process `TryLock`. At startup, set any `running` row to `failed: abandoned`. Drop the partial unique index. **`StartSync` takes the lock and checks the cooldown before it returns**, then starts `Run` on a detached context with the 30-minute timeout. The cooldown applies to manual syncs only. | **Adopt Z-2 plus the synchronous checks.** Reject the heartbeat lease (I-2) and the budget-aware start. Both solve problems this single-process design doesn't have. |
| **F3** | Major | **The range rule is written in two languages** (Stage 5 M4, Stage 6 C3 and C12, Stage 7 D-N4). Keep `report.InRange` as the only copy, and have SQL filter only by scope. Go handles even ~32k rows ("All time" + "All projects") in milliseconds. | **Adopt Z-3.** Drop item 33 and the three date indexes. Reject the loose SQL filter (I-4) unless it's measured as needed. |
| **F4** | Major | **Segment-view drill-down** (Stage 5 M3, Stage 6 C11 and C17, Stage 7 CR-5). | Minimal fix: |
| | | | 1. `CellTasks` returns each task with the points it contributed to that cell (same weight function), so the totals match by construction. Unit-test it. |
| | | | 2. Add **one** field, `share_pts`, to `TaskRow`. `split_of` can be derived. |
| | | | 3. **Hide the Segment column on segment views.** |
| **F5** | Major | **Remove gRPC** (Stage 5 M5, Stage 6 C19). There is no consumer; Alternative F was rejected for the same reason. | **Adopt.** This removes the listener, port 9000, a code generator, a config field and the `golang-grpc` skill requirement. |
| **F6** | Major | **Deleted, archived and disabled entities** (Stage 7 CR-3). | **Drop trashed tasks at fetch time** instead of adding a column. Fetch users with `includeDisabled`. State once in the Sync section that **every** entity is fetched with `includeArchived` (item 25 already says so, but the design part doesn't). **Rebuild the golden values from a fresh snapshot** (see the Elephant checks). |
| **F7** | Major | **Risks** (Stage 7 CR-9, CR-11 and CR-6, Stage 5 N10). Don't add a fourth list. **Turn "Weak points" into an H2 "Risks & Mitigations"** and have the NFRs and Open Questions point to it. | For disclosure, add **one static "How to read these numbers" note** in the UI. It covers: points aren't a common unit; past months can change; a long task counts in every month it overlaps (**CR-6: disclose, don't change the rule**); and shares are of tagged work only. The untagged-points line (N10) is the author's call. |
| **F8** | Major (new) | **The design and the plan restate the same rules**, which caused the range-rule, `segments_from_project`, "deferred", archived and weight-scope findings. | Plan items **name the function and point to the rules-table row** instead of paraphrasing it. Add a "deferred" row. Leave builder-level details out of the design: helper names, the exact backoff steps, batch sizes and retry counts. |
| **F9** | Major | **Initiative scope is undefined** (Stage 6 C5, Stage 7 CR-7). | Decide **"tagged projects only"**, which is what the mockup and the golden test already assume. Also state: "an initiative appears under each segment its in-range tagged projects carry". *The author confirms.* |
| **F10** | Nit (code); the decision is still open | **Security** (Stage 7 CR-10). | A dedicated **service account** scoped to the relevant teams handles private teams without any filter code; resolve the existing Open Question. The manual cooldown already limits quota abuse to about 1,440 requests per hour. A same-origin check on POST and error cleanup are optional. |
| **F11** | Nit | **Alternatives** (Stage 7 CR-13). | Rewrite the rejection reasons as trade-offs. Add **only** the sync-model alternative. Drop "easier to cache" from E. |
| **F12** | Nit | **The conditional overlap note** (Stage 5 N9). | **Reject.** Keep decision 5's static note. |
| **F13** | Nit | **Missing estimates scope** (Stage 5 N2). | **Adopt the single `scope` encoding.** This removes `INVALID_SCOPE` and the `seg` URL gap. |
| **F14** | Nit | **JSON naming and TypeScript types** (Stage 5 N1). | Don't add TypeScript code generation; hand-written types for about 9 responses are fine. Verify the naming. *Elephant check: confirmed camelCase for v2.9.2, and v3 differs, so pin the version.* |
| **F15** | Nit | **Speculative knobs** (Stage 5 N5). | Drop the constant `measure` field; proto can add it later without breaking clients. Make `page_size` a constant. Settings are declared in three places (env, `config.yaml`, `config.proto`); keep one. |
| **F16** | Nit (new) | **Drop `dim` from the scope endpoints.** Return all three contributor tables (~30 × 5 each). | This removes `dim` validation on 5 endpoints and a full server re-render each time the column choice is toggled. `dim` stays in the URL and on `/v1/tasks`. |
| **F17** | Nit | **Mechanical items, accept as proposed:** | Fix these as text edits. A single short **glossary** resolves Stage 6 C13 and C14 and the Terms nits together. Stage 5 N6 (folder renames) needs no change; it is the author's decision. |
| | | Stage 6 C1 (B2G), C2 and C4/CR-8. | |
| | | CR-4: show the arithmetic (5 × 360 ≈ 1,800 < 2,500) and drop the complexity worry, since 8 per page was measured. | |
| | | CR-12 and CR-14. | |
| | | Stage 5 N4 (drop the initiative-label fetch) and N7 (segments stored on projects only). | |
| | | Stage 6 C6. | |

## Adopt / reject summary

| Proposal | Call |
|---|---|
| Fetch-then-write with hard delete (Z-1) | **Adopt.** Re-decide decision 3. |
| Snapshot generations or `AS OF SYSTEM TIME` (I-1) | Reject |
| In-process lock, startup cleanup, synchronous checks in `StartSync`, detached run context, cooldown for manual syncs only | **Adopt** |
| Heartbeat lease, budget-aware start | Reject |
| Single Go range rule (Z-3) | **Adopt** |
| Loose SQL filter with proof test (I-4) | Reject unless measured |
| `share_pts`, `CellTasks` with contributed points, hide the Segment column on segment views | **Adopt** |
| Remove gRPC | **Adopt** |
| Exclude trashed tasks at fetch, `includeDisabled` users, rebuild golden values from a fresh snapshot | **Adopt** |
| Merge "Weak points" into a Risks section, one static disclosure note | **Adopt** |
| Plan items point to the rules table instead of restating it | **Adopt** |
| Initiative scope = tagged projects only | **Adopt** (author confirms) |
| Service account instead of private-team filter code | **Adopt** (resolves an Open Question) |
| Four new alternatives | Add only the sync-model one |
| Conditional overlap note (N9) | Reject |
| Single `scope` parameter for Missing estimates | **Adopt** |
| TypeScript code generation | Reject. Pin the Kratos version and state the JSON naming. |
| `measure` field, three-layer config | Trim |
| Return all three contributor tables (drop `dim` from scope endpoints) | **Adopt** |

## Author decisions this triggers

1.  **Sync model (F1):** replaces reconciliation decision 3 ("recompute
    always") with fetch-then-write.
2.  **Initiative scope (F9):** tagged projects only?
3.  **Linear key (F10):** a dedicated service account?
4.  **Coverage line (N10):** reopen decision 1 to show untagged points on the
    overview, or record low coverage as an accepted risk?
5.  **Kratos version:** pin v2.9.x, or adopt v3.0.0?
