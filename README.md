# PARKEE Effort Dashboard

Shows how engineering effort is spread across the B2B, B2G, B2B NG and B2C
segments. The data comes from Linear and is copied every night.

Effort is **Linear estimate points, not hours**. How each number is counted is
set out in [the design](docs/designs/effort-dashboard.md). The section
"Decisions after review" overrides the rest of that document.

## Run it

```sh
cp .env.example .env      # set LINEAR_API_KEY (production: a dedicated service account)
make up                   # docker compose up --build
open http://localhost:3300
```

The first start copies the whole Linear workspace, which takes about 4 minutes
for about 32,000 tasks. The page shows "first sync is still running" until it
finishes. After that, a sync runs every night at 01:00 and every weekday at
17:00 Asia/Jakarta (`SYNC_CRON`, several schedules separated by `;`). Two
buttons in the header sync on demand:

| Button | What it copies | Time | At most once every |
|---|---|---|---|
| **Refresh** | tasks changed in Linear since the last successful sync, plus teams, users, labels, initiatives and projects | seconds | 2 minutes (`REFRESH_COOLDOWN`) |
| **Resync** | the whole workspace | about 4 minutes | 15 minutes (`SYNC_COOLDOWN`) |

Linear does not mark a task as changed when it is deleted, so Refresh misses
most deletions: a deleted task stays until the next Resync or scheduled sync,
which are always full. Ignored tasks are left out before anything is counted, in
both kinds of sync.

| Service | What it does |
|---|---|
| `cockroach` | Single-node CockroachDB holding the copy and the sync history. Not published. |
| `backend` | Go with Kratos v3. It runs the Linear sync and the counting rules, and serves `/v1/*` as JSON. Not published. |
| `frontend` | Next.js with shadcn/ui, on port 3300. `/api/v1/*` is passed through to the backend. |

## Custom dashboards

Under **Dashboards** in the left menu you can build your own dashboards from
insights, modeled on [Linear Insights](https://linear.app/docs/insights).

*   **An insight** plots **Effort** (pts) or **Issue count**:
    *   **sliced** by assignee, priority, status, segment, label, team,
        project, initiative, created month or completed month;
    *   optionally **split** by a second dimension;
    *   shown as a **bar chart**, a **table** or a **metric**.
*   **Clicking any bar, cell or metric** lists the tasks behind it.
*   **Filters** for segments, status, priority, teams and untagged work can be
    set on the dashboard, on each insight, or both.
*   **Dashboards are shared:** everyone who can open the app can see and edit
    them.

**Bug and cycle insights** use time measures, in days:

| Measure | From → to | Which tasks |
|---|---|---|
| **Cycle time** | started → completed | completed in the range |
| **Lead time** | created → completed | completed in the range |
| **Issue age** | created → last sync | still open |
| **Triage time** | entered triage → triaged | triaged in the range |

*   Each one is shown as the **median** per group, with the 75th and 95th
    percentiles, or as a **scatter** where clicking a point opens the task.
*   The **"Labels"** filter takes exact label names, or "label contains"
    text. Typing `bug` matches `Bug` and labels such as
    `New Bug - Agent` and `Repeat Bug - Agent`, and so on.
*   A ready-made **"Bug & cycle performance"** dashboard shows bug volume, open
    bugs, resolution time, cycle time, age of open bugs and triage time.

**Ready-made dashboards:**

| Dashboard | Insights |
|---|---|
| **Bug & cycle performance** | bug volume, open bugs, resolution time, cycle time, age of open bugs, triage time |
| **Delivery & throughput** | tasks completed per month (bugs vs other work), effort delivered by segment, lead time trend, cycle time by estimate size (how good the estimates are), throughput by team |
| **Team health** | work in progress per person, age of in-progress work, bug share by team, hotfixes per month, effort by team and segment |

**Feature effort by label** (Engineering effort dashboard):

*   **Split effort:** a task with 2 labels gives each label half its effort.
*   **Shared count:** each label row shows how many of its tasks are shared
    ("65 shared").
*   **Task badge:** in the drill-down, a shared task carries a badge such as
    "2 labels".
*   **Narrowing to features:** pick your feature labels in the insight's
    Labels filter. The grouping then uses only those labels, so a feature is
    not split with process labels such as "Release This Sprint".

**Two extra dimensions:**

*   **Work type:** bugs and hotfixes vs other work. A task is a bug when one
    of its labels mentions "bug" or "hotfix".
*   **Estimate size:** tasks grouped by their estimate in points.

Effort follows the same rules as the rest of the app:

*   canceled tasks don't count;
*   tasks whose title has one of the words in `IgnoredTaskKeywords`
    (`backend/internal/report/rules.go`) are left out entirely: Meeting, Meet,
    DISCUSS, DISCUSSION, RECAP, Check, Asses Issue, Assess, Assessment,
    OS_Data. Matching ignores case and needs a whole word, so "Checkout"
    still counts;
*   effort comes from the issue, not its sub-tasks: below an estimated issue,
    every sub-task counts as 0;
*   a shared project's effort is split evenly across its segments;
*   a task with several labels splits its effort evenly across them.

The issue count adds 1 to every group a task is in, as Linear does.

## Develop

```sh
make db           # CockroachDB only, on 127.0.0.1:26257
make backend      # go run the backend against it (reads .env)
make frontend     # next dev on :3300 against the local backend
make test         # Go unit and golden tests, frontend type check and lint
make smoke        # end-to-end checks against a running stack
make proto        # regenerate Go code from backend/api/dashboard/v1/dashboard.proto
make fixture      # re-export the anonymized golden-test fixture from the local database
```

Git hooks (Husky) run on every commit:

*   They block Linear keys, `.env` files and Linear data snapshots.
*   They run eslint, prettier and `tsc` on staged frontend files, and `gofmt`
    on staged Go files.

## Layout

```text
backend/
  api/dashboard/v1/     API contract (protobuf) and generated code
  cmd/server/           backend entry point (-healthcheck for container checks)
  cmd/fixture/          exports the anonymized golden-test fixture
  internal/report/      counting rules and views: pure Go, no I/O
  internal/linear/      Linear GraphQL client
  internal/snapshot/    sync: fetch everything, derive, replace in one transaction
  internal/store/       CockroachDB access and migrations
  internal/handler/     HTTP handlers
  internal/app/         wiring: config, scheduler, HTTP server
frontend/               Next.js app (single page, filterable left menu)
scripts/                smoke test, pre-commit secret check
```

## Fixing data in Linear

The housekeeping pages in the left menu list what to fix in Linear:

*   **Missing estimates:** tasks with no estimate, grouped by person.
*   **Segment label on task:** tasks tagged `b2b` and so on. Tag their project
    instead; task labels never count.
*   **Untagged projects:** projects with no segment label.
*   **Projects without initiative:** projects that are in no `[W]`/`[WS]`
    initiative. Draft initiatives (a word from `IgnoredInitiativeKeywords` in the name)
    don't count.
*   **Initiatives without projects:** initiatives with no linked project.

## Frozen work

Completed work is frozen, so reported effort does not change afterwards.
Linear cannot be locked, so after every sync the dashboard holds back changes
to frozen work and keeps counting the old value:

| Completed | Held back |
|---|---|
| task | a change to its estimate |
| project | a task moved into or out of it, created in it, or re-estimated while in it |
| `[W]`/`[WS]` initiative | a project linked to or unlinked from it |

A first estimate on a task that had none goes through without approval.
Estimates on unfinished tasks outside completed projects are never held.

Held changes are listed under **Pending changes** in the left menu. Anyone
with the app can decide them:

*   **Approve:** the dashboard takes the new value.
*   **Reject:** the dashboard sets Linear back to the old value and writes a
    comment there saying what was undone and why, with your reason if you give
    one. It does this with the `LINEAR_API_KEY`, so it appears as that key's
    owner: use a service account in production.

If Linear changes back on its own, the pending change is withdrawn.
