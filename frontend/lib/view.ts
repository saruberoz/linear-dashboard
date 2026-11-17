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
  | { kind: "dashboard"; id: string; data: T.SavedDashboard }
  | { kind: "error"; error: T.ApiError }

export type Range = { from: string; to: string }

/** The current month in Asia/Jakarta, as YYYY-MM-DD bounds. */
export function currentMonth(now = new Date()): Range {
  const ymd = new Intl.DateTimeFormat("en-CA", {
    timeZone: "Asia/Jakarta",
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
  }).format(now)
  const [y, m] = ymd.split("-").map(Number)
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
