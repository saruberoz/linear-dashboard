import { DEFAULT_TEAMS } from "@/lib/constants"
import type * as T from "@/lib/types"

/** What the main area shows, with the data the server fetched for it. */
export type View =
  | { kind: "all"; data: T.OverviewResponse }
  | { kind: "segment"; id: string; data: T.SegmentResponse }
  | { kind: "initiative"; id: string; data: T.InitiativeResponse }
  | { kind: "project"; id: string; data: T.ProjectResponse }
  | { kind: "missing"; data: T.MissingResponse }
  | { kind: "labelwarn"; data: T.WarningsResponse }
  | { kind: "untagged"; data: T.ProjectListResponse }
  | { kind: "noinit"; data: T.ProjectListResponse }
  | { kind: "emptyinit"; data: T.EmptyInitiativesResponse }
  | { kind: "noproject"; data: T.AssigneeGroupsResponse }
  | { kind: "unassigned"; data: T.UnassignedTasksResponse }
  | { kind: "notimeline"; data: T.MissingTimelineResponse }
  | { kind: "notasks"; data: T.ProjectListResponse }
  | { kind: "frozen"; data: T.FrozenChangesResponse }
  | { kind: "dashboard"; id: string; data: T.SavedDashboard }
  | { kind: "error"; error: T.ApiError }

export type Range = { from: string; to: string }

/** The team keys a `teams` URL value selects; an empty list means every team. */
export function teamsOf(param: string | undefined): string[] {
  if (!param) return DEFAULT_TEAMS
  return param === "all" ? [] : param.split(",")
}

const jakartaParts = (now: Date) =>
  new Intl.DateTimeFormat("en-CA", {
    timeZone: "Asia/Jakarta",
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
  })
    .format(now)
    .split("-")
    .map(Number)

/** The Monday–Sunday week in Asia/Jakarta containing `now`, shifted by `offset` weeks. */
export function jakartaWeek(now = new Date(), offset = 0): Range {
  const [y, m, d] = jakartaParts(now)
  const sinceMonday = (new Date(Date.UTC(y, m - 1, d)).getUTCDay() + 6) % 7
  const start = Date.UTC(y, m - 1, d - sinceMonday + 7 * offset)
  const iso = (t: number) => new Date(t).toISOString().slice(0, 10)
  return { from: iso(start), to: iso(start + 6 * 86_400_000) }
}

/** The current month in Asia/Jakarta, as YYYY-MM-DD bounds. */
export function currentMonth(now = new Date()): Range {
  const [y, m] = jakartaParts(now)
  const last = new Date(Date.UTC(y, m, 0)).getUTCDate()
  const mm = String(m).padStart(2, "0")
  return {
    from: `${y}-${mm}-01`,
    to: `${y}-${mm}-${String(last).padStart(2, "0")}`,
  }
}

/** The scope string the backend's scoped endpoints take, for a view. */
export function scopeOf(scope: string): string {
  const [kind] = scope.split(":")
  return ["segment", "initiative", "project"].includes(kind) ? scope : "all"
}
