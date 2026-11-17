# Stage 9: Implementation Readiness Test (round 1)

*   **Target:** `docs/designs/effort-dashboard.md`, lines 636–1201, the
    Detailed Implementation Plan.
*   **Reviewers** (all ran in fresh contexts and all **completed**):
    *   `egm-implementation-readiness` (I), against the
        implementation-readiness standard;
    *   `egm-codebase-details` (C), against the same standard;
    *   `egm-style` (Y), with no standard.
*   **Context given to the reviewers:**
    *   the earlier spec;
    *   `mockup/lib/model.js`;
    *   the five mockup components.

    The codebase reviewer also inspected the rest of `mockup/`, excluding
    `data/` and `node_modules/`.
*   **Date:** 2026-11-17

## Verdicts

| Reviewer | Verdict | Blocker | Major | Nit |
|---|---|---|---|---|
| Implementation readiness | Not ready | 6 | 17 | 8 |
| Codebase details | Fail | 2 | 19 | 9 |
| Style | Not ready | 1 | 4 | 13 |

After removing duplicates, that leaves **6 Blockers and 18 Major findings**.
The plan names the right files, but it **leaves these for the builder to
decide**, which the standard's zero-decision rule forbids:

*   the API messages;
*   the database schema;
*   the sync start and recovery path;
*   several rules carried over from the mockup.

It also has no phases.

## Elephant checks (verified by the orchestrator)

| Claim | Result |
|---|---|
| The mockup doesn't use the `cn` package (plan line 1090) | **False; the reviewers are right.** `mockup/lib/utils.ts` contains `export { cn } from "cn"`, and **21 of 22** `components/ui/*.tsx` import from `"cn"`. Removing it breaks the build. |
| The golden totals contradict each other | **Confirmed, and the cause is known.** B2G is **16.3**, not 18.3, because of a transcription error. Separately, the backend will include **58 archived tasks** in tagged projects that the mockup snapshot lacks, and **18 of them count in November**. Rebuild the golden values from a fresh snapshot (Stage 8). |
| shadcn `toast` was replaced by `sonner`, so `add toast` may fail (I-M14, C-N4) | **Contested.** For this project's `base-nova` style, `npx shadcn view @shadcn/toast` resolves (dependencies `cn`, `@base-ui/react`), and `spinner` resolves too. The plan's `add spinner toast` is valid. Only the root `Toaster` mount is missing. |
| Kratos JSON is camelCase (C, "verified accurate") | **Confirmed for v2.9.2** (`protojson`, `EmitUnpopulated: true`). Kratos **v3.0.0** (2026-06-26) changed the codec, so pin the version (Stage 8). |
| CockroachDB health check needs `--insecure`; `start-single-node` doesn't create `dashboard` | Accepted. This is how the `cockroach` CLI behaves. |
| Next.js `rewrites()` are fixed at build time | Accepted. They are written into `routes-manifest.json` during `next build`. |
| buf STANDARD lint rejects `service Dashboard`, `service Sync` and an unversioned config package | Accepted. These fail the `SERVICE_SUFFIX` and `PACKAGE_VERSION_SUFFIX` rules. |

--------------------------------------------------------------------------------

## Blockers

| ID | Finding | Source | Fix |
|---|---|---|---|
| **RB1** | **`cn` removal breaks the build.** Plan line 1090 removes `cn` while item 42 copies 21 components that import it. | Y-B1, C-B1 (confirmed) | Keep `cn`, and add `frontend/lib/utils.ts` to the file list. |
| **RB2** | **The golden totals are wrong.** The segments sum to 1,065.9 while the stated total is 1,064, so no implementation can pass both the golden test and Goal 2. | I-B4, C-B2, Y-M1 ↔ Stage 6 C1, Stage 8 | Rebuild the fixture from a fresh snapshot taken the way the real sync takes it: archived included, trashed excluded. Name the field each golden value checks. |
| **RB3** | **The sync can't be started or recovered correctly.** | I-B1, I-B3, C-M11 ↔ Stage 7 B1, Stage 8 F2 | Split `Start` (synchronous: lock and cooldown) from `Execute` (a goroutine on a background context). At startup, mark leftover `running` rows `failed`. Cooldown applies to manual syncs only. Decide whether failed runs count toward the cooldown, and set `stop_grace_period`. |
| | `StartSync` runs `Run` in a goroutine, so the 409 and 429 responses and the run ID never reach the API. | | |
| | A leftover `running` row (crash, SIGKILL, or the 10 s Docker stop arriving while the 5-minute derive runs) blocks every later sync. | | |
| | Unstated: whether cron obeys the cooldown, whether failed runs count, and which check comes first. | | |
| **RB4** | **The API contract is undefined.** | I-B2, C-M6, C-M7 | Define every request and response message with field types (points as `double`, counts as `int32`, which avoid int64-as-string in JSON). Define `/v1/tasks` parameters and reserved column keys. Add `project_name` to `TaskRow`. Have the drill-down's `counted_pts` carry the cell's weight. ↔ Stage 8 F4 |
| | There are no request or response messages for 11 RPCs: the menu tree, the overview share rows with `shared`, the warnings summary (count, pts, uncounted), the segment-view initiative groups with subtotals, the warning groups, and the untagged list. | | |
| | Shared messages have no field types. | | |
| | `/v1/tasks` has no defined `scope` encoding, "Unassigned" value, column keys per `dim`, or reserved keys for the **Total** and **Unestimated** cells. | | |
| | `TaskRow` has no project name. | | |
| **RB5** | **The database schema is undefined.** There are no types, primary keys (including composite link-table keys), nullability or defaults. `projects.segments` and `initiatives.is_visible` have no step that computes them. | I-B5, C-M9 ↔ Stage 5 N7 | Write out the `0001_init.sql` schema in the plan: types, primary keys, nulls and defaults. Assign the computation of project segments and initiative visibility to a sync step. |
| **RB6** | **`docker compose up` never becomes healthy.** The health check lacks `--insecure`, and the `dashboard` database is never created. | I-B6, C-M12 | Use `cockroach sql --insecure -e "select 1"`, and set `COCKROACH_DATABASE=dashboard` (or run `CREATE DATABASE IF NOT EXISTS` before goose). |

## Major

| ID | Finding | Source | Fix |
|---|---|---|---|
| R1 | **The range rule is worded differently in Go and SQL.** The Go wording excludes open tasks, while the SQL and the test include them. The mockup uses the sync time as "now"; the SQL uses infinity. | I-M1, C-N2 ↔ Stage 5 M4, Stage 6 C3 | Keep one rule, in Go (Stage 8 F3). State how open tasks are handled and which of `completed_at` / `canceled_at` wins when both are set. |
| R2 | **Visibility and tree rules from the mockup were never carried over.** | I-M2, I-M12, C-M8 ↔ Stage 6 C5, Stage 7 CR-7 | Add a rules-table row for each. |
| | A project is shown if it was created in range or has an in-range task. | | |
| | The untagged count: limited to the range, or all projects? | | |
| | Initiative views with untagged projects. | | |
| | Building the task tree: children whose parent is out of range become roots; sort by numeric identifier. | | |
| | Menu sort, with "No initiative" last. Group sorting. | | |
| R3 | **What `Derive` reads and when.** | I-M3, I-M4, C-M10, C-M15 ↔ Stage 7 CR-2, CR-3 | Adopt fetch-then-derive-then-write (Stage 8 F1). Exclude trashed tasks at fetch. Name `issueEstimationType == "notUsed"`. |
| | It runs over rows already marked deleted. | | |
| | It runs before this run's deletions are applied. | | |
| | Upserts never reset `deleted`. | | |
| | Trashed tasks are counted. | | |
| | "Estimates off" is never named (`notUsed`). | | |
| | Its return type is undefined. | | |
| | Batched writes contradict the claim that a failed derive keeps the previous values. | | |
| R4 | **Only the task query is specified.** No field lists for teams (`issueEstimationType`), users (`includeDisabled`; which name to show), labels (parent group), projects (labels, state, URL, initiative links) or initiatives. How `labels(first:20)` truncation is handled isn't stated. | I-M5, C-M15 | List each query's fields, and say which side fills `initiative_projects`. |
| R5 | **The sync-status header isn't built anywhere.** `page.tsx` never fetches `/v1/sync/status`. The three header states aren't assigned to a file. | I-M6, C-M3, Y-N3 | Embed `SyncStatus` in `/v1/menu`, or add a third server fetch. Give the header states to item 41. |
| R6 | **The URL and query parameters don't line up.** | I-M7, C-M4, C-M5 | Use one `scope` encoding (Stage 8 F13). Write a URL-parameter table. Use `await searchParams` (Next.js 16). |
| | `page.tsx` leaves out `via`. | | |
| | There are no URL parameters for the Missing-estimates initiative and project filters or `all`. | | |
| | `?initiative=&project=` won't bind to the proto fields `initiative_id` and `project_id`. | | |
| R7 | **The `rewrites()` target is fixed during `next build`.** A runtime `BACKEND_URL` is ignored. | I-M8, C-M16 | Add `app/api/v1/[...path]/route.ts`, which reads the variable at runtime, or pass it as a build argument. |
| R8 | **Integration tests can't reach CockroachDB**, because no port is published and no test database or URL is defined. | I-M9, C-M13 | Add a `docker-compose.test.yml` override with a published port, a `TEST_DATABASE_URL` and a reset per test. |
| R9 | **The fixture has no source, schema, anonymizer or loader.** Its source is deleted with `mockup/`. | I-M10, C-M14 | Add a committed `scripts/make-fixture` step, define the schema and the hashing, and order it before the mockup is deleted. |
| R10 | **There is no environment-to-config mechanism.** Kratos `config/env` doesn't map `LINEAR_API_KEY` onto `linear.api_key`. The type of `cooldown` is undefined. The Dockerfile doesn't copy `configs/`. | I-M11, C-M18 ↔ Stage 8 F15 | Use `${VAR:default}` placeholders in `config.yaml`, read by the env source. Make `cooldown` a `Duration`. |
| R11 | **`dim=segment` on a segment view splits effort twice.** | I-M13 ↔ Stage 8 F4 | Hide the Segment column on segment views. |
| R12 | **Files the copied code depends on are missing from the file list.** | I-M14, C-M1 | List them all. Remove `ThemeHotkey`. Mount `<Toaster/>`. |
| | Missing: `lib/utils.ts`, `hooks/use-mobile.ts`, `components/theme-provider.tsx`, `favicon.ico`, the prettier configs, `.dockerignore` files, the golangci config, and the `<Toaster/>` mount. | | |
| | `ThemeHotkey` ("d" switches to dark) contradicts the fixed light theme. | | |
| R13 | **Mockup constants have nowhere to go.** | C-M2 | Add `frontend/lib/constants.ts`. Define the `status_bucket` values and map them. |
| | `SEGMENTS`/`SEG` colors, `STATUS`, `PRIORITIES`, `PRIORITY_LABEL`, `PIVOT_DIMENSIONS`, `scopeTitle` and `scopeKey` aren't in any API response. | | |
| | The status keys don't match: the UI uses `done/progress/not`, the API `done/in_progress/not_started`. | | |
| R14 | **There are no phases or checkpoints.** The plan is a flat list of 43 items. | I-M15, C-M19 | Six phases, each buildable and tested: |
| | | | 1. schema and store; |
| | | | 2. counting rules and the golden test; |
| | | | 3. Linear client and sync; |
| | | | 4. API; |
| | | | 5. frontend switch-over; |
| | | | 6. Compose and smoke test, then delete the mockup. |
| R15 | **Test plan gaps.** Missing tests for: handler input checks and 404s, store upserts and link replacement, cycles in `Derive`, the duplicate state, the `b2b_ng` spelling, `breadcrumbFor`, and anything in the frontend. | I-M16, C-M19 | Add these to items 24, 28 and 33, plus a minimal frontend test or a Playwright smoke check. |
| R16 | **Error handling is incomplete.** `INVALID_SCOPE` and an invalid `dim` or segment key are missing from the error list. There is no `errors.proto` or `protoc-gen-go-errors`. | I-M17, C-M5, Y-M4 | Keep one complete error list. Add `api/dashboard/v1/errors.proto` and its plugin. |
| R17 | **buf STANDARD lint fails.** The service names lack the `Service` suffix, and the config package is unversioned. | C-M17 | Rename to `DashboardService` / `SyncService` with `<Rpc>Request`/`<Rpc>Response`, or list exceptions in `buf.yaml`. |
| R18 | **Inaccurate statements.** | Y-M2, Y-M3, Y-N13 | Narrow each claim. |
| | "Renaming breaks nothing": `kratos proto server` writes to `internal/service`. | | |
| | `views.go` says "each" use case runs all 4 steps; most don't. | | |
| | `internal/report` "holds every counting rule", while SQL also holds one. | | |

## Nits (grouped)

| Group | Items | Source |
|---|---|---|
| Rules and text | `NormalizeLabel` strips every character except a–z and 0–9, which matches the mockup; make the design's wording say the same. "Sorted" segments should mean canonical order (B2B, B2G, B2B NG, B2C), not alphabetical. | I-N1, I-N2, C-N1 |
| Frontend | `api.ts`'s "only tasks" leaves out sync and status. Name the Dockerfile's `.next/static` and `public/` copy. Don't list `next-env.d.ts`, which is generated and git-ignored. `breadcrumbFor` should check `via` against the item's own response, not the menu. | I-N3, I-N8, C-N5, C-N9 |
| Tooling | Pin versions (CockroachDB, Go, Node, buf, protoc plugins, golangci-lint). goose needs `*sql.DB` (`stdlib.OpenDBFromPool`). `CGO_ENABLED=0` for distroless. Give a retry interval and a startup-sync timeout. Give real paths for "Go build output". | C-N6, C-N7, C-N8, I-N5 |
| Tests | The smoke test's first POST runs a real full sync, so the 409 test is timing-dependent. The percentages' rounding tolerance needs stating. Globs used for files that aren't generated. | I-N4, I-N6 |
| Wording | The "mechanical choices" caveat vs "author's decision". "The skills win" → "take precedence". Move the header sentence out of the proto list. "Logged without query secrets", but there are none. Drop "thin" and "defensively". Name the "test table". Reorder the success/failure rule. Make the buried test assertion its own bullet. Back the complexity figure with the measured 8 per page. `no-store` applies to all fetches. "With checking only". "Every … always". "Real (rotated) key". The plan's own choices override the skills. | Y-N1–N12, I-N7 |

--------------------------------------------------------------------------------

## Items that need the author

The open decisions from Stages 5–8, still unresolved:

1.  **Sync model:** fetch-then-write, which replaces reconciliation
    decision 3 (Stage 8 F1). RB3 and R3 depend on it.
2.  **Initiative scope:** tagged projects only? (Stage 8 F9, R2)
3.  **Linear key:** a dedicated service account? (Stage 8 F10)
4.  **Coverage line:** show untagged points on the overview (reopens
    decision 1), or accept the risk?
5.  **Kratos version:** pin v2.9.x, or adopt v3.0.0?
6.  **Long tasks across months (CR-6):** disclose only, as Stage 8 recommends,
    or change the rule?

## Traceability

| Raw | Consolidated |
|---|---|
| Y-B1, C-B1 | RB1 |
| I-B4, C-B2, Y-M1 | RB2 |
| I-B1, I-B3, C-M11 | RB3 |
| I-B2, C-M6, C-M7 | RB4 |
| I-B5, C-M9 | RB5 |
| I-B6, C-M12 | RB6 |
| I-M1, C-N2 | R1 |
| I-M2, I-M12, C-M8 | R2 |
| I-M3, I-M4, C-M10, C-M15 (trashed) | R3 |
| I-M5, C-M15 | R4 |
| I-M6, C-M3, Y-N3 | R5 |
| I-M7, C-M4, C-M5 | R6 |
| I-M8, C-M16 | R7 |
| I-M9, C-M13 | R8 |
| I-M10, C-M14 | R9 |
| I-M11, C-M18 | R10 |
| I-M13 | R11 |
| I-M14, C-M1 | R12 |
| C-M2 | R13 |
| I-M15, C-M19 | R14 |
| I-M16 | R15 |
| I-M17, Y-M4 | R16 |
| C-M17 | R17 |
| Y-M2, Y-M3, Y-N13 | R18 |
| remaining I-N, C-N, Y-N | Nit groups |
