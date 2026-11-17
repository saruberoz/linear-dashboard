# Stage 5: Initial Zoom Out Review

*   **Target:** `docs/designs/effort-dashboard.md`, the full file, lines 1–1201
*   **Reviewers:** `egm-zoom-out`, the simplicity and scope reviewer, and
    `egm-inventive-catalyst`, which looks for false trade-offs. Both ran in a
    fresh context.
*   **Context given to the reviewers:**
    *   the earlier spec, `docs/superpowers/specs/2026-11-17-linear-dashboard-design.md`;
    *   `mockup/lib/model.js`;
    *   the mockup components `dashboard.tsx`, `contributors-pivot.tsx`,
        `app-sidebar.tsx`, `panels.tsx` and `task-list.tsx`.
*   **Date:** 2026-11-17

## Verdict

| Reviewer | Verdict | Blocker | Major | Nit |
|---|---|---|---|---|
| Zoom Out | Approve with changes | 0 | 4 | 7 |
| Inventive Catalyst | False compromises found | 0 | 3 | 4 |

Neither reviewer found a blocker. Both judged the architecture proportionate:
the in-process scheduler, single-node CockroachDB, no cache, one endpoint per
level, fetching archived tasks, and the rate-limit arithmetic. Both found the
design aimed at the right problem.

**The largest risk both reviewers raised independently is the sync failure
path** (Z-1 and I-1, plus Z-2 and I-2). The design's own shutdown sequence can
leave the sync permanently locked.

Finding IDs: **Z-n** is from the Zoom Out reviewer, **I-n** from the Inventive
Catalyst. "Elephant check" notes record what the orchestrator verified against
the files.

--------------------------------------------------------------------------------

## Major findings

### M1. The partial-failure sync is complicated, and the mixed state is visible even during a normal sync (Z-1, I-1)

*   **Scope:** Sync steps 1–5 and "What a failed run leaves behind" (lines
    297–338); Goal 9's failure header (116–123); item 27 (991–1003); item 28
    (1014–1018).
*   **Problem:**
    *   **Extra machinery.** Writing each page as it arrives forces several
        added pieces:
        *   "derive always runs";
        *   a separate 5-minute context;
        *   a mixed-state header;
        *   a dedicated failure test.
    *   **Stale derived fields during every sync.** Raw columns are already new
        while derived ones are still old, for 5–10 minutes. For example, a
        task shows "Estimate 8 / Counted 3".
    *   **The draft contradicts itself.** Line 326 says the derived fields
        "keep the last good values" if derive fails. But derive writes in
        batches of 500, so a failure partway through leaves them partly
        updated.
*   **Proposed resolutions:**
    *   **Z-1: fetch first, write later.** Fetch everything into memory (about
        32k narrow rows, tens of MB). Derive in memory. Only then write in
        batches and delete rows that weren't seen. A failed fetch writes
        nothing. This removes:
        *   the derive-after-failure path;
        *   the separate context;
        *   the mixed-state header;
        *   most of the failure test.
    *   **I-1: publish a snapshot.** Each run writes rows keyed by `(run_id, id)`
        and derives within that generation. On success it updates one row:
        `published_run = run_id`. Reads always filter on the published run.
        Readers never see a partial, failed or half-derived run, and the
        `deleted` flag and its step go away. A lighter variant uses
        CockroachDB's `AS OF SYSTEM TIME` with the timestamp of the last
        successful run.
*   **Trade-offs:**
    *   **Z-1:** the write phase can still fail partway, but that is a local
        database write, which fails far less often than ~360 remote calls.
        Memory use is tens of MB.
    *   **I-1:** storage doubles during a sync, and old generations must be
        cleaned up. Every read query must carry the generation filter, and one
        forgotten filter mixes data. The `AS OF SYSTEM TIME` variant needs the
        garbage-collection window raised (default 4 h) above the longest
        failure streak.
*   **Note:** the author chose "recompute always" in reconciliation decision 3.
    Both resolutions above would replace that decision, so the author has to
    re-decide it.

### M2. A crashed sync can lock all future syncs forever (Z-2, I-2)

*   **Scope:** Step 1 "Start" (299–302); the `sync_runs` partial unique index
    (444, 1036–1038); shutdown (829–830); item 27 (998–1000).
*   **Problem:**
    *   A `status='running'` row is cleared only if the run reaches its finish
        step. An OOM kill, SIGKILL or host reboot leaves the row behind.
    *   The design makes this likely. Docker stop gives 10 s before SIGKILL,
        while the derive step deliberately runs on a separate 5-minute context
        that ignores cancellation.
    *   After that, the nightly sync and Refresh both get `SYNC_RUNNING`
        indefinitely, and the header says "Syncing…" forever.
    *   The startup check only runs when there has never been a successful
        sync, so it doesn't help.
    *   **I-2 adds:** the design doesn't say which context the `StartSync`
        goroutine gets. If it inherits the request context, the sync is
        canceled as soon as the response is sent.
*   **Proposed resolutions:**
    *   **Z-2:** guard against a second run with a lock inside the one backend
        process. At startup, mark leftover `running` rows `failed`. Keep
        `sync_runs` only for history and status.
    *   **I-2:** a heartbeat lease. A run counts as `running` only while its
        `heartbeat_at` is fresh. Abandoned rows are cleaned up at startup.
        `StartSync` runs on a detached context with the 30-minute timeout.
*   **Trade-offs:**
    *   **Z-2:** assumes a single backend instance, which the design already
        assumes.
    *   **I-2:** the heartbeat interval needs tuning. Rate-limit backoff can
        pause a healthy run for about 31 s, so too tight a threshold would
        start a second run.

### M3. On segment views, the drill-down can't add up to the number clicked (I-3)

*   **Scope:** CUJ 2 (186–192); the segment-split and contributor rules (352,
    399–400); `TaskRow` (795–797); `Contributors` / `CellTasks` (911–914);
    the smoke check (733).
*   **Problem:**
    *   On a segment view, a contributor cell is weighted by 1/N for
        multi-segment projects. But `TaskRow` carries only `counted_pts`, which
        is unweighted. CUJ 2 promises "the points in the panel add up to the
        number they clicked", and that fails for tasks in the 32 multi-segment
        projects.
    *   **A second bug:** with `dim=segment` on a segment view, effort is split
        twice. First by 1/N for the view, then again across the project's
        segments. So the B2B view shows B2C columns holding a quarter of a
        shared task.
    *   The smoke test checks an overview cell, where the weight is 1, so it
        wouldn't catch either problem.
*   **Elephant check, confirmed in the mockup:**
    *   `mockup/lib/model.js` `pivot()` computes `e = i.eff * w`, then adds
        `e / keys.length` to each segment column. That is the double split.
    *   `mockup/components/task-list.tsx` shows `i.eff`, which is unweighted.
    *   The golden numbers in the plan come from this mockup. Overview cells
        are unaffected, but segment-view drill-downs are wrong today.
*   **Proposed resolution:**
    *   Give `TaskRow` `share_pts` and `split_of` next to `counted_pts`, the
        same way `ProjectRow` already does.
    *   The drill-down totals `share_pts` and shows "4 · half of 8".
    *   On a segment view, `dim=segment` collapses to that one segment, or is
        hidden.
    *   Add a smoke check on a segment-view cell that contains a multi-segment
        task.
*   **Trade-off:** one more field per task row, and each task list must choose
    which value to show.

### M4. The date-range rule exists twice, in Go and SQL, and the two versions already disagree (Z-3, I-4)

*   **Scope:** Request path (362–372); item 20 (886–893); item 32 (1058–1061);
    item 33 (1068–1074); the range indexes (446–449).
*   **Problem:**
    *   It goes against the design's premise of "each counting rule once, in
        code", and it needs an equivalence test just to keep the two copies in
        step.
    *   The wordings already differ. Item 20 says "finished … on or after its
        start", which leaves out tasks that are still open. The rules table
        (line 355) and the SQL `COALESCE(…, 'infinity')` both include them.
    *   The table says "started on or before the end", while `daterange.go` uses
        a half-open interval. These match only if "end" means the last
        millisecond of the day in Jakarta time, and the design never says so.
*   **Proposed resolutions:**
    *   **Z-3:** SQL filters only on scope and `NOT deleted`. `report.InRange`
        is the single rule. Drop item 33 and the three date indexes. Scopes hold
        at most ~1k tasks, or ~32k for the whole workspace.
    *   **I-4:** SQL applies a deliberately loose filter that can only include
        too many rows, never too few:
        `created_at < end AND COALESCE(completed_at, canceled_at, 'infinity') >= start`.
        Go applies the exact rule. The test then only has to prove that the
        loose filter never drops a task.
*   **Trade-off:** more rows are read for "All time" (up to ~32k), which should
    be measured.

### M5. A gRPC server and port are built with no client (Z-4)

*   **Scope:**
    *   line 248, "HTTP + gRPC handlers";
    *   items 9, 10, 11, 12, 13, 15 and 34;
    *   the `golang-grpc` skill in the required list (647).
*   **Problem:** The only consumer is the Next.js frontend, which uses JSON
    over HTTP. Alternative F was rejected because "no consumer other than this
    page is planned". gRPC adds a listener, middleware, a code-generation
    plugin, a config field and an exposed port for nobody.
*   **Proposed resolution:** run Kratos with the HTTP transport only. Remove
    `protoc-gen-go-grpc`, `grpc.addr`, port 9000, `grpc.go` and the
    `golang-grpc` skill requirement.
*   **Trade-off:** adding gRPC later means regenerating code and adding a
    listener, which is a small job.

--------------------------------------------------------------------------------

## Nit findings

| ID | Finding | Scope | Proposed resolution |
|---|---|---|---|
| N1 (Z-N1) | Frontend `types.ts` is copied by hand from the proto, so it will drift. The doc says Kratos emits **camelCase** JSON, but every name in the doc is snake_case, and the reviewer believes Kratos's codec uses proto names (snake_case). It's also unverified whether zero values are left out of the JSON, for example B2B NG = 0. | item 39 (1121–1122) | Generate TypeScript types or OpenAPI from the proto with `buf`. Verify Kratos's JSON naming and zero-value output, and state both. |
| N2 (Z-N2) | Missing estimates uses three mutually exclusive parameters plus an `INVALID_SCOPE` check, which isn't in the error list. Its frontend URL carries only `seg`, so it can't express the initiative or project filter that Goal 5 and CUJ 4 need. | 386–387, 419–421, 781–783, 801–805, 1109 | Use the single `scope=segment:KEY|initiative:ID|project:ID` encoding everywhere. |
| N3 (Z-N3) | Deleted rows are kept with a `deleted` flag although no history is kept. **Side note from I:** upserts don't say they reset `deleted = false` when a row comes back. | 321–322, 437–443, 1050–1051 | Hard-`DELETE` unseen rows, or remove the problem by adopting M1 (I-1). |
| N4 (Z-N4) | Initiative labels are fetched but never used. | 277–278, 439 | Drop the fetch and the `initiative` label kind. |
| N5 (Z-N5) | Speculative knobs: a constant `measure` field, and configurable cron, time zone, cooldown and page size for one deployment. | 394, 455–460, 810–818 | Keep them only if the author wants them. They are cheap. |
| N6 (Z-N6) | Renaming the Kratos folders (`biz` → `report` and so on) breaks the match with Kratos CLI scaffolding and examples. Only `sync` → `snapshot` has a technical reason. | 643–662 | Author's decision (reconciliation decision 6). Noted for its cost only. |
| N7 (Z-N7) | `segments_from_project` is listed as a derived task field, but the data model stores `segments` on `projects`. | 310 vs 441, 443 | Store segments on the project only. |
| N8 (I-5) | The fixed 15-minute cooldown also blocks the nightly run: a Refresh at 00:50 makes the 01:00 cron fail with `SYNC_COOLDOWN`. A run that fails early still locks everyone out for 15 minutes. The rate-limit headers that are already read aren't used to decide. | 229–231, 299–302, 613–615, 991–1006 | Exempt the scheduler from the cooldown. Optionally, start a run only when the observed remaining request budget covers it, about 400 plus a margin. An incremental Refresh is possible but not needed now. |
| N9 (I-6) | The initiative-overlap note shows everywhere, but it's unknown whether the visible `[W]`/`[WS]` initiatives overlap at all. The 213 overlaps were counted mostly among sprints and buckets. | 358, 1154–1156 | Show the note only when the listed initiatives overlap, and mark shared rows "also in &lt;initiative&gt;". This would reverse the author's reconciliation decision 5. |
| N10 (I-7) | The overview hides untagged effort even though "Weak points the design must not hide" lists low coverage. The number is already computed for `/v1/untagged-projects`. | 58–61, 104–106, 133, 631–632 | Add one line: "Tagged work: X pts; untagged projects: Y pts (not in the shares)". This would reverse the author's reconciliation decision 1. |

--------------------------------------------------------------------------------

## Items that need the author

These findings reopen decisions the author made during reconciliation (step
6.1). They need an explicit re-decision, not a quiet edit:

*   **M1** reopens decision 3 (a failed sync recomputes always).
*   **N9** reopens decision 5 (the overlap note always shown).
*   **N10** reopens decision 1 (the overview shows the untagged count only).
*   **N6** concerns decision 6 (the skills win over kratos-layout). The reviewer
    raised it for information only.

## Traceability

| Raw finding | Consolidated |
|---|---|
| Z-1, I-1 | M1 |
| Z-2, I-2 | M2 |
| I-3 | M3 |
| Z-3, I-4, I side note 2 | M4 |
| Z-4 | M5 |
| Z-N1 … Z-N7 | N1 … N7 |
| I-5 | N8 |
| I-6 | N9 |
| I-7 | N10 |
| I side note 1 | N3 |
