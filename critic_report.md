# Stage 7: Critic Review (round 1)

*   **Target:** `docs/designs/effort-dashboard.md`, lines 1–635, the design
    part.
*   **Reviewers:**
    *   `egm-design` (D), checked against the feature-spec standard;
    *   `egm-structural` (S), checked against the feature-spec standard;
    *   `egm-style` (Y), with no standard.

    All three ran in a fresh context and all three **completed**.
*   **Context given to the reviewers:**
    *   the earlier spec;
    *   `mockup/lib/model.js`;
    *   the five mockup components.
*   **Date:** 2026-11-17

## Verdicts

| Reviewer | Verdict | Blocker | Major | Nit |
|---|---|---|---|---|
| Design critic | Not approved | 1 | 11 | 8 |
| Structural | Revise | 0 | 4 | 9 |
| Style | No blocker | 0 | 9 | 25 |

After removing duplicates within this stage and across Stages 5 and 6, that
leaves **1 Blocker, 14 Major findings and a set of grouped Nits**. Findings
that earlier stages already raised are marked "↔ Stage 5/6".

### Elephant checks run against live Linear and the snapshot

| Claim | Result |
|---|---|
| D-M3: a task page costs about 6k complexity, so a sync uses about 2M of the 3M/h budget | **Not confirmed.** A 100-task page with `labels(first:20)` and every planned field reported `x-complexity: 8`, so a full sync costs about 2.6k. The point about the request budget being shared with the key owner's other use still stands. |
| D-M4: deleted tasks keep counting | **Confirmed.** Of 32,445 tasks fetched with archived ones included, **279 are trashed (deleted)** and would be counted. |
| D-M4: users who have left disappear | **Confirmed.** `users` returns 103 by default and **167** with `includeDisabled`. |
| D-N5: sub-tasks in a different project from their parent | **Confirmed, rare.** There are 3 in the 60-project snapshot. |

--------------------------------------------------------------------------------

## Blocker

### B1. A crashed sync locks out all future syncs (D-B1) ↔ Stage 5 M2

*   **Scope:** lines 299–302, 340–342 and 444.
*   **Problem:**
    *   The partial unique index on `status='running'` is never cleared when a
        run dies. An OOM kill, a restart, `docker compose down` or a reboot is
        enough.
    *   After that, the nightly run, Refresh and the startup sync all fail for
        good, and the header shows "Syncing…" forever.
    *   It is most likely on the very first startup. An operator who presses
        Ctrl-C during the first 5–10 minute sync ends up with a database that
        can't sync without a manual SQL fix.
*   **Raised by:** Stage 5 rated this Major. The design critic rates it a
    Blocker because there is no way to recover without manual SQL.
*   **Proposed fix:**
    *   At startup, mark any `running` row as `failed: abandoned`.
    *   Either use an in-process lock (one backend process), or a heartbeat
        lease that treats a stale heartbeat as dead.
    *   `StartSync` runs on a detached context. Say so in the Sync section and
        in the new Risks section.

--------------------------------------------------------------------------------

## Major

### Sync correctness

| ID | Finding | Source | Proposed fix |
|---|---|---|---|
| CR-1 | **The sync isn't atomic, and readers see mixed data during every sync, not only failed ones.** For 5–10 minutes, new rows sit next to old derived values. A new estimated sub-task can count while its parent's `effective_pts` is still non-zero, which is double counting and breaks Goal 2. The claim that "derived fields keep the last good values" conflicts with writing in batches. | D-M1 ↔ Stage 5 M1, Stage 6 C10 | Re-decide reconciliation decision 3. Options: fetch everything into memory, then write; or publish a run as a snapshot and switch reads to it at once; or explicitly accept the mixed-data window and show it in the UI. |
| CR-2 | **Derive runs before deletions are marked, so a successful sync can silently understate effort.** A sub-task deleted today still holds its parent at 0 during derive. Then the sub-task is marked deleted and the parent's effort disappears. This is a "Must not happen" item. | D-M2 (new) | On success: mark deletions first, then derive. On failure: derive only. Or have derive ignore rows not seen in this run. |
| CR-3 | **Deleted, archived and disabled entities are handled wrongly.** 279 deleted (trashed) tasks would count. 64 users who have left would lose their names in past months. Archived projects and initiatives would be marked deleted, so their tasks fall out of tagged scope. | D-M4 (new, confirmed) | Store `trashed` and exclude it. Fetch users with `includeDisabled`. Fetch projects and initiatives with `includeArchived`. Spell out the handling per entity, and test it with a deleted task. |
| CR-4 | **The budget statement leaves things out.** The statement checks only the request count. The real complexity cost is negligible (8 per page; claim not confirmed). But the request budget is shared with every other use of the personal key, and 5 retries can't wait out an hourly reset. | D-M3 (partly contested) | Keep the request arithmetic, show the sum (5 × ~360 ≈ 1,800 < 2,500), and note the shared-key caveat. Drop the complexity worry, citing the measured value. |

### Counting semantics

| ID | Finding | Source | Proposed fix |
|---|---|---|---|
| CR-5 | **The Segment column and the segment-view drill-down are undefined and broken.** The mockup splits effort twice, so B2C effort shows up on the B2B view. A drill-down lists unweighted points, so CUJ 2's "adds up to the number clicked" fails for the 32 multi-segment projects. | D-M5 ↔ Stage 5 M3, Stage 6 C11/C17 | Add `share_pts` and `split_of` to task rows, and have the drill-down sum `share_pts`. On a segment view, the Segment dimension shows only that segment, or is hidden. Fix CUJ 2. |
| CR-6 | **A long-running task counts in full in every month it overlaps.** A task open from September to November adds its whole estimate to each month, so months don't add up to the quarter. This isn't listed as a weak point. | D-M6 (new) | Either list it as a weak point and label it in the UI, or decide a different rule: for example, count effort in the month the task is completed and show in-progress work separately. **This is the author's call.** |
| CR-7 | **The initiative-view scope is undefined.** Does an initiative's total include its untagged projects? The "No initiative" group rule is missing. | D-M11 ↔ Stage 6 C5 | **The author decides**, then state it in the counting-rules table. |
| CR-8 | **CUJ 5 says tasks "drop off" the warning list** after tagging. The rule, the endpoint and the mockup keep them listed as "counted via project". | D-M7, Y-M6 ↔ Stage 6 C4 | Reword CUJ 5, or redefine the warning to cover only tasks that aren't counted. |

### Risks and disclosure

| ID | Finding | Source | Proposed fix |
|---|---|---|---|
| CR-9 | **There's no Risks & Mitigations section.** The feature-spec standard requires one. Risks are spread across "Weak points", Open Questions, the sync section and the NFRs, without a mitigation for each. | S-M1, D (overview) | Add an H2 **Risks & Mitigations** with one row per risk: B1, CR-1 to CR-6, CR-10, the shared key, and stale data. Give each a mitigation or an explicit acceptance. |
| CR-10 | **Security and privacy risks aren't stated.** A personal key copies private teams and shows them to anyone on the network. `POST /api/v1/sync` has no auth or cross-site protection, so any web page can drain the key's quota. Raw sync errors reach the browser. | D-M8 (new) | Add to Risks. Choose mitigations: exclude private teams or use a scoped service account, require a same-origin header on POST, and clean error text before it reaches the browser. |
| CR-11 | **"The dashboard has to show them" has no mechanism behind it.** Only the task-label warning has UI. Point scales and changing past months have none, and coverage shows only as a project count. | D-M9, Y-M2, S-M1 ↔ Stage 5 N10 | For each weak point, name the UI text that discloses it, or retitle the section "Known limitations" and record the accepted risk. |
| CR-12 | **"The dashboard solves these" overclaims.** Problem 1 (no time tracking) isn't solved, problem 4 is only made visible, and problem 5 is only partly addressed. | Y-M1 | Calibrate: "fixes 2, 3 and 5; makes 1 and 4 visible". |

### Document integrity

| ID | Finding | Source | Proposed fix |
|---|---|---|---|
| CR-13 | **Rejection reasons don't follow the Alternatives rule.** All 11 reasons start "Rejected by author", and none links a record. C and D are preferences only ("the author asked for…"). E's "easier to cache" conflicts with "no cache needed". D, E, F and K have no source. Alternatives are missing for the sync consistency model, normalising points, an external scheduler, and cross-project sub-tasks. | S-M2, S-M3, D-M10, Y-N25 | Rewrite each reason as the trade-off it reflects; the Cons lists already hold them. Link sources where they exist, or drop the "§6/§7" citation. Add the missing alternatives. |
| CR-14 | **Spec contradictions and inconsistent terms.** "All endpoints are GET" vs `POST /v1/sync`. "Exactly 100%" vs the 0.1 tolerance. "Synced …" vs "Data as of". "Owner" vs "author". Undefined `seg`, and Missing-estimates filters missing from the URL. Inconsistent size and speed claims (~1,000 vs "a few thousand" vs 32k with "All projects"; 200 ms "expected" vs "target"). Test coverage described as if it already exists. | S-M4, Y-M3, Y-M4, Y-M5, Y-M7, Y-M8, Y-M9, D-N1, D-N7 ↔ Stage 6 C7, C12, C15, C16 | One pass to make terms consistent: define each role, each URL parameter (in a table), the UI label, the rounding method, and the largest-scope figure, and use future tense for planned tests. |

--------------------------------------------------------------------------------

## Nits (grouped)

| Group | Items | Source |
|---|---|---|
| Sync contract | Is POST `/v1/sync` asynchronous (202 plus polling)? Kratos's 1 s default HTTP timeout and the Next.js proxy timeout would cancel a sync tied to the request. Does the nightly run bypass the cooldown? (↔ Stage 5 N8.) There's no alert or log when syncs keep failing, and no stale-data warning. | D-N2, D-N3 |
| Counting edge cases | Sub-tasks in a different project from their parent (3 confirmed). The segment rule no longer requires the `WS` label group: confirm that's intended. The drill-down `column` values for Total, Unestimated and Unassigned are undefined. | D-N5, D-N6, D-N8 |
| Sizing | The largest-scope claim leaves out "All time" and "All projects" (~32k). Three ORed date conditions likely mean a full scan. ↔ Stage 5 M4. | D-N4 |
| Structure | Promote "Future Improvements" to H2. Move Open Questions after the architecture and Risks. Use horizontal rules consistently. Make the Cons format in Alternatives consistent. Keep line 212 within the wrap width. Make path forms in the Frontend section consistent. Fix the semicolon in the `issues` row. Mention key rotation once. | S-N1–N4, S-N8, S-N9 |
| Terms | "Task" vs "issue" (add a glossary line). Define FE, OS, DVOS, WIB, `[W]`/`[WS]` and B2B NG. One name for "unestimated", "missing" and "no estimate". Define "shared" and "housekeeping pages". Use one date format. "You" vs "the manager". | S-N5, Y-N7, Y-N23, Y-N24 ↔ Stage 6 C13 |
| Wording | "read-only" → "read-only toward Linear". Hours are mentioned 4 times. "Counts twice" → "can count twice". "Half the work" → "about half the tasks". Drop "such as the author". "comparison of estimates" → "sum". "Data loss is not a risk" overclaims. "some data is from the last successful sync" isn't guaranteed. Drop filler ("only ever", "white (light)", "quickly", "plain", "mature"). | Y-N1–N6, Y-N8–N22, Y-N25 |

--------------------------------------------------------------------------------

## Items that need the author

These need a decision, not a text edit:

1.  **CR-1:** the sync consistency model. It reopens reconciliation decision 3.
2.  **CR-6:** how a long task counts across months. Choose between counting it
    in full in each month, with a warning, or another rule.
3.  **CR-7:** whether initiative totals include untagged projects.
4.  **CR-10:** the security posture for a personal key with no sign-in.
5.  **CR-11 / Stage 5 N10:** whether to show the coverage figure. It reopens
    decision 1.

## Traceability

| Raw | Consolidated |
|---|---|
| D-B1 | B1 |
| D-M1 | CR-1 |
| D-M2 | CR-2 |
| D-M3 | CR-4 |
| D-M4 | CR-3 |
| D-M5 | CR-5 |
| D-M6 | CR-6 |
| D-M7, Y-M6 | CR-8 |
| D-M8 | CR-10 |
| D-M9, Y-M2 | CR-11 |
| D-M10, S-M2, S-M3 | CR-13 |
| D-M11 | CR-7 |
| S-M1 | CR-9 |
| S-M4, Y-M3, Y-M4, Y-M5, Y-M7, Y-M8, Y-M9 | CR-14 |
| Y-M1 | CR-12 |
| D-N1–N8, S-N1–N9, Y-N1–N25 | Nit groups |
