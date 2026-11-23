# PARKEE Effort Dashboard (Linear)

*   **Author:** Wilson Sumanang
*   **Status:** Implemented (see "As built")
*   **Date:** 2026-11-17, updated 2026-11-20 (see "Changes after launch")
*   **Review:** EGM review finished on 2026-11-17 (Stages 5–9). The author
    concluded it **with findings still open**. The implementation plan is not
    ready to build from: 6 Blockers in Stage 9. The review reports and the
    reviewed draft (`design_draft.md`) have been removed from the repository.
    They are in git history at commit `1be0f60`; the six author decisions are
    listed at the end of `implementation_readiness_report.md` there.

--------------------------------------------------------------------------------

## Decisions after review (2026-11-17), which override the text below

Where this section and the rest of the document disagree, this section wins.

### Author decisions

| # | Decision | Effect |
|---|---|---|
| D1 | **Sync: fetch everything, then write** | The sync downloads the whole workspace into memory and runs `Derive` over exactly the rows it fetched. It then upserts each row with its derived columns in one pass, and **hard-deletes** rows not seen in this run. If the fetch fails, nothing is written. There is no `deleted` column and no "derive after failure" path. The only risk left is a few seconds of half-written pages during the local write, and that risk is accepted. |
| D2 | **Initiatives cover tagged projects only** | An initiative's total, its menu entry and its missing-estimates scope use only its segment-tagged projects. An initiative appears under each segment that its in-range tagged projects carry. Tagged projects with no `[W]`/`[WS]` initiative go under "No initiative". |
| D3 | **Production uses a dedicated Linear service account** | Its key goes in `.env`, and it should see only the relevant teams. Development may use the author's key **after rotating it**. |
| D4 | **No untagged points on the overview** | The overview shows the untagged project count only. A static **"How to read these numbers"** note covers four points: shares are of tagged work only; points are summed across different estimate scales; past months can change after re-estimates; and a task open across several months counts in each of them, so months don't add up to a quarter. |
| D5 | **Kratos v3.0.0** | Module `github.com/go-kratos/kratos/v3`. The HTTP generator is `github.com/go-kratos/kratos/cmd/protoc-gen-go-http/v3`, pinned as a Go `tool` dependency. **The HTTP server always encodes responses with Kratos's `protojson` codec**: camelCase, zero values included. v3's default `json` codec would give snake_case and leave out zeros. |
| D6 | **Long tasks count in every month they overlap** | The rule is unchanged, and the note under D4 discloses it. |

### Added after the review (author's request, 2026-11-17)

*   **Projects without initiative:** a housekeeping page listing projects
    active in the range that are in no `[W]`/`[WS]` initiative, tagged or not.
    Each row shows the hidden sprint or bucket initiatives the project belongs
    to. Endpoint: `GET /v1/projects-without-initiative`.
*   **Initiatives without projects:** a housekeeping page listing initiatives
    with no linked project, `[W]`/`[WS]` first. It ignores the date range.
    Endpoint: `GET /v1/initiatives-without-projects`.

*   **Custom dashboards** (author's request, modeled on Linear Insights and
    Dashboards):
    *   An insight has a **measure**: Effort or Issue count. Time measures are
        left out, per the earlier "no elapsed time" decision.
    *   It has a **slice**, an optional **segment** and **filters**, and is
        shown as a **bar chart, table or metric**.
    *   Dashboards hold insights and apply **dashboard-level filters** that
        intersect with each insight's own.
    *   Dashboards are **shared, stored in CockroachDB**, and anyone can edit
        them.
    *   **Multi-valued dimensions:** effort splits evenly across segments and
        labels and counts in full under each initiative. The issue count adds
        1 per group a task is in, and a bar counts distinct tasks.
    *   Every bar, cell and metric drills down to tasks whose shares add up to
        the clicked value. A property test enforces this.
    *   API: `InsightService` in `api/dashboard/v1/insights.proto`.
    *   This reopens rejected Alternative F, a query-shaped API, because
        custom dashboards are the consumer it was missing.

*   **Bug and cycle insights** (author's request; this reverses "no elapsed
    time" for these insights only, and effort is still points):
    *   **Time measures:** cycle time (started → completed), lead time
        (created → completed), issue age (open tasks, created → last sync)
        and triage time (`startedTriageAt` → `triagedAt`, now synced).
    *   **Populations differ from the range rule:** completed in range,
        triaged in range, or still open.
    *   **Results:** each group shows the median, with p25, p75, p95 and max.
        A scatter view plots one point per task, up to 3,000.
    *   **Drill-down:** lists tasks with their durations, and its median
        equals the clicked value.
    *   **Label filters:** exact label names and "label contains". Bugs in
        this workspace use `Bug`, `HOTFIX` and per-team labels such as
        `New Bug - …` and `Repeat Bug - …`.
    *   **Data note:** triage is barely used. Only 55 tasks have ever been
        triaged.

*   **Software-company insights** (author's request):
    *   **New dimensions:**
        *   `work_type`: "bug" when a label contains "bug" or "hotfix", else
            "other". OKR labels that mention bugs also match.
        *   `estimate`: the estimate in points, ordered by size, with "No
            estimate" last.
    *   **Shared dashboards:**
        *   "Delivery & throughput": completed per month, effort delivered,
            lead time trend, cycle time by estimate size, throughput by team.
        *   "Team health": WIP per person, age of in-progress work, bug share
            by team, hotfixes per month, effort by team and segment.
    *   **Both cover all projects,** untagged included.

*   **Feature effort by label** (author's request: features are labels;
    mark tasks with 2 labels):
    *   **Shared counts:** `InsightValue.shared_tasks` and
        `InsightGroup.shared_tasks` count the tasks that also sit in another
        group.
    *   **Split count:** `TaskRow.split_count` is how many groups a
        drill-down task is split across.
    *   **Filtered grouping:** with label filters set, the label dimension
        keeps only the matching labels.

### As built (2026-11-17)

*   **Migrations** use golang-migrate (CockroachDB driver), because the Go
    database skill requires an established migration tool. This replaces
    goose.
*   **Configuration** comes from environment variables only. There is no
    config proto and no `google/wire`; wiring is plain constructors in
    `internal/app`.
*   **Requests are served from memory.** After each successful sync, and at
    startup, the backend loads the stored copy into an immutable in-memory
    dataset, and requests read from that.
*   **Golden test:** `internal/report/testdata/fixture.json` is an anonymized
    export (`cmd/fixture`) of the 62 tagged projects. `TestGoldenNovember2026`
    asserts the Go rules match `mockup/lib/model.js` on that same data:
    B2B 402.33, B2G 16.33, B2B NG 0, B2C 715.33, a total of 1,134, and 161
    unestimated out of 585.

### Changes after launch (2026-11-18)

*   **Fifth segment, Internal PARKEE** (key `INTERNAL`): a project counts when
    it carries the project label `Internal_PARKEE`. The usual matching applies,
    so "Internal PARKEE" and "internal-parkee" also match, but plain
    "Internal" does not. It is listed after B2C everywhere segments appear. The
    label sits in the `WS` label group in Linear, with the other four.
*   **Sync schedule:** `SYNC_CRON` takes several cron specs separated by `;`.
    The default, `0 1 * * *; 0 17 * * 1-5`, syncs every night at 01:00 and
    every weekday at 17:00, in `SYNC_TZ`. The value is quoted in `.env` so the
    Makefile can source it with the shell.
*   **Sync status in the header:** a black pill.
    *   The text is green when the last successful sync is from today
        (Asia/Jakarta), and red when it is older, failed or missing.
    *   While a sync runs, the pill shows "Syncing… 42%" with a bar that fills
        as records arrive. The Linear client reports the record count after
        every page. The runner compares the running count with the size of the
        current copy, so downloading covers 0–95%. Writing to the store shows
        "Saving…" at 95%. The first sync has no copy to compare with, so its
        bar pulses without a number.
    *   `SyncInfo` gains `phase` (`fetching` or `saving`) and `progress`
        (0–100, or -1 when the size is unknown). The page polls every 2 seconds
        while a sync runs.
*   **Untagged projects page:** a row of project-state toggles (Backlog,
    Planned, Started, Paused, Completed, Canceled), each with its count, kept
    in the URL as `pstate`. Canceled is off by default. The menu and overview
    untagged counts leave canceled projects out, to match the page's default.
    The endpoint still returns canceled projects, so the page can show them.
*   **Task lists:** where a list shows projects (drill-down panels and Missing
    estimates), the project is its own column, linked to the project page.
    The drill-down panels widen to `max-w-5xl` to fit it.
*   **Three more housekeeping pages:**
    *   **Tasks without project** (`GET /v1/tasks-without-project`): open
        tasks in range with no project, grouped by assignee like Missing
        estimates. Canceled tasks are left out.
    *   **Projects without tasks** (`GET /v1/projects-without-tasks`):
        projects with no tasks at all, by name. It ignores the range, because
        such a project has no task dates.
    *   **Missing timeline** (`GET /v1/missing-timeline`): projects active in
        range without a start or target date, and open `[W]`/`[WS]`
        initiatives without a target date (Linear has no initiative start
        date). The initiative list ignores the range.
    *   Both project lists reuse the `pstate` state filter, and their menu
        counts leave canceled projects out.
    *   The sync fetches project `startDate` and `targetDate` and initiative
        `targetDate`. Migration `000004_timelines` stores them as nullable
        `STRING` columns (`YYYY-MM-DD`). `ProjectRow` gains `start_date` and
        `target_date`.

### Changes after launch (2026-11-20)

*   **Initiative subtotals on a segment page add up to the segment total.**
    A project listed under N shown initiatives on a segment page gives each
    of them 1/N of its segment share, instead of its full share to each. This
    replaces the "counts fully in each" rule for segment pages (author's
    decision). Initiative pages still show each project at its real total.
    *   `ProjectRow` gains `initiative_split_of` (>1 only on a segment page),
        and `share_pts` is divided by it there.
    *   The row note reads, for example, "1/4 of 12 · 2 initiatives" when a
        project is split across 2 segments and 2 initiatives.
    *   Segment percentages are unchanged: they never used initiatives.
*   **Avg effort per person** on the "Effort share by segment" table, between
    Effort and Status: the segment's assigned effort ÷ its contributors. A
    contributor is a distinct assignee with at least one non-canceled task in
    range in the segment's projects. Unassigned tasks count in neither the
    effort nor the people; the Effort and Share columns still include them. A
    person on a multi-segment project counts in each of its segments, and the
    project's assigned effort is split 1/N as usual. `SegmentShare` gains
    `contributors` and `pts_per_contributor`.
*   **Unassigned with effort** housekeeping page
    (`GET /v1/unassigned-tasks`): open tasks in range with no assignee that
    count more than 0 pts, with or without a project, largest effort first.
    The menu shows their count (`unassigned_task_count`).
*   **Contributor count** next to the Contributors table title: its rows,
    leaving out Unassigned. On a segment page it equals that segment's
    contributor count on the overview.

### Simplifications adopted from the review

*   **Transport:** HTTP only, with no gRPC.
*   **Locking:** one backend process holds an in-process lock. At startup it
    marks any leftover `running` row as `failed`.
*   **Starting a sync:** `POST /v1/sync` checks the lock and the cooldown
    **before it replies**: 409 if a sync is running, 429 during the cooldown.
    The run itself goes on in the background with a 30-minute timeout. The
    cooldown applies only to manual syncs, and failed runs count toward it.
*   **Date rule:** the range rule exists **only in Go** (`report.InRange`). SQL
    filters by scope only, and there are no date indexes.
*   **Tasks fetched:** archived tasks are included and **trashed (deleted)
    tasks are excluded at fetch time**. Users are fetched with
    `includeDisabled`. Initiative labels are not fetched.
*   **Drill-down:** each row carries `sharePts`, its contribution to the
    clicked cell, so the drill-down adds up to that cell. The Segment column
    choice is hidden on segment views.
*   **Missing estimates:** filtered with the same `scope` encoding as
    everything else (`segment:KEY|initiative:ID|project:ID`), plus `all=true`.
*   **Contributor tables:** each scope endpoint returns all three, one per
    column choice, so there is no `dim` parameter.
*   **Removed:** the `measure` field.

### Blocker fixes

*   **`cn` package:** keep it. The shadcn components import it, and
    `frontend/lib/utils.ts` re-exports it.
*   **Golden numbers:** rebuilt from a fresh snapshot taken the way the real
    sync takes it, with archived tasks included and trashed ones excluded. The
    earlier B2G = 18.3 figure was a transcription error; the mockup gives 16.3.
*   **Docker Compose:**
    *   the health check is `cockroach sql --insecure -e "select 1"`;
    *   `COCKROACH_DATABASE=dashboard` creates the database on first start.
*   **Backend address:** the frontend reaches the backend through a route
    handler, `app/api/v1/[...path]/route.ts`, which reads `BACKEND_URL` when it
    runs. Next.js `rewrites()` would fix the address at build time.
*   **Build order**, each phase tested before the next:
    1.  schema;
    2.  counting rules with the golden test;
    3.  Linear client and sync;
    4.  API;
    5.  frontend;
    6.  Compose and smoke test.

--------------------------------------------------------------------------------

## Overview

The Effort Dashboard is an internal, read-only web app. It shows PARKEE's
management and stakeholders where engineering effort goes across the five
business segments: B2B, B2G, B2B NG, B2C and Internal PARKEE. It also shows how
far along that work is.

Every night, and every weekday at 17:00, it copies the workspace from Linear
into its own database. It then
applies a fixed set of counting rules and shows the result on one page:

*   a left-hand menu that you can filter by name, listing segments, then
    initiatives, then projects;
*   summary panels for the selected item;
*   a contributor table in the style of Linear Insights, where you can click any
    number to see the tasks behind it.

"Effort" means Linear **estimate points**, not hours. Linear records no time
spent. Logged hours are a later addition and will come from a separate source.

## The Problem

Management wants to know which segments take the team's effort, who is doing
the work, and how much of it is finished. All the work lives in Linear, but
Linear cannot answer the question directly:

1.  **It does not track time.** The only effort signal it has is the estimate on
    each task.
2.  **Segments are project labels, and many projects have more than one.** 32
    of the 60 tagged projects carry two or three segments. Linear Insights
    counts such a project in full under every segment, so the segment totals add
    up to more than the real total.
3.  **Parents and sub-tasks are both estimated.** 616 of the 1,051 tasks in
    tagged projects are sub-tasks. Adding up every estimate counts the same work
    twice.
4.  **Half the work has no estimate.** 517 of the 1,051 tasks have one. Effort
    is therefore understated, and nothing in Linear lists which people need to
    add estimates.
5.  **"Initiatives" mix real initiatives with containers.** Sprints, kanban
    buckets and team groupings are also stored as initiatives. 213 projects sit
    in more than one initiative, so initiative totals overlap and are noisy.

The dashboard solves these by deciding each counting rule once, in code, and
showing the result in a form that looks like Linear.

### Weak points the design must not hide

These are facts about the data, not bugs. The dashboard has to show them,
because a reader who doesn't know about them will draw the wrong conclusion.

*   **Coverage is low.** Only 60 of 984 projects carry a segment label. A
    reader will take "B2C = 64%" to mean 64% of all engineering. It is 64% of the
    tagged work only. People whose work is all in untagged projects, such as the
    author, do not appear at all.
*   **Points are not a common unit.** FE estimates on a Fibonacci scale, most
    other teams on a 1–5 scale, and several business teams do not estimate at
    all. The dashboard adds the points up as they are. "Done 875 pts" is
    therefore a comparison of estimates, not a measure of output.
*   **Past months can change.** The dashboard keeps only the latest copy of each
    task. If someone re-estimates, deletes or moves an old task, the numbers for
    a past month change when the next sync runs.
*   **A task label `b2b` exists outside the segment system.** 350 tasks carry
    it, and 344 of them are in untagged projects or in no project. Under the
    rules these tasks are not counted. The dashboard warns about them instead.

## Goals

1.  **Segment share.** For any date range, the overview shows each segment's
    share of effort, effort in points, project count, and the split between done
    and in progress. The segment shares add up to exactly 100% of the tagged
    work.
2.  **No double counting.**
    *   A project with N segments gives 1/N of its effort to each segment.
    *   Effort comes from the issue, not its sub-tasks. Below an estimated
        issue, every sub-task counts as 0. A sub-task's estimate counts only
        when no task above it is estimated. *(Changed 2026-11-23; sub-tasks used
        to win over their parent.)*
    *   Canceled tasks count as 0.
    *   Segment totals add up to the overall tagged total, within 0.1 pt of
        rounding.
3.  **Drill-down on one page.** You can choose "All tagged work", a segment, an
    initiative or a project from the left menu, and filter the menu by name.
    Each selection shows the same panels:
    *   totals;
    *   current status;
    *   effort by priority;
    *   effort by label, with "No label" shown separately;
    *   the contributor table;
    *   the items one level down.
4.  **Contributor table in the style of Linear Insights.**
    *   Rows are assignees. Columns are total effort plus one of Priority,
        Status or Segment, and the unestimated count.
    *   Clicking any number lists the tasks behind it, with links to Linear.
5.  **Missing estimates.** A page lists every unestimated task in range,
    grouped by assignee, so the owner can remind each person.
    *   It can be filtered by segment, initiative or project.
    *   The "N of M tasks unestimated" card on every view opens it already
        filtered to that view.
    *   By default it covers tagged projects. An "All projects" toggle adds
        untagged projects and tasks with no project.
6.  **Untagged work.** The overview shows only the number of untagged
    projects, not counting canceled ones (author's decision). Their effort
    appears on the Untagged projects page, not on the overview, and is never
    part of the segment percentages. That page filters by project state and
    hides canceled projects by default.
7.  **Data-quality warnings.** The dashboard lists:
    *   tasks that carry a segment label on the task itself;
    *   projects that have no segment label;
    *   tasks that have no project;
    *   projects that have no tasks;
    *   projects and initiatives that have no timeline (start or target date).

    Each item links to Linear, so someone can fix the tagging there.
8.  **Date filter.**
    *   A task counts in a range if it was created, completed or in progress
        during that range.
    *   The range uses Asia/Jakarta time and defaults to 1–30 November 2026.
9.  **Freshness.** Data is synced from Linear every night at 01:00 and every
    weekday at 17:00. A Refresh button starts a sync on demand. The header on
    every view shows the **last sync time and its result** (author's request)
    in a black pill, green when the data is from today and red otherwise:
    *   after a success, "Synced 17 Nov 2026, 01:07 WIB";
    *   while a sync runs, "Syncing… 42%" with a progress bar;
    *   after a failure, a warning: "Last sync failed 18 Nov, 01:09 WIB · some
        data is from the last successful sync, 17 Nov, 01:07 WIB", with the
        error reason in a tooltip.
10. **Shareable views.** The selection, the date range, the contributor-table
    columns and the breadcrumb path (`via`) live in the URL, so any view can be
    sent as a link.
11. **Runs with one command.** `docker compose up` starts CockroachDB, the Go
    (Kratos) backend and the Next.js frontend. The first sync runs on its own.

### Must not happen

*   A number labeled or read as hours when it is estimate points.
*   Effort that is understated without the understatement being visible.
*   Segment shares that add up to more than 100%, or a project counted in full
    under two segments.
*   The Linear API key reaching the browser.

## Non-Goals

*   **Hours.** No time tracking and no hours field in this version. The data
    model leaves room for hours to be added later as a second measure.
*   **Elapsed or cycle time.** The owner ruled these out.
*   **Status at a past date.** "Current status" means the status in Linear
    today. The dashboard does not rebuild what the status was at the end of a
    past range.
*   **Sign-in and access control.** The dashboard runs on a trusted internal
    network. Anyone who can reach it can see everything and can start a sync.
*   **Writing to Linear.** The dashboard never creates, edits or tags anything
    in Linear. Fixing labels or estimates is done by people, in Linear.
*   **Counting segment labels from tasks or initiatives.** Segments come from
    project labels only. Segment labels on tasks are reported, not counted.
    Initiatives have no segment labels in Linear today.
*   **Automatic reminders.** No emails or Slack messages. The Missing estimates
    page is the reminder tool.
*   **A time series of past snapshots.** Each sync replaces the previous copy.
    Earlier versions are not kept.

## Open Questions

*   **Who owns the Linear API key?** The sync currently uses a personal key
    (Wilson's). It sees only what that person can see, and it stops working if
    that account is disabled. The key was also pasted into a chat session and
    should be rotated. Should production use a dedicated service account?
*   **Where does it run, and who can reach it?** Without sign-in, the network
    is the only access control. The host and how stakeholders reach it are not
    decided.
*   **Do changing past-month numbers matter?** If management compares October
    with November in reports, a re-estimate made in December changes October's
    figure. Is that acceptable, or does a month need to be frozen once it's
    reported?

## Critical User Journeys (CUJs)

### CUJ 1: Monthly segment report

*   **Given**: A scheduled sync has finished. The date range is the default,
    1–30 November 2026.
*   **When**: A manager opens the dashboard.
*   **Then**: The overview shows each segment's share of tagged effort; the
    shares add up to 100%. Each segment also shows its points, project count
    and done-vs-in-progress split. Two warnings sit at the top: the number of
    tasks with a segment label on the task itself, and the number of untagged
    projects.

### CUJ 2: Who worked on a segment, and on what

*   **Given**: The manager is on the overview.
*   **When**: They click B2B in the left menu, set the contributor columns to
    Priority, and click the number where one person's row meets the High column.
*   **Then**: A side panel lists that person's High-priority B2B tasks in range.
    Each task shows its status, estimate, counted points and a link to Linear.
    The points in the panel add up to the number they clicked.

### CUJ 3: Find a project quickly

*   **Given**: The left menu shows the segments, then initiatives, then
    projects.
*   **When**: The manager types "protection" in the menu filter and clicks a
    project.
*   **Then**: The menu shows only items whose name matches. The project view
    shows the project's real total, not its share of a segment. It also shows its
    segment badges (for example "B2B · B2C · multi-segment") and its task list,
    with sub-tasks nested under their parent.

### CUJ 4: Chase missing estimates

*   **Given**: Some tasks in range have no estimate.
*   **When**: The owner opens "Missing estimates" and picks B2C. Or, from a
    project view, they click that project's "N of M unestimated" card.
*   **Then**: The unestimated tasks for that segment or project are listed,
    grouped by assignee, with a link to each task in Linear. Turning on "All
    projects" adds unestimated work in untagged projects. Canceled tasks are left out, and so are teams
    with estimates turned off, sub-tasks of estimated issues, and issues whose
    sub-tasks are estimated.

### CUJ 5: Fix tagging

*   **Given**: Tasks carry the task label `b2b`, but their project has no
    segment label.
*   **When**: The owner opens "Segment label on task".
*   **Then**: The tasks are grouped by project, with a link to tag each project
    in Linear. After the project is tagged and the next sync runs, the project's
    tasks count toward the segment and drop off the warning list.

### CUJ 6: Fresh numbers on demand

*   **Given**: Someone changed estimates in Linear an hour ago.
*   **When**: The owner clicks Refresh.
*   **Then**: A sync starts, and the button shows that it is running. When the
    sync finishes, "Data as of" updates and the view reloads. A second click
    while a sync is running, or within the 15-minute cooldown, is refused with a
    message.

## Technical Plan (Architecture)

### Components and data flow

```text
    01:00 daily, 17:00 weekdays (WIB)  /  POST /v1/sync (15 min cooldown)
Linear GraphQL ───────────────────────────────────────────────┐
 (2,500 req/h, 3M complexity/h)                               ▼
                                         ┌──────────────────────────────────┐
                                         │ backend  (Go, Kratos v2)         │
                                         │  linear/  GraphQL client, paging │
                                         │  snapshot/ copy → upsert →       │
                                         │           derive → mark deleted  │
                                         │  report/  counting rules (pure)  │
                                         │  store/   pgx repositories       │
                                         │  handler/ HTTP + gRPC handlers   │
                                         └───────┬──────────────▲───────────┘
                                                 │ pgx          │ JSON /v1/*
                                                 ▼              │
                                     ┌────────────────────┐     │ (server side,
                                     │ CockroachDB        │     │  service name)
                                     │ single node        │     │
                                     └────────────────────┘ ┌───┴─────────────────┐
                                                            │ frontend (Next.js,  │
                       browser ── /  and  /api/v1/* ──────► │ shadcn/ui)          │
                                                            │ rewrites /api/v1/*  │
                                                            │ → backend           │
                                                            └─────────────────────┘
```

There are three services, all run by Docker Compose:

*   **CockroachDB** is a single node with a persistent volume. It is the only
    store.
*   **The backend**, in Go with Kratos v2, is the only thing that talks to
    Linear. It also runs the scheduled syncs in-process. It holds every counting
    rule and serves JSON.
*   **The frontend**, in Next.js with shadcn/ui, renders the single page. The
    browser only ever talks to the frontend. The frontend passes `/api/v1/*` on
    to the backend, so the backend never needs to be reachable from the browser.
    The Linear API key exists only in the backend container.

### Sync: copying Linear into CockroachDB

**What is fetched:** teams, users, task labels, project labels, initiative
labels, initiatives with their project links, projects with their labels, and
all tasks with their labels. **Archived tasks are included.** Linear archives
finished tasks automatically, and the workspace has 32,401 tasks, of which
about 11,000 are archived. Without them, older months would lose their finished
work.

**Cost:**

*   Tasks are fetched 100 per request, with only the fields needed. That is
    about 330 requests for tasks, plus about 30 for everything else.
*   That fits Linear's limit of 2,500 requests per hour.
*   A test query showed that paging through task IDs alone takes about 2
    minutes. A full sync is expected to take 5–10 minutes.
*   The client reads Linear's rate-limit response headers. When it gets a
    rate-limit error, it waits with increasing delays and retries, up to 5
    times.
*   The client never requests more than 10,000 complexity points in a single
    query.

**Steps for one run:**

1.  **Start.** Insert a `sync_runs` row with status `running`. A database
    constraint allows only one `running` row at a time. If another sync is
    running, `POST /v1/sync` returns 409. If the last run started less than 15
    minutes ago, it returns 429.
2.  **Fetch and upsert.** For each kind of entity, page through Linear and
    upsert each page as it arrives. Every row written gets
    `last_seen_run = <this run id>`. Upserting page by page avoids one very large
    transaction, because CockroachDB limits transaction size.
3.  **Derive (always runs, author's decision).** After the fetch, whether it
    succeeded or failed, compute these fields for every stored task over the
    full task tree, and store them on the task row:
    *   `segments_from_project`;
    *   `has_estimated_descendant`;
    *   `effective_pts`;
    *   `is_missing_estimate`;
    *   `task_segment_labels`, used for the warnings.

    These fields depend on the whole tree, not on the date range, so they are
    computed once per sync instead of on every request. Running this step even
    after a failed fetch keeps the stored totals consistent with the stored
    tasks. A sub-task is never counted on top of a parent that was stored in
    the same run.
4.  **Mark deletions.** Only after every fetch has succeeded, mark rows whose
    `last_seen_run` is older than this run as `deleted = true`.
5.  **Finish.** Set the run to `succeeded` and record the row counts. If the
    fetch failed: still run step 3, skip step 4, and set the run to `failed`
    with the error message. If step 3 itself fails, the run is `failed`, and
    the derived fields keep the values from the last run that completed step 3.

**What a failed run leaves behind:**

*   Rows already upserted in the failed run hold newer data. Rows not reached
    yet hold the previous run's data.
*   The derived fields are recomputed from exactly those stored rows, so every
    total is consistent with what is stored.
*   Nothing is marked deleted, so a task removed in Linear stays visible until
    the next successful run.
*   The header shows the failure: "Last sync failed <time> · some data is from
    the last successful sync, <time>". `GET /v1/sync` returns the
    last success time, the last attempt time and the attempt's error.

**When it runs:** an in-process scheduler runs the sync every day at 01:00 and
every weekday at 17:00, Asia/Jakarta. If the database has no successful run
yet, the sync also runs at startup.

### Counting rules (backend `report` package)

Every rule lives in pure Go functions with no database or HTTP access. Each one
is covered by table-driven tests built from the cases found in the real data.

| Rule | Input | Result |
|---|---|---|
| Segment of a project | project labels | the set of B2B, B2G, B2B NG, B2C and Internal PARKEE labels. A label matches if it equals a segment name after lower-casing and removing spaces, hyphens and underscores. `b2bng` and `b2bnongroup` both mean B2B NG; `internalparkee` means Internal PARKEE. |
| Segment split | project effort, N segments | each segment gets effort ÷ N |
| Effective points | task, its sub-tasks | 0 if canceled, a duplicate, or below an estimated, non-canceled task anywhere above it; otherwise the task's own estimate (0 if none) |
| Missing estimate | task, team | no estimate, and not canceled, and the team has estimates turned on, and no estimated, non-canceled task above or below it |
| In range | task, from, to (Asia/Jakarta, inclusive) | created in range, or completed in range, or started on or before the end and not completed or canceled before the start |
| Status bucket | Linear state type | completed → Done; started → In progress; backlog, unstarted or triage → Not started; canceled or duplicate → Canceled |
| Visible initiative | initiative name | the name starts with `[W]` or `[WS]`, ignoring case and spaces |
| Initiative total | its projects | on an initiative page, the full total of each of its projects. On a segment page, each project's segment share split evenly across the shown initiatives it is listed under, so the subtotals add up to the segment total (author's decision, 2026-11-20) |
| Label share | task labels | a task's effort is split evenly across its labels; tasks with no label are counted separately as "No label" |
| Task-label warning | task labels | the task carries a segment-named task label. It is reported, never counted. |

### Request path

1.  The handler checks the query parameters: `from` and `to` as `YYYY-MM-DD`,
    `dim` as `priority`, `status` or `segment`, and the scope ID.
2.  The store layer loads the in-range, non-deleted tasks for the scope with a
    single SQL query that filters on project IDs and the three date conditions.
    The largest scope, all tagged work, is about 1,000 tasks for a month.
3.  The `report` functions build the response blocks in memory.

No result cache is needed at this size. The expected response time is under
200 ms.

### API: one endpoint per level (author's choice)

All endpoints are `GET`, take `from` and `to`, and return JSON. They are
defined in protobuf with `google.api.http` mappings, as Kratos does.

| Endpoint | Returns |
|---|---|
| `/v1/menu` | the left menu tree: segments, then visible initiatives, then projects, each with an in-range count; plus counts for the housekeeping pages |
| `/v1/overview?dim=` | the segment share table (pts, %, projects, shared); a summary block over all tagged work; warnings summary; the untagged **project count** (no points) |
| `/v1/segments/{segment}?dim=` | a summary block weighted by the segment split; initiatives grouped with their projects, each at the segment's share |
| `/v1/initiatives/{id}?dim=` | a summary block at real totals; its projects |
| `/v1/projects/{id}?dim=` | a summary block at real totals; the task tree in range |
| `/v1/tasks?scope=&assignee=&column=&dim=` | the drill-down list behind one contributor-table number |
| `/v1/missing-estimates?segment=&initiative=&project=&all=` | unestimated tasks in range, grouped by assignee; filtered to one segment, initiative or project; `all=true` adds untagged projects and tasks with no project |
| `/v1/warnings/segment-labels` | tasks with a segment label on the task itself, grouped by project, each group marked counted-via-project or not counted |
| `/v1/untagged-projects` | projects in range with no segment label, with their effort and project state; canceled ones included, for the page's state filter |
| `/v1/tasks-without-project` | open tasks in range with no project, grouped by assignee |
| `/v1/projects-without-tasks` | projects with no tasks at all, whatever the range |
| `/v1/missing-timeline` | projects in range without a start or target date; open `[W]`/`[WS]` initiatives without a target date |
| `/v1/sync` (POST) and `/v1/sync` (GET) | start a sync; latest run status and "data as of", plus `phase` and `progress` while a sync runs |

**The summary block has the same shape on every level:**

*   `measure` is `"pts"` today. This field leaves room for `"hours"` later.
*   `total`, and `by_status` with `done`, `in_progress` and `not_started`.
*   `by_priority[]`.
*   `by_label[]`, which covers labeled effort only, plus `unlabeled` as a
    separate figure.
*   `contributors`, the table: rows of `{assignee, total, cells{column: {pts,
    count}}, missing}`, laid out for the chosen `dim`.
*   `tasks_total` and `tasks_missing`.

### Frontend

*   **Stack:** Next.js App Router with TypeScript, shadcn/ui on Base UI and
    Tailwind v4, Recharts through the shadcn Chart component, and lucide icons.
*   **Theme:** the white (light) Linear look, as built in `mockup/`.
*   **Breadcrumbs follow the path you came from (author's decision).**
    *   Following a link from a segment or initiative view adds
        `via=segment:KEY` or `via=initiative:ID` to the URL. A project opened
        from B2B shows "All tagged work › B2B › Project", and from B2C it shows
        "… › B2C › Project".
    *   A segment or initiative in `via` that doesn't match the item is
        ignored, so a hand-edited URL can't produce a false breadcrumb.
    *   `via` changes only the breadcrumb and which left-menu entry is
        highlighted. Every number stays the item's real total.
    *   With no `via`, a single-segment project shows its segment, and a
        multi-segment project shows "All tagged work › Project".
*   **One route, `/`.** The state lives in the URL as
    `?scope=all|segment:KEY|initiative:ID|project:ID|missing|labelwarn|untagged`,
    plus `&from&to&dim&seg&via`, and `pstate` for the untagged page's state
    filter. The page is a server component. It reads the URL,
    calls the matching backend endpoint and `/v1/menu` from the server, and
    hands the data to client components. Changing the selection or the date
    calls `router.replace`, which loads the page again on the server.
*   **The drill-down side panel** loads `/api/v1/tasks` from the browser
    through the frontend's pass-through rule.
*   **The mockup is the visual reference:** `components/app-sidebar.tsx`,
    `contributors-pivot.tsx`, `panels.tsx`, `task-list.tsx` and
    `linear-icons.tsx`. The real frontend takes over these components and swaps
    the mockup's in-browser `lib/model.js` for API calls. No counting rule stays
    in the frontend.

### Data model (CockroachDB)

| Table | Key columns |
|---|---|
| `teams` | id, key, name, estimation_type, last_seen_run, deleted |
| `users` | id, name, display_name, active, last_seen_run, deleted |
| `labels` | id, kind (`issue`, `project` or `initiative`), name, group_name, last_seen_run, deleted |
| `initiatives` | id, name, status, url, created_at, is_visible, last_seen_run, deleted |
| `projects` | id, name, state, url, created_at, segments (string array, derived), last_seen_run, deleted |
| `project_labels`, `initiative_projects`, `issue_labels` | link tables, replaced in full for each parent row that is upserted |
| `issues` | id, identifier, title, url, team_id, project_id, parent_id, assignee_id, estimate, priority, state_name, state_type, created_at, started_at, completed_at, canceled_at, archived_at, plus derived columns effective_pts, has_estimated_ancestor, is_missing_estimate, task_segment_labels; last_seen_run, deleted |
| `sync_runs` | id, started_at, finished_at, status, stats (JSONB), error; partial unique index on `status = 'running'` |

Indexes:

*   `issues (project_id)`, `issues (created_at)`, `issues (completed_at)` and
    `issues (started_at)`, for the range query;
*   `issues (parent_id)`, for building the task tree;
*   `issues (assignee_id)`.

### Configuration and secrets

*   **The backend reads:**
    *   `LINEAR_API_KEY`;
    *   `DATABASE_URL`;
    *   `SYNC_CRON`, one or more cron specs separated by `;`, default
        `0 1 * * *; 0 17 * * 1-5`;
    *   `SYNC_TZ`, default `Asia/Jakarta`;
    *   `SYNC_COOLDOWN`, default `15m`.
*   **The frontend reads** `BACKEND_URL`, which is used only on the server and
    in the pass-through rule.
*   **The Linear key** lives in a git-ignored `.env`, and a `.env.example` is
    committed. The key is never logged and never sent to the browser.

## Alternatives Considered

The rejected options below come from the earlier spec
(`docs/superpowers/specs/2026-11-17-linear-dashboard-design.md`, §6 and §7)
and from decisions made while reviewing the mockup.

### Alternative A: Use Linear Insights and Dashboards as they are

*   **Pros**: No build and no hosting. The data is always live. Stakeholders
    already know the interface.
*   **Cons**:
    *   A project with several segments counts in full under each one, so the
        segment totals add up to more than the real total.
    *   Parent and sub-task estimates are added together.
    *   Nothing lists the tasks that have no estimate, by person.
    *   Sprint and bucket initiatives can't be hidden.
*   **Reason for Rejection**: Rejected by author: it can't apply the counting
    rules this report depends on. Its interface is kept as the visual model
    instead.

### Alternative B: A web app that reads Linear live, with "Sign in with Linear"

*   **Pros**: The data is never stale. Linear itself controls access, so each
    viewer sees only what Linear lets them see.
*   **Cons**:
    *   Every page view calls the Linear API, and the key allows 2,500 requests
        per hour.
    *   The rules that need the whole task tree would require pulling many
        tasks per view.
    *   Responses would be slow, because the work is about 330 requests to page
        through every task.
    *   It needs OAuth setup.
*   **Reason for Rejection**: Rejected by author: its separate frontend and
    backend are kept, but the data is copied nightly instead, and sign-in is out
    of scope for now. Live reads can't apply the full-tree rules within Linear's
    limits.

### Alternative C: A static page that filters a nightly data file in the browser

*   **Pros**: No backend or database to run. It's the fastest to build. The
    first mockup worked this way.
*   **Cons**:
    *   Every viewer downloads the whole data set, about 0.5 MB for the tagged
        projects and much more for the whole workspace.
    *   The counting rules would live in browser code.
    *   There is nowhere to run the sync, the cooldown or the Refresh button.
    *   It doesn't leave room for hours from a second source.
*   **Reason for Rejection**: Rejected by author: the author asked for a Go
    (Kratos) backend with CockroachDB and Docker Compose. The nightly copy idea
    is kept, stored in CockroachDB instead of a file.

### Alternative D: An MUI frontend with one page per level

*   **Pros**: Mature components. Every level has its own URL and page.
*   **Cons**:
    *   It looks unlike Linear, which stakeholders use every day.
    *   Moving between segments, initiatives and projects means opening one page
        after another, with no way to search for a project.
*   **Reason for Rejection**: Rejected by author after reviewing the MUI
    mockup: the author asked for shadcn/ui with Linear styling, on one page with
    a filterable left menu.

### Alternative E: API shaped as one "view" endpoint per scope

*   **Pros**: One request per screen. All the rules sit behind a single
    function per scope.
*   **Cons**: The response is tied to this page's layout. A new panel or a
    second consumer means changing the response shape.
*   **Reason for Rejection**: Rejected by author in favor of one endpoint per
    level, which is plain REST and easier to cache and test level by level.

### Alternative F: API shaped as a query endpoint like Linear Insights

*   **Pros**: New panels, the hours measure and other consumers would need no
    API change.
*   **Cons**:
    *   It needs the most backend work.
    *   The segment-split and sub-task rules must hold for every combination of
        groupings, which multiplies the tests needed.
    *   Each page makes 6–8 requests.
*   **Reason for Rejection**: Rejected by author: no consumer other than this
    page is planned. One endpoint per level was chosen.

### Alternative G: Count a multi-segment project in full under each segment

*   **Pros**: Each segment's number matches the project's real size.
*   **Cons**: The segment totals add up to more than the real total, so shares
    can't be shown as percentages of one whole. 32 of the 60 tagged projects are
    affected.
*   **Reason for Rejection**: Rejected by author: an even split was chosen, so
    the shares add up to 100%. Project and initiative views still show real,
    unsplit totals.

### Alternative H: Take segments from task labels as well as project labels

*   **Pros**: It would pick up 344 tasks, mostly from the OS and DVOS teams,
    that carry the task label `b2b` but sit in untagged projects or in no
    project. That is 183 pts in November 2026.
*   **Cons**:
    *   Task labels and project labels are separate lists in Linear, and only a
        lowercase `b2b` exists as a task label.
    *   Counting it would let individual task tags override the project-level
        segment decision.
    *   It would also hide the fact that those projects are not tagged.
*   **Reason for Rejection**: Rejected by author: segments come from project
    labels only. Segment labels on tasks are reported as a warning, with links
    to fix the project.

### Alternative I: Count a task in a range if it was updated in that range

*   **Pros**: It's the simplest rule, using a single date field.
*   **Cons**: Any edit counts, including bulk relabeling. On the real data it
    pulled 200 old, already-finished tasks into November because of bulk edits
    on Nov 13 and 16. It would also need an update log to cover past ranges.
*   **Reason for Rejection**: Rejected by author: a task counts only if it was
    created, completed or in progress during the range.

### Alternative J: Show every initiative

*   **Pros**: Nothing is hidden.
*   **Cons**: Sprints ("46.Cloud Sprint …"), buckets ("Cloud Kanban", "2026
    Roadmap") and team groupings are stored as initiatives. The same project
    would appear three or four times on a segment view.
*   **Reason for Rejection**: Rejected by author: only initiatives whose name
    starts with `[W]` or `[WS]` are shown.

### Alternative K: Show elapsed time (from "In Progress" to "Done") as time spent

*   **Pros**: It comes from Linear at no extra cost and is measured in time
    units.
*   **Cons**: It measures calendar time, not work. Weekends and tasks worked in
    parallel inflate it.
*   **Reason for Rejection**: Rejected by author: effort is estimate points.
    Logged hours will come later from a separate source.

--------------------------------------------------------------------------------

## Non-Functional Requirements

*   **Security and privacy.**
    *   There is no sign-in, so the network is the only access control. The
        dashboard shows task titles and people's names, so it must not be
        reachable from the internet. Compose publishes only the frontend port.
        CockroachDB and the backend stay on the internal Compose network.
    *   The Linear key lives only in the backend's environment. It is never
        logged, never returned by the API, and never in the browser.
    *   Rotate the key that was pasted into chat before first deployment.
*   **Load on Linear.** One full sync makes about 360 requests, within the limit
    of 2,500 per hour. The 15-minute cooldown on Refresh caps manual syncs at 4
    per hour, so the worst case stays under the limit.
*   **Speed.** Each view is computed from at most a few thousand rows, with a
    target under 200 ms per endpoint. A full sync is expected to take 5–10
    minutes.
*   **Rollout and rollback.** This is a new standalone system, and nothing else
    depends on it. Rolling back means stopping the containers. Data loss is not
    a risk, because the database is a copy that the next sync rebuilds.

### Future Improvements

*   **Hours.** Add a `time_entries` source and a `measure=hours` option to the
    same summary blocks.
*   **Sign-in with Linear.** Restrict viewers to members of the PARKEE
    workspace.
*   **Frozen months.** Store a snapshot of each reported month, so later
    re-estimates don't change it.
*   **Coverage figure.** Show "tagged work is X% of all estimated effort" on the
    overview. The author decided against this for now, on 2026-11-17.

--------------------------------------------------------------------------------

## Detailed Implementation Plan

Mechanical choices made in this plan. The author can change any of them before
building starts.

*   **Go module:** `linear-dashboard/backend`, with Go 1.25+ and Kratos v2.
*   **Wiring:** `google/wire`, per the `golang-google-wire` skill.
*   **Go conventions (author's decision):** the builder must load the
    `cc-skills-golang` skills before writing Go code. Required:
    `golang-project-layout`, `golang-naming`, `golang-error-handling`,
    `golang-database`, `golang-testing`, `golang-safety`, `golang-context`,
    `golang-google-wire` and `golang-grpc`. **Where they disagree with Kratos's
    `kratos-layout` template, the skills win.** The folder renames that follow
    from this are:

    | kratos-layout | This project | Reason |
    |---|---|---|
    | `internal/biz` | `internal/report` | The skills want package names that describe their content. This package holds the counting rules and the report use cases. |
    | `internal/data` | `internal/store` | `data` is a vague name. The package is the CockroachDB store. |
    | `internal/service` | `internal/handler` | The skills' service layout uses `handler/` for request handlers. |
    | `internal/server` | `internal/app` | The skills' layout uses `app/` for application startup, servers and the scheduler. |
    | `internal/conf` | `internal/config` | The skills' layout uses `config/`. |
    | (new) `internal/sync` | `internal/snapshot` | `sync` would shadow Go's standard-library `sync` package. |

    Kratos itself only requires the `api/` protobuf contract and its transport
    packages. The folder names are a template convention, so renaming them
    breaks nothing.
*   **Protobuf:** generated with `buf`. googleapis comes in as a buf dependency,
    so no `third_party/` folder is needed. Generated code is committed.
*   **Database:** `pgx/v5` with hand-written SQL, and no ORM or sqlc.
    Migrations use `pressly/goose` with embedded SQL files and run at startup.
*   **Scheduler:** `robfig/cron/v3` with the time zone from config.
*   **Linear client:** hand-written on `net/http` with typed response structs.
    No GraphQL code generation.
*   **Frontend:** built from `mockup/`. The mockup folder is deleted once the
    frontend matches it.

### Target Files to Create/Modify

#### Repository root

##### 1. `.gitignore` (Create)

*   **Type / Scope**: Repository configuration.
*   **Proposed Changes**: Ignore `.env`, `node_modules/`, `.next/`, Go build
    output, `.superpowers/` and any `data/` snapshot folders. Real Linear data
    must never be committed.

##### 2. `.env.example` (Create)

*   **Type / Scope**: Configuration template.
*   **Proposed Changes**: Lists `LINEAR_API_KEY=` (empty), `SYNC_CRON=0 1 * * *`,
    `SYNC_TZ=Asia/Jakarta`, `SYNC_COOLDOWN=15m` and `FRONTEND_PORT=3000`.
    Developers copy it to `.env`.

##### 3. `docker-compose.yml` (Create)

*   **Type / Scope**: Deployment.
*   **Proposed Changes**:
    *   **`cockroach`**: image `cockroachdb/cockroach` (pinned version),
        `start-single-node --insecure`, a named volume, and a health check
        through `cockroach sql -e "select 1"`. No port is published.
    *   **`backend`**: built from `backend/Dockerfile`. It waits until
        `cockroach` is healthy, gets `DATABASE_URL` pointing at
        `postgresql://root@cockroach:26257/dashboard?sslmode=disable`, and loads
        its other settings from `.env`. No port is published.
    *   **`frontend`**: built from `frontend/Dockerfile`, with
        `BACKEND_URL=http://backend:8000`. It publishes `${FRONTEND_PORT}:3000`.
    *   Restart policy `unless-stopped` for all three.

##### 4. `Makefile` (Create)

*   **Type / Scope**: Developer commands.
*   **Proposed Changes**:
    *   `make up` and `make down` wrap `docker compose`.
    *   `make proto` runs `buf generate` in `backend/`.
    *   `make test` runs `go test ./...` and the frontend typecheck and lint.
    *   `make smoke` runs `scripts/smoke.sh`.

##### 5. `README.md` (Create)

*   **Type / Scope**: Documentation.
*   **Proposed Changes**: Explains:
    *   what the dashboard counts, linking the counting-rules table in the
        design;
    *   setup: copy `.env.example`, set the key, run `make up`;
    *   where to see sync status;
    *   how to fix tagging warnings in Linear.

##### 6. `scripts/smoke.sh` (Create)

*   **Type / Scope**: End-to-end check.
*   **Proposed Changes**: Against a running stack, uses `curl` and `jq` to call
    each endpoint for 1–30 November 2026. It checks:
    *   HTTP 200 on every endpoint;
    *   segment percentages add up to 100 ± 0.1;
    *   the sum of segment points equals the overview's tagged total, ± 0.1;
    *   `/v1/tasks` points for one contributor cell equal that cell's value;
    *   a second `POST /v1/sync` within 15 minutes returns 409 or 429.

##### 7. `docs/superpowers/specs/2026-11-17-linear-dashboard-design.md` (Modify)

*   **Type / Scope**: Superseded spec.
*   **Proposed Changes**: Add a line at the top saying it is superseded by the
    finalized design document, with a link. Remove the stale line about seeding
    the update log in §9, because the update log was dropped.

##### 8. `mockup/` (Delete, at the end)

*   **Type / Scope**: Throwaway prototype.
*   **Proposed Changes**: Delete the whole folder once the frontend reaches
    parity: the six CUJs pass against real backend data. The folder holds a
    snapshot of real Linear data in `mockup/data/`, so deleting it also removes
    that copy.

#### Backend (`backend/`)

##### 9. `backend/go.mod`, `backend/go.sum` (Create)

*   **Type / Scope**: Go module.
*   **Proposed Changes**: Module `linear-dashboard/backend`. Dependencies:
    Kratos v2, wire, pgx/v5, goose, robfig/cron/v3, protobuf and gRPC.

##### 10. `backend/buf.yaml`, `backend/buf.gen.yaml` (Create)

*   **Type / Scope**: Protobuf build.
*   **Proposed Changes**: Depends on `buf.build/googleapis/googleapis`.
    Generates with `protoc-gen-go`, `protoc-gen-go-grpc` and
    `protoc-gen-go-http` (Kratos), writing into `api/` and `internal/config/`.

##### 11. `backend/api/dashboard/v1/dashboard.proto` (Create) and generated `*.pb.go`, `*_grpc.pb.go`, `*_http.pb.go` (Generated)

*   **Type / Scope**: API contract.
*   **Proposed Changes**: `service Dashboard` has these RPCs, each with a
    `google.api.http` GET mapping as in the Technical Plan table:
    *   `GetMenu`;
    *   `GetOverview`;
    *   `GetSegment`;
    *   `GetInitiative`;
    *   `GetProject`;
    *   `ListTasks`;
    *   `ListMissingEstimates`;
    *   `ListSegmentLabelWarnings`;
    *   `ListUntaggedProjects`.

    `ListMissingEstimates` takes optional `segment`, `initiative_id`,
    `project_id` and `all`. At most one of the three scope fields may be set;
    otherwise it returns `INVALID_SCOPE` (400).

    `service Sync` has `StartSync` (POST `/v1/sync`) and `GetSyncStatus`.

    Shared messages:
    *   `Range{from, to}`, with dates as `YYYY-MM-DD` strings;
    *   `SummaryBlock{measure, total, by_status, by_priority[], by_label[],
        unlabeled, contributors, tasks_total, tasks_missing}`;
    *   `ContributorTable{dim, columns[], rows[{assignee, total, cells
        map<string, Cell{pts, count}>, missing}]}`;
    *   `ProjectRow{id, name, url, state, segments[], pts, share_pts,
        split_of, status, tasks, missing}`;
    *   `TaskRow{id, identifier, title, url, state_name, status_bucket,
        priority, assignee, labels[], estimate (optional), counted_pts,
        deferred, missing, project_id, depth}`;
    *   `SyncStatus{state, last_success_at, last_run{started_at, finished_at,
        status, error}}`. This is shown in the header of every view.

    Errors use Kratos `errors` reasons:
    *   `INVALID_RANGE` → 400;
    *   `NOT_FOUND` → 404;
    *   `SYNC_RUNNING` → 409;
    *   `SYNC_COOLDOWN` → 429.

##### 12. `backend/internal/config/config.proto` (Create) and `config.pb.go` (Generated)

*   **Type / Scope**: Typed configuration.
*   **Proposed Changes**: `Bootstrap{server{http.addr, grpc.addr},
    data{database_url}, linear{api_key, page_size}, sync{cron, tz, cooldown}}`.
    Values come from environment variables, overriding `configs/config.yaml`.

##### 13. `backend/configs/config.yaml` (Create)

*   **Type / Scope**: Default configuration.
*   **Proposed Changes**: HTTP `:8000`, gRPC `:9000`, `page_size: 100`, cron
    `0 1 * * *`, time zone `Asia/Jakarta`, cooldown `15m`. It contains no
    secrets.

##### 14. `backend/cmd/server/main.go`, `wire.go`, `wire_gen.go` (Create)

*   **Type / Scope**: Entry point.
*   **Proposed Changes**:
    1.  Load the config, open the pgx pool and run the goose migrations.
    2.  Build the services through wire.
    3.  Start the HTTP and gRPC servers and the cron scheduler.
    4.  If `sync_runs` has no successful row, start a sync in the background.
    5.  On SIGTERM, shut down cleanly: stop the scheduler, cancel a running
        sync through its context, and close the pool.

##### 15. `backend/internal/app/http.go`, `grpc.go`, `server.go` (Create)

*   **Type / Scope**: Transport.
*   **Proposed Changes**: Register both services. Add recovery and logging
    middleware. Requests are logged without query secrets.

##### 16. `backend/internal/handler/dashboard.go`, `sync.go` (Create)

*   **Type / Scope**: API handlers.
*   **Proposed Changes**: Thin handlers that:
    *   check `from <= to`, that both dates are valid, and that `dim` is one of
        `priority`, `status` or `segment`;
    *   call the use cases in `report`;
    *   map domain results to protobuf messages.

    No counting logic lives here.

##### 17. `backend/internal/report/model.go` (Create)

*   **Type / Scope**: Domain types.
*   **Proposed Changes**: `Segment` (an enum with B2B, B2G, B2BNG and B2C, each
    with a display label), `StatusBucket`, `Task`, `Project`, `Initiative`,
    `DateRange` (from and to as dates in a time zone), `Summary`,
    `ContributorTable` and `Scope` (Overview, Segment, Initiative or Project,
    with an ID).

##### 18. `backend/internal/report/segments.go` (Create)

*   **Type / Scope**: Rule.
*   **Proposed Changes**:
    *   `NormalizeLabel(name)`: lower-case, then drop everything except a–z and
        0–9.
    *   `SegmentsOf(labels)`: returns a sorted, de-duplicated list of segments
        using the aliases `b2b`, `b2g`, `b2bng`, `b2bnongroup` and `b2c`.
    *   `SplitWeight(project)`: returns 1 ÷ the number of segments.

##### 19. `backend/internal/report/effort.go` (Create)

*   **Type / Scope**: Rule, run during sync.
*   **Proposed Changes**: `Derive(tasks, teams)` walks the full parent tree
    once, memoizing results. For each task it sets:
    *   `HasEstimatedDescendant`;
    *   `EffectivePts`, which is 0 if canceled, a duplicate or deferred, and
        otherwise the estimate or 0;
    *   `IsMissingEstimate`;
    *   `TaskSegmentLabels`.

    It must handle cycles defensively: if a task is visited twice on one path,
    stop and treat it as having no descendants.

##### 20. `backend/internal/report/daterange.go` (Create)

*   **Type / Scope**: Rule.
*   **Proposed Changes**:
    *   `ParseRange(from, to, tz)` returns the half-open interval
        `[from 00:00, to+1 day 00:00)` in the time zone.
    *   `InRange(task, r)` is true if the task was created in the interval, was
        completed in it, or was started before the interval ends and finished
        (completed or canceled) on or after its start.

    The same conditions are expressed in SQL in `store/read.go`. Both must match
    the test table.

##### 21. `backend/internal/report/status.go`, `initiatives.go`, `labels.go` (Create)

*   **Type / Scope**: Rules.
*   **Proposed Changes**:
    *   `BucketOf(stateType)`.
    *   `IsVisibleInitiative(name)`: the name starts with `[W]` or `[WS]`,
        ignoring case and spaces.
    *   `LabelShares(tasks, weight)`: returns per-label points and the unlabeled
        points separately. Labels display as `Group / Name`.

##### 22. `backend/internal/report/summary.go`, `contributors.go` (Create)

*   **Type / Scope**: Aggregation.
*   **Proposed Changes**:
    *   `Summarize(tasks, weightFn)` builds a `Summary`. Canceled tasks count
        toward neither effort nor `tasks_total`.
    *   `Contributors(tasks, weightFn, dim)` builds the table. For `dim=segment`,
        a task's points are split across its project's segments.
    *   `CellTasks(tasks, assignee, dim, column)` returns exactly the tasks
        behind one cell, so the drill-down always matches the number clicked.

##### 23. `backend/internal/report/views.go` (Create)

*   **Type / Scope**: Use cases.
*   **Proposed Changes**: One method per endpoint on a `DashboardUsecase` that
    depends on a `Reader` interface. `Overview`, `Segment`, `Initiative`,
    `Project`, `Tasks`, `MissingEstimates`, `SegmentLabelWarnings`,
    `UntaggedProjects` and `Menu` each:
    1.  ask the reader for the in-range tasks and projects for the scope;
    2.  apply the segment weight (1 ÷ N) only for `Segment` scopes;
    3.  group initiatives under a segment, putting projects with no visible
        initiative under "No initiative";
    4.  sort contributors by points, then by task count.

##### 24. `backend/internal/report/*_test.go` and `backend/internal/report/testdata/snapshot.json` (Create)

*   **Type / Scope**: Unit and golden tests.
*   **Proposed Changes**:
    *   **Table tests for every rule:**
        *   label spellings: `B2B`, ` b2b `, `B2B NG`, `b2b-ng`, `B2B Non
            Group`, `B2BX` → none;
        *   segment split for 1, 2 and 3 segments;
        *   parent with an estimated child, a canceled estimated child, a
            grandchild estimate, and no estimates;
        *   missing-estimate exclusions;
        *   range boundaries at 00:00 Jakarta time on the first and last day;
        *   a task started before the range and still open;
        *   a task completed the day before the range;
        *   initiative name variants;
        *   label split and "No label";
        *   the contributor table for each `dim`;
        *   `CellTasks` points equal the cell value.
    *   **A golden test** loads `snapshot.json`, an anonymized copy of the
        60-project snapshot from 2026-11-17, with titles replaced and names
        hashed. For November 2026 it expects the mockup's results:
        *   B2B 364.3 pts;
        *   B2G 18.3 pts;
        *   B2B NG 0 pts;
        *   B2C 683.3 pts;
        *   1,064 pts in total;
        *   160 unestimated out of 559;
        *   115 task-label warnings.

##### 25. `backend/internal/linear/client.go`, `queries.go`, `types.go` (Create)

*   **Type / Scope**: Linear API client.
*   **Proposed Changes**:
    *   **Request:** `Do(ctx, query, vars, out)` POSTs to
        `https://api.linear.app/graphql` with the `Authorization: <key>` header.
    *   **Errors:** it treats a GraphQL `errors` array as a failure. On
        `RATELIMITED` or HTTP 429 it waits with increasing delays (1 s, 2 s,
        4 s, 8 s, 16 s, plus jitter), up to 5 times.
    *   **Paging:** `Paginate[T](ctx, query, pageSize, fn)` follows
        `pageInfo.endCursor` and calls `fn` once per page.
    *   **Queries:** one per entity, all with `includeArchived: true`.
        *   Tasks select `id`, `identifier`, `title`, `url`, `estimate`,
            `priority`, `createdAt`, `startedAt`, `completedAt`, `canceledAt`,
            `archivedAt`, `team{id}`, `state{name type}`, `assignee{id}`,
            `project{id}`, `parent{id}` and `labels(first:20){nodes{id}}`.
        *   Labels are fetched separately with group names, so task pages stay
            under 10,000 complexity.

##### 26. `backend/internal/linear/client_test.go`, `testdata/*.json` (Create)

*   **Type / Scope**: Unit tests with an `httptest` server.
*   **Proposed Changes**: Cover:
    *   paging across 3 pages;
    *   a GraphQL error surfacing as a Go error;
    *   a rate-limit response followed by success;
    *   that the key appears only in the request header and never in error
        messages.

##### 27. `backend/internal/snapshot/runner.go`, `scheduler.go` (Create)

*   **Type / Scope**: Sync job.
*   **Proposed Changes**:
    *   **`Runner.Run(ctx)`:**
        1.  Insert a `sync_runs` row with status `running`. A unique violation
            becomes `SYNC_RUNNING`. A last start within the cooldown becomes
            `SYNC_COOLDOWN`.
        2.  Fetch and upsert, in this order: teams, users, labels, initiatives,
            projects with their links, then tasks.
        3.  Load all stored tasks, call `report.Derive` and write the derived
            columns in batches of 500. This step runs even if step 2 failed,
            using a fresh context with a 5-minute timeout, so that a canceled
            fetch doesn't also cancel the derive.
        4.  Only if step 2 succeeded: mark rows unseen in this run as deleted.
        5.  Mark the run `succeeded` with its stats. If step 2 or step 3
            failed, mark it `failed` with the error text.
    *   **`Scheduler`:** wraps cron in the configured time zone and calls `Run`
        with a 30-minute timeout. `StartSync` from the API calls `Run` in a
        goroutine and returns at once with the new run ID.

##### 28. `backend/internal/snapshot/runner_test.go` (Create)

*   **Type / Scope**: Integration test, build tag `integration`.
*   **Proposed Changes**: Runs against a fake Linear server and a real
    CockroachDB started by `docker compose`. Checks:
    *   the first run inserts the rows;
    *   a second run with one task removed marks it deleted;
    *   a run whose fetch fails halfway marks nothing deleted, still
        recomputes the derived fields, and ends `failed`, while
        `last_success_at` keeps the previous success. In the test, a parent
        whose new estimated sub-task was stored before the failure counts as 0;
    *   two runs at once give `SYNC_RUNNING`.

##### 29. `backend/internal/store/data.go`, `migrate.go` (Create)

*   **Type / Scope**: Database setup.
*   **Proposed Changes**: Build the pgx pool from `DATABASE_URL`, with a retry
    loop of 30 attempts while CockroachDB starts. Run goose with the embedded
    `migrations/` folder.

##### 30. `backend/internal/store/migrations/0001_init.sql` (Create)

*   **Type / Scope**: Schema.
*   **Proposed Changes**:
    *   **Tables**, as in the Technical Plan data model: `teams`, `users`,
        `labels`, `initiatives`, `projects`, `project_labels`,
        `initiative_projects`, `issues`, `issue_labels` and `sync_runs`.
    *   **Indexes**, as listed in the Technical Plan.
    *   **The partial unique index**
        `sync_runs (status) WHERE status = 'running'`, which prevents two syncs
        running at once.

##### 31. `backend/internal/store/write.go` (Create)

*   **Type / Scope**: Sync repository.
*   **Proposed Changes**:
    *   **Upserts:** `UpsertTeams`, `UpsertUsers`, `UpsertLabels`,
        `UpsertInitiatives`, `UpsertProjects` and `UpsertIssues`. Each takes one
        page and does a multi-row `UPSERT` setting `last_seen_run`.
    *   **Links:** link tables are replaced for the parent IDs in the page, in
        the same transaction as the page.
    *   **Derived columns:** `WriteDerived(batch)`.
    *   **Deletions:** `MarkDeleted(runID)` runs one `UPDATE … SET deleted =
        true WHERE last_seen_run <> $1` per table.
    *   **Run tracking:** `StartRun`, `FinishRun` and `LastSuccess`.

##### 32. `backend/internal/store/read.go` (Create)

*   **Type / Scope**: Read repository (implements `report.Reader`).
*   **Proposed Changes**:
    *   **`TasksForProjects(projectIDs, range)`**: one query over non-deleted
        issues joined to their labels. It applies the range rule in SQL as
        `created_at ∈ R OR completed_at ∈ R OR (started_at < R.end AND
        COALESCE(completed_at, canceled_at, 'infinity') >= R.start)`.
    *   **Other reads:** `ProjectsInScope`, `VisibleInitiatives`, `Menu`,
        `TasksWithSegmentLabels(range)`, `UntaggedProjects(range)` and
        `MissingEstimateTasks(range, scope, all)`. The last one filters on the
        stored `is_missing_estimate` flag. With `all=true` it also takes tasks
        in untagged projects and tasks with no project.

##### 33. `backend/internal/store/read_test.go` (Create)

*   **Type / Scope**: Integration test, build tag `integration`.
*   **Proposed Changes**: Seeds the database from `report/testdata/snapshot.json`.
    Checks that the SQL range rule returns the same task set as `report.InRange`
    for 6 ranges: this month, last month, the last 3 months, all time, a single
    day, and an empty range.

##### 34. `backend/Dockerfile` (Create)

*   **Type / Scope**: Container.
*   **Proposed Changes**: A multi-stage build. The builder stage uses the
    official Go image and runs `go build`. The runtime stage is a distroless
    static image with a non-root user. It exposes 8000 and 9000.

#### Frontend (`frontend/`, built from `mockup/`)

##### 35. `frontend/package.json`, `package-lock.json`, `tsconfig.json`, `components.json`, `eslint.config.mjs`, `postcss.config.mjs`, `next-env.d.ts` (Create, copied from `mockup/`)

*   **Type / Scope**: Project setup.
*   **Proposed Changes**: Copy from the mockup, rename the package to
    `effort-dashboard`, and keep the pinned Next.js, React, Base UI, Recharts and
    shadcn versions. Remove the `cn` package, which the mockup doesn't use.

##### 36. `frontend/next.config.ts` (Create)

*   **Type / Scope**: Next.js config.
*   **Proposed Changes**: Add `rewrites()` so that `/api/v1/:path*` goes to
    `${BACKEND_URL}/v1/:path*`. Set `output: "standalone"` for the Docker image.

##### 37. `frontend/app/layout.tsx`, `app/globals.css` (Create, from mockup)

*   **Type / Scope**: Shell and theme.
*   **Proposed Changes**: Keep the Inter font and the Linear light-theme tokens
    as in the mockup: the `--seg-*`, `--st-*` and `--warn` colors. The theme
    provider stays fixed on `light`.

##### 38. `frontend/app/page.tsx` (Create)

*   **Type / Scope**: The single page (a server component).
*   **Proposed Changes**:
    1.  Read `searchParams` (`scope`, `from`, `to`, `dim`, `seg`).
    2.  Call `lib/api.ts` on the server to get the menu and the scope's endpoint
        in parallel.
    3.  Render `AppSidebar` and `Dashboard` with the data.

    A backend error shows the shadcn `Alert` with the error reason. An empty
    result shows `Empty`.

##### 39. `frontend/lib/api.ts`, `lib/types.ts`, `lib/format.ts` (Create)

*   **Type / Scope**: Data access.
*   **Proposed Changes**:
    *   **`types.ts`**: TypeScript types that mirror the proto JSON (camelCase,
        as emitted by Kratos).
    *   **`api.ts`**: typed fetchers. Server-side calls use `BACKEND_URL`.
        Browser-side calls, which are only `tasks`, use `/api/v1`. Data that
        can change is fetched with `cache: "no-store"`.
    *   **`format.ts`**: `fmt` and `pct`, moved out of the mockup's `model.js`.

##### 40. `frontend/hooks/use-dash.ts` (Create, from mockup)

*   **Type / Scope**: URL state.
*   **Proposed Changes**: Same as the mockup: read the URL state and build
    links. `parseScope` moves here from `model.js`, with checking only. The
    server does all the counting.
    *   Add `via`. Links from a segment or initiative view carry it along.
    *   `breadcrumbFor(scope, via, menu)` checks `via` against the item's
        segments and initiatives, using `/v1/menu` data, before showing it.

##### 41. `frontend/components/app-sidebar.tsx`, `dashboard.tsx`, `contributors-pivot.tsx`, `panels.tsx`, `task-list.tsx`, `linear-icons.tsx` (Create, from mockup)

*   **Type / Scope**: UI.
*   **Proposed Changes**:
    *   Same layout and behavior as the mockup.
    *   Replace every import from `lib/model` with props filled from API
        responses.
    *   `contributors-pivot` loads drill-down rows from `/api/v1/tasks` when a
        cell is clicked, and shows a `Skeleton` while loading.
    *   The Refresh button calls `POST /api/v1/sync` and shows a `Spinner`
        while it runs. It polls `/api/v1/sync/status` every 5 seconds until
        the run is done, then calls `router.refresh()`. A 409 or 429 response
        is shown as a toast.
    *   The Missing estimates view gets a segment / initiative / project
        filter and an "All projects" switch. Every "N of M unestimated" card
        links to it already filtered to the current view.
    *   Every list of initiatives always shows the overlap note: under the
        "Initiatives and projects" table on segment views, and on the
        initiative view next to the total.
    *   Remove the mockup banner.
    *   Add the overview's untagged line, showing the project count only and
        linking to the Untagged projects page.

##### 42. `frontend/components/ui/*` (Create, copied from mockup)

*   **Type / Scope**: shadcn components.
*   **Proposed Changes**: Copy them unchanged. Add `spinner` and `toast` with
    `npx shadcn add`, for the Refresh flow.

##### 43. `frontend/Dockerfile` (Create)

*   **Type / Scope**: Container.
*   **Proposed Changes**: A multi-stage build with `npm ci`, `next build` and
    the standalone output on a Node LTS slim image, running as a non-root user.
    It exposes 3000.

### Verification Plan

*   **Build:**
    *   `cd backend && buf lint && buf generate && go build ./...`;
    *   `cd frontend && npm run build`;
    *   `docker compose build`.
*   **Unit tests:** `cd backend && go test ./...`. This includes every
    counting-rule table test and the November 2026 golden totals.
*   **Integration tests:** `cd backend && go test -tags integration ./...`
    against the Compose CockroachDB. It covers the sync runner and the
    SQL-versus-Go range check.
*   **Static analysis:** `golangci-lint run` in `backend/`, and
    `npm run lint && npm run typecheck` in `frontend/`.
*   **Coverage:** at least 90% statement coverage for `internal/report`, which
    holds every counting rule. No target for the other packages.
*   **End to end:**
    1.  Run `make up` with a real (rotated) key.
    2.  Wait for the first sync to succeed, as shown by `/v1/sync/status`.
    3.  Run `make smoke`.
    4.  Walk through CUJs 1–6 by hand in the browser.
*   **Reconciliation:** compare the backend's November 2026 numbers with
    Linear for 3 hand-picked projects: one with a single segment, one with
    several segments, and one with nested sub-tasks. Every difference must be
    explained by a rule in the counting-rules table.

--------------------------------------------------------------------------------

*Drafted with EGM*
