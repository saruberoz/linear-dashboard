# Stage 6: Comprehension Test

*   **Target:** `docs/designs/effort-dashboard.md`. The design part is lines
    1–635 and the implementation plan is lines 636–1201.
*   **Reviewer:** `egm-comprehension`, a fresh context that read only the
    target document, with no extra context files and no document standard.
*   **Date:** 2026-11-17
*   **Result:** every section could be paraphrased, so there are **no
    Blockers**. 6 Major, 13 Nit.

## What this proposes

The document proposes an internal, read-only dashboard for PARKEE. It shows how
engineering effort is spread across four business segments (B2B, B2G, B2B NG,
B2C) and how much of that work is done.

*   A Go (Kratos) backend copies the Linear workspace into a single CockroachDB
    node every night, and on demand through a Refresh button with a 15-minute
    cooldown.
*   Counting rules are applied once, in pure Go:
    *   a project with N segments gives 1/N of its effort to each segment;
    *   a parent with an estimated sub-task counts as 0;
    *   canceled tasks count as 0;
    *   only initiatives whose names start with `[W]` or `[WS]` are shown.
*   The backend serves one JSON endpoint per level.
*   A Next.js/shadcn single page shows a filterable left menu, summary panels
    and a contributor table in the style of Linear Insights, where each number
    drills down to its tasks. It also has pages for missing estimates,
    segment-label warnings and untagged projects, all reached through the URL.

## How the existing system works, per this document

All work is tracked in Linear.

*   **Effort:** Linear records no time, only estimate points per task, and teams
    use different scales. FE uses Fibonacci, others 1–5, some none.
*   **Segments** are project labels. Only 60 of 984 projects have one, and 32
    of those 60 have two or three.
*   **Linear Insights** counts a multi-segment project in full under each
    segment, and adds parent and sub-task estimates together.
*   **Initiatives** mix real initiatives with sprints, kanban buckets and team
    groupings. 213 projects sit in more than one.
*   **A stray task-level `b2b` label** sits on 350 tasks, mostly in untagged
    projects.
*   **Existing work:** a mockup (`mockup/`, with in-browser counting in
    `lib/model.js`) and an earlier spec under `docs/superpowers/specs/`.

> **Elephant note on the paraphrase:** it matches the intent. The reviewer
> understood the purpose, the counting rules and the architecture from the
> document alone.

--------------------------------------------------------------------------------

## Major findings

### C1. The golden-test totals break the document's own 0.1-pt rule (lines 950–954 vs 82–83)

*   **Problem:** 364.3 + 18.3 + 0 + 683.3 = 1,065.9, but the stated total is
    1,064. Goal 2 requires the segments to add up to the tagged total within
    0.1 pt.
*   **Elephant check, confirmed:** I re-ran the mockup's rules for 1–30 November
    2026. **B2G is 16.33 pts, not 18.3.** The draft has a transcription error.
    The real results are:
    *   B2B 364.33, B2G 16.33, B2B NG 0, B2C 683.33;
    *   the sum of the segments is 1,064.00, which equals the all-tagged total;
    *   160 unestimated out of 559;
    *   115 warnings.

    The rule holds; only the reference number is wrong.
*   **Fix:** change B2G to 16.3 in the golden test (item 24). Say the values are
    rounded to one decimal, and compare to within 0.1.

### C2. "Deferred" is never defined (line 797 `TaskRow.deferred`, line 874)

*   **Problem:** the rules table describes the parent rule in words, but
    `effort.go` and `TaskRow` use the term "deferred" without defining it.
*   **Fix:** define it once in the rules table: "deferred: a parent whose own
    estimate is ignored because an estimated, non-canceled sub-task exists
    below it". Use that wording everywhere.

### C3. The in-range rule is worded three different ways (lines 113–114, 355, 888–890, 1060–1061)

*   **Problem:**
    *   The rules table says "started on or before the end and not completed or
        canceled before the start".
    *   `InRange` says "finished … on or after its start". Read literally, this
        leaves out tasks that are still open.
    *   The SQL and the test case include still-open tasks.

    It's unclear which wording is the spec.
*   **Same issue in Stage 5:** M4 in `zoom_out_report.md`.

### C4. Does a task leave the warning list once its project is tagged? (CUJ 5 lines 220–222, line 388, Goal 7 line 108)

*   **Problem:**
    *   CUJ 5 says the tasks drop off the list.
    *   The endpoint says groups are "marked counted-via-project", so they would
        stay listed.
    *   Goal 7 lists every task that carries a segment label.

    The "115 warnings" figure can't be checked, and its relation to the 350 /
    344 figures is never explained.
*   **Elephant check:** in the mockup, the warning list contains *every*
    in-range task with a segment task label, including those in tagged
    projects. Those are shown as "already counted through the project's
    segment". So CUJ 5's "drop off" is the sentence that's wrong, unless the
    author wants that behavior.

### C5. How initiatives are placed under segments is never stated (lines 196–197, 381, 383, 925–926)

*   **Problem:** initiatives have no segment labels. The document never says
    that an initiative appears under every segment its tagged projects carry.
    It also doesn't say whether initiative views and totals include untagged
    projects, or where an initiative with only untagged projects appears.
*   **Elephant check:** the mockup only ever loaded tagged projects, so it
    doesn't answer the question either.
    *   **Menu placement:** the mockup puts an initiative under each segment
        that any of its in-range tagged projects carries. This is
        implementation behavior, not something the document states.
    *   **Initiative totals:** whether they count untagged projects is
        **undecided**. This is a real gap for the author to decide.

### C6. The "Go conventions" requirement is not self-contained (lines 642–658)

*   **Problem:** "load the `cc-skills-golang` skills" means nothing to a reader
    who doesn't know Claude Code skills. The document gives no source and no
    link. The folder-rename table rests on "the skills' layout", which a reader
    can't check.
*   **Fix:** name the source (the `cc-skills-golang` plugin by samber) and how
    it's used. State the layout rules being applied right in the document:
    descriptive package names, no package named after a Go standard-library
    package, and the `cmd/` + `internal/` layout. That way the renames can be
    justified without the skill files.

--------------------------------------------------------------------------------

## Nit findings

| ID | Finding | Scope |
|---|---|---|
| C7 | `seg` is listed as URL state but never explained. Goal 10 leaves out `seg`. `page.tsx` reads `seg` but not `via`. | 421, 1109 |
| C8 | The cooldown is checked in `Runner.Run`, which the nightly scheduler shares, so a Refresh at 00:50 would block the 01:00 run. The order of "insert `running` row" vs "check last start" is unclear. (Same as Stage 5 N8.) | 299–302, 993–1005 |
| C9 | `segments_from_project` is said to be stored "on the task row", but the `issues` table has no such column. (Same as Stage 5 N7.) | 310–314 vs 441, 443 |
| C10 | "Derived fields keep the last values if step 3 fails" can't be true when they're written in batches of 500. "A parent is never counted on top of sub-tasks stored in the same run" is hard to parse. (Related to Stage 5 M1.) | 319–326, 998 |
| C11 | "Weight only for `Segment` scopes" reads too narrowly. The overview share table and the `dim=segment` columns also split. (Related to Stage 5 M3.) | 924, 912 |
| C12 | "About 1,000 tasks for a month" vs 559 in the November golden test. 1,051 is the all-time count. | 368 vs 955, 40 |
| C13 | Undefined names: FE, OS and DVOS teams, B2B NG = "B2B Non Group" (shown only in the test aliases), the meaning of `[W]`/`[WS]`, WIB = Asia/Jakarta, "superpowers". | 13, 62, 119, 357, 469, 561, 681 |
| C14 | Linear terms with no pointer: Insights, complexity points, the `duplicate` state type, team estimate settings, label groups. | throughout |
| C15 | CUJ 6 says "Data as of", but Goal 9 specifies the header text as "Synced …". Are these the same element? | 229, 390 vs 117–123 |
| C16 | "The owner" vs "the author": is this one role or two? | 98, 142, 208, 219, 227 |
| C17 | Unstated edge cases: unassigned tasks in the contributor table and Missing estimates; whether `counted_pts` is weighted on segment views (Stage 5 M3); whether the Untagged page includes tasks with no project; whether the default range is fixed to November 2026 or means "this month"; what "shared", `share_pts` and `split_of` mean. | various |
| C18 | Relies on outside documents: rejected options cite §6/§7 of the earlier spec; "the stale line about the update log" assumes the old spec is open; the golden test makes `mockup/lib/model.js` the de facto rule reference without saying so. | 468–470, 739–741, 949 |
| C19 | gRPC has no stated consumer. (Same as Stage 5 M5.) | 248, 763, 1081 |

## Overlap with Stage 5

C3 = M4, C8 = N8, C9 = N7, C10 ≈ M1, C11 and C17 (weighted `counted_pts`) ≈
M3, and C19 = M5. **New in this stage:** C1 (a wrong golden value, now
verified), C2, C4, C5 (a real decision gap), C6, C7, and C12–C18.
