# Linear Effort Dashboard — Design

Date: 2026-11-17
Status: Draft, awaiting owner approval (builder check done, open questions answered)

## 1. Intent

PARKEE's management and stakeholders need to see where the team's effort goes, by business segment, and how far along the work is. Linear holds the work but has no time tracking, and Linear Insights cannot:

- split a project tagged with several segments evenly between them;
- avoid double counting when a parent task and its sub-tasks are both estimated;
- list the tasks that are missing estimates so the owner can chase each person.

This dashboard does those three things. Everything else should feel like Linear Insights.

Audience: management and stakeholders, all of whom have Linear accounts. It's read-only.

## 2. Goals

1. **Segment overview** is the landing page. It shows the four segments (B2B, B2G, B2B Non Group, B2C) with each one's effort share (%), project count, total effort, and done vs in progress.
2. **Drill-down:** segment → initiatives and projects in that segment → project → tasks. Each level shows:
   - total effort, split by status (done, in progress, not started);
   - contributors (assignees) and their effort;
   - effort by priority;
   - effort share (%) by task label, covering all labels and not only the segments;
   - the items one level down, with status.
3. A **date filter** on every page. It defaults to 1–30 November 2026 and also accepts any custom range.
4. A **Missing estimates** page that lists tasks with no estimate in the date range, grouped by assignee and linked to Linear, so the owner can remind each person.
5. Projects tagged with more than one segment are **marked as such** (for example "B2B + B2C") wherever they appear.
6. Data is copied from Linear **daily** into CockroachDB. The UI shows when the data was last refreshed.

## 3. Non-goals (for now)

- Hours or time tracking. Effort means **estimate points**. The UI always says "effort (pts)" and never "hours". Hours will come later from a separate source, and the schema and API should leave room for it (see §8.6). Nothing about hours gets built now.
- Elapsed or cycle time. The owner explicitly does not want it.
- Sign-in or access control. It runs on a trusted internal network for now.
- Writing anything back to Linear.
- Automated reminders. The Missing estimates page is enough; the owner reminds people manually.

## 4. Anti-goals (failure even if it "works")

- A number that reads as hours when it's estimate points.
- Effort silently understated. The unestimated count must be visible on every page, because about 50% of tasks in tagged projects have no estimate today.
- Segment percentages that add up to more than 100%, or projects counted twice in a segment total.
- Exposing the Linear API key to the browser.

## 5. What exists in Linear today (recon, 2026-11-17)

- Workspace: PARKEE. 984 projects (299 started, 325 completed, 228 backlog, 89 planned, 43 canceled).
- **The segments are project labels** in the label group `WS`: `B2B`, `B2C`, `B2G`, `B2B Non Group`. "B2B NG" in conversation means `B2B Non Group`, which no project uses yet.
- Only 60 of 984 projects carry a segment label (38 B2B, 54 B2C, 6 B2G), and **32 of the 60 carry more than one**.
- 501 projects belong to an initiative, and 213 belong to more than one.
- Tagged projects contain 1,051 tasks:
  - 517 are estimated;
  - 616 are sub-tasks;
  - 23 are unassigned.
- Most teams use the `linear` estimate scale. FE uses `fibonacci`, and many business teams don't use estimates at all. Points are summed as-is across teams.
- There is also a workspace-level **task** label `b2b`, on 350 tasks since July 2026. 344 of them are in untagged projects or in no project. It is **not** a segment source; those tasks show up as warnings (§7.1).

## 6. Approaches considered

| Approach | Trade-off | Verdict |
|---|---|---|
| A. Web app reading Linear live with "Sign in with Linear" | Live data and access control, but needs OAuth and hosting, and each page view hits the Linear rate limits. | Partly taken: a separate frontend and backend. |
| B. Daily data copy plus a page that filters in the browser | Simple and gentle on Linear, but the data is up to a day old and the published-page hosting was a poor fit. | Partly taken: the daily copy, stored in CockroachDB. |
| C. Linear Insights or Dashboards | No build needed, but it can't do segment splitting, sub-task de-duplication or the missing-estimates list. | Its UX is the reference model. |

**Chosen (owner's decision):** a combination of all three without their shortfalls.

- **Backend:** Go + Kratos and CockroachDB, with a daily sync from Linear.
- **Frontend:** separate Next.js app using Material Design (MUI).
- **Deployment:** everything runs under Docker Compose.
- **Access:** no sign-in for now.

## 7. Decided rules (owner-approved)

### 7.1 Segments
- A project belongs to segment S if it has the project label S from the `WS` group.
- **Label matching ignores case, spaces, hyphens and underscores** (owner's requirement). `B2B NG`, `b2b ng`, `B2B-NG` and `B2B Non Group` all mean B2B NG.
- Canonical segments and their display names:
  - `B2B` → "B2B"
  - `B2G` → "B2G"
  - `B2B Non Group` → "B2B NG"
  - `B2C` → "B2C"
- `B2B` never matches `B2B Non Group`.
- **A segment label on a task never counts** (owner's decision, 2026-11-17). Segments come from project labels only. Any task that carries a segment label itself (today that's the workspace task label `b2b`, on 350 tasks, mostly OS and DVOS) is listed as a **warning** instead:
  - a banner on the overview gives the count and pts;
  - a "Segment label on task" page groups the tasks by project, with a link to tag that project in Linear;
  - each group says whether the project's own segment already counts the tasks or whether they're not counted at all.
- **A multi-segment project's effort is split evenly:** project effort ÷ number of its segments goes to each segment. The project is badged with all its segments.
- **Overview percentages are taken over tagged work only.** A segment's % equals its effort divided by the sum of all four segments' effort, so the four add up to 100%.
- The overview also shows a separate "Untagged projects: N projects, X pts" line, linked to a list of untagged projects. That list is not part of the percentages.

### 7.2 Effort
- Effort is the sum of task estimates in points.
- **Parent and sub-task rule:**
  - If a task has sub-tasks and at least one of them is estimated, the parent's own estimate is ignored and only the sub-tasks count.
  - If none of its sub-tasks is estimated, the parent's estimate counts.
  - The rule applies recursively.
- Each sub-task counts towards its own project, assignee, priority and labels.
- **The parent/sub-task decision uses the full task tree, regardless of the date filter.** A parent whose estimated sub-tasks fall outside the range still contributes 0, so a range never double counts work that another range already counted. Canceled or deleted sub-tasks don't count as "estimated" for this rule. *(Claude's call.)*
- **Canceled tasks** are excluded from effort totals but still appear in task lists, marked canceled. *(Claude's call.)*
- **Unestimated tasks** count as 0 effort. Every page shows "N of M tasks unestimated", which links to the Missing estimates page pre-filtered to that scope.
- **A task is "missing an estimate" when all of these hold** *(Claude's call)*:
  - it has no estimate;
  - it is not canceled or deleted;
  - its team has estimates turned on;
  - it is not a parent whose sub-tasks are estimated.

  Without a scope, `/missing-estimates` covers tagged projects only, with a toggle for all projects. Every team in tagged projects currently has estimates turned on.

### 7.3 Status (from Linear's state type)
| Bucket | Linear state types |
|---|---|
| Done | `completed` |
| In progress | `started` |
| Not started | `backlog`, `unstarted`, `triage` |
| Canceled | `canceled` (excluded from effort) |

Task rows also show the actual Linear state name, for example "In Review".

- **Status is the current status in Linear, not the status at the end of the selected range.** The owner asked for status "pulled directly from Linear". The UI labels it "Current status".
- **"Done vs in progress" always means task effort split by task status.** Project rows also carry a chip with the Linear *project* state (Started, Completed and so on). *(Claude's call.)*

### 7.4 Initiatives
- **Only initiatives whose name starts with `[W]` or `[WS]` are shown** (owner's decision). Matching ignores case and spaces, so it also catches `[W][DRAFT] …` and `[W]  EV …`.
  - Every other initiative is hidden and doesn't count: sprints (`NN.Cloud Sprint …`), buckets (`Cloud Kanban`, `2026 Roadmap`, `Q2 2026`) and team initiatives (`[OS] …`, `[PAY] …`).
  - A project whose only initiatives are hidden ones goes under "No initiative".
- Initiatives aren't nested in this workspace: 0 of 118 have a parent. The initiative level is a flat list.
- An initiative's effort is the full effort of all its projects.
- A project in two initiatives counts fully in both, so initiative totals can overlap. The initiative list shows a note to that effect.
- **On a segment page,** initiatives are listed with only their projects in that segment, at that segment's share of each project's effort. A multi-segment project's row reads, for example, "10 pts · half of 20 · B2B + B2C".
- **Project and initiative pages always show real totals and are never scoped to a segment** (owner's decision). One project has one page and one set of numbers, whichever way the user got there. The page header shows the project's segment chips (for example B2B, B2C). Breadcrumbs are navigation history, carried in the URL as `?via=<segment>`, and never change the numbers.
- Tagged projects with no initiative are grouped under "No initiative".

### 7.5 Date filter
- The range is inclusive, in the Asia/Jakarta time zone, and defaults to 2026-11-01 → 2026-11-30.
- **A task is in range if any of these is true** (owner's decision, 2026-11-17):
  - it was **created** in the range;
  - it was **completed** in the range;
  - it was **in progress** during the range: `startedAt` is on or before the range end, and the task wasn't completed or canceled before the range start.
- Edits alone never count. On the real data, "updated in range" had pulled 200 old, finished tasks into November because of bulk edits.
- A project or initiative appears if any of its tasks is in range, or if it was created in the range.
- All effort numbers on a page count only tasks in range.
- Everything this needs (`createdAt`, `startedAt`, `completedAt`, `canceledAt`) is stored on the task. **No update log is needed**, which makes the sync simpler.

### 7.6 Contributors, priority and labels
- **Contributor** is the task's assignee. Unassigned tasks go to an "Unassigned" bucket.
- **Priority** follows Linear's values: 0 = No priority, 1 = Urgent, 2 = High, 3 = Medium, 4 = Low.
- **Label share** uses the task's own labels (issue labels), all of them.
  - A task with k labels gives effort ÷ k to each label.
  - **Unlabeled effort is kept separate** (owner's decision): it's shown as its own "No label: X pts · Y% of effort" figure, and the label chart covers labeled effort only, adding up to 100%.
  - Grouped labels are shown as `Group / Label`.

## 8. Architecture

```
Linear GraphQL API
        │  daily (and on demand)
        ▼
┌──────────────────────────────┐      ┌──────────────────────┐
│ backend (Go, Kratos)         │ HTTP │ frontend (Next.js,   │
│  - sync job (cron)           │◄─────│  MUI, App Router)    │
│  - aggregation (biz layer)   │ JSON │  server components   │
│  - REST + gRPC (protobuf)    │      │  call backend        │
└──────────────┬───────────────┘      └──────────────────────┘
               │ pgx
               ▼
        CockroachDB (single node in compose)
```

### 8.1 Backend
- Go with Kratos v2, following the `kratos-layout` structure (`api/`, `cmd/`, `internal/{biz,data,service,server,conf}`). The API is defined in protobuf with `google.api.http` annotations, so the frontend gets JSON over HTTP. Implementation follows the `cc-skills-golang` skills (layout, error handling, testing, database).
- **Database access:** `pgx`, with SQL migrations applied at startup. The builder picks the migration tool and decides whether to use sqlc.
- **Aggregation:** the biz layer applies the rules in §7 in Go, as pure functions with table-driven tests. The data volume is small (thousands of tasks).
  - The parent/sub-task decision needs the full task tree, not only in-range rows. Either compute each task's *effective effort* once per sync and store it on the task, or load the full tree for a scope.
- **Linear client:** GraphQL with an API key from env `LINEAR_API_KEY`.
  - Linear limits query complexity to 10,000, so the client uses page sizes of 50 projects or about 250 tasks with narrow selections.
  - It handles rate-limit responses with backoff.

### 8.2 Sync
- **Schedule:** daily at 01:00 Asia/Jakarta, configurable. It can also be triggered with `POST /v1/sync`.
- Only one sync runs at a time. A second trigger returns "already running".
- **Full snapshot, made idempotent:**
  1. Fetch all teams, users, issue labels, project labels, initiatives, projects and issues, including archived ones.
  2. Upsert every row.
  3. Mark rows that were not seen in this run as deleted, so they are excluded.
- A sync that fails partway leaves the previous good data in place. Rerunning it converges to the same state.
- `sync_runs` records start, finish, status, row counts and errors. The UI shows "Data as of <last successful finish>".

### 8.3 Data model (sketch; the builder refines it)
- `teams(id, key, name, estimation_type)`
- `users(id, name, display_name, avatar_url, active)`
- `labels(id, name, group_name, kind[issue|project])`
- `initiatives(id, name, status, url, created_at, updated_at, deleted)`
- `projects(id, name, state, url, created_at, updated_at, deleted)`
- `project_labels(project_id, label_id)`
- `initiative_projects(initiative_id, project_id)`
- `issues(id, identifier, title, url, team_id, project_id, parent_id, assignee_id, estimate, priority, state_name, state_type, created_at, updated_at, started_at, completed_at, canceled_at, deleted)`
- `issue_labels(issue_id, label_id)`
- `sync_runs(id, started_at, finished_at, status, stats jsonb, error)`

### 8.4 API (JSON over HTTP; all list endpoints take `from`, `to`)
- `GET /v1/overview`: per-segment effort, %, project count and status split; the untagged summary; totals; the unestimated count; data-as-of.
- `GET /v1/segments/{segment}`: the segment's summary blocks (contributors, priority, labels, status) plus initiatives, each with its in-segment projects.
- `GET /v1/initiatives/{id}`: summary blocks plus projects (with segment badges).
- `GET /v1/projects/{id}`: summary blocks plus tasks (estimate, effective effort, status, priority, assignee, labels, Linear link, parent/sub-task nesting).
- `GET /v1/projects?untagged=true`: list of untagged projects with effort.
- `GET /v1/missing-estimates?segment=&initiative=&project=`: unestimated in-range tasks, grouped by assignee.
- `POST /v1/sync` and `GET /v1/sync/status`.

**"Summary blocks"** has the same shape on every level: `effort_total`, `effort_by_status`, `by_contributor[]`, `by_priority[]`, `by_label[]` (pts and %), and `tasks_total` / `tasks_unestimated`.

### 8.5 Frontend
> **Superseded on 2026-11-17 after the mockup (owner's decision).** The frontend uses **shadcn/ui** (Base UI and Tailwind v4) with **Linear styling**: the white (light) Linear theme by default (owner's choice), Inter font, compact 13px text, Linear's priority and status icons, and `#5e6ad2` as the accent. It is **one page with a filterable left menu**: All tagged work, then segments → [W]/[WS] initiatives → projects, then Missing estimates and Untagged projects. Clicking an item changes the main view instead of opening a new page. The selection, date range and pivot dimension live in the URL (`?scope=segment:B2B&from=&to=&dim=`).
>
> The **Contributors pivot** follows Linear Insights: assignee rows, with columns chosen from Priority, Status or Segment, plus total effort and unestimated count. Clicking any number opens a side sheet with the tasks behind it.
>
> The reference implementation is `mockup/`. The text below describes the earlier multi-page MUI version and is kept only for history.
- Next.js (App Router, TypeScript), MUI (Material Design 3 look), and MUI X Charts. It calls the backend from server components. The backend URL comes from env, and the browser never talks to Linear.
- **Routes:**
  - `/` is the segment overview.
  - `/segments/[segment]`, `/initiatives/[id]`, `/projects/[id]` and `/projects/untagged` are the drill-down pages.
  - `/missing-estimates` is the reminders page.
- **Every page has:**
  - a breadcrumb;
  - a date range picker kept in the URL (`?from=&to=`), so links can be shared and the range carries through drill-down;
  - "Data as of …" with a Refresh button that calls sync;
  - an "effort (pts)" label on every number;
  - the unestimated indicator linking to `/missing-estimates`, scoped.
- **Layout:** summary cards on top, then charts (label share, priority, contributors, status), then the table of child items. Rows link down, and tasks link out to Linear.
- Multi-segment projects show segment chips.

### 8.6 Room for hours later
"Effort" is one metric (`pts`). Summary blocks and rows carry a metric field so that `hours` can be added later as a second source. That source would probably be a `time_entries` table fed by a tracker integration. Nothing about hours is built now.

### 8.7 Docker Compose
- Services:
  - `cockroach` (single node, `--insecure`, persistent volume, health check);
  - `backend` (waits for the DB, runs migrations, serves HTTP and gRPC, runs the cron);
  - `frontend`.
- `.env` holds `LINEAR_API_KEY`. It is git-ignored, and a `.env.example` is provided.
- `docker compose up` gives a working dashboard after the first sync, which runs at startup if the DB is empty.

## 9. Left to the builder (marked as Claude's calls; owner may override)
- Canceled tasks are excluded from effort.
- The update log is seeded from `createdAt` and the current `updatedAt` on the first sync.
- The migration tool, sqlc or hand-written SQL, and the cron library.
- Exact MUI theme and colors, chart types per block, and table pagination.
- Repository layout: `backend/`, `frontend/`, `docker-compose.yml` and `docs/` at the root.
- Test strategy: table-driven tests for every rule in §7 (segment split, parent/sub-task de-duplication, label split, date-range membership), plus a sync test against recorded Linear responses.
