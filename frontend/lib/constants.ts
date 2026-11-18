// Display-only constants: labels and colors. No counting rules live in the frontend.

export const SEGMENTS = [
  { key: "B2B", label: "B2B", color: "var(--seg-b2b)" },
  { key: "B2G", label: "B2G", color: "var(--seg-b2g)" },
  { key: "B2BNG", label: "B2B NG", color: "var(--seg-b2bng)" },
  { key: "B2C", label: "B2C", color: "var(--seg-b2c)" },
] as const

export const SEG: Record<string, (typeof SEGMENTS)[number]> =
  Object.fromEntries(SEGMENTS.map((s) => [s.key, s]))

export const STATUS = [
  { key: "done", label: "Done", color: "var(--st-done)" },
  { key: "inProgress", label: "In progress", color: "var(--st-progress)" },
  { key: "notStarted", label: "Not started", color: "var(--st-not)" },
] as const

export const PRIORITY_LABEL: Record<number, string> = {
  0: "No priority",
  1: "Urgent",
  2: "High",
  3: "Medium",
  4: "Low",
}

export const DIMENSIONS = [
  { key: "priority", label: "Priority" },
  { key: "status", label: "Status" },
  { key: "segment", label: "Segment" },
] as const

export const fmt = (n: number) =>
  (Math.round(n * 10) / 10).toLocaleString("en-US")
export const pctFmt = (n: number) => `${Math.round(n * 10) / 10}%`
export const pct = (a: number, b: number) =>
  b ? Math.round((a / b) * 1000) / 10 : 0

// Insight dimensions and measures (keys match the backend).
export const INSIGHT_DIMENSIONS = [
  { key: "assignee", label: "Assignee" },
  { key: "priority", label: "Priority" },
  { key: "status", label: "Status" },
  { key: "segment", label: "Segment" },
  { key: "label", label: "Label" },
  { key: "team", label: "Team" },
  { key: "project", label: "Project" },
  { key: "initiative", label: "Initiative" },
  { key: "created_month", label: "Created month" },
  { key: "completed_month", label: "Completed month" },
  { key: "work_type", label: "Work type (bugs vs other)" },
  { key: "estimate", label: "Estimate size" },
] as const

export const DIMENSION_LABEL: Record<string, string> = Object.fromEntries(
  INSIGHT_DIMENSIONS.map((d) => [d.key, d.label])
)

export const MEASURES = [
  {
    key: "effort",
    label: "Effort (pts)",
    time: false,
    hint: "Sum of estimate points",
  },
  { key: "count", label: "Issue count", time: false, hint: "Number of tasks" },
  {
    key: "cycle_time",
    label: "Cycle time",
    time: true,
    hint: "Started → completed, for tasks completed in the range",
  },
  {
    key: "lead_time",
    label: "Lead time",
    time: true,
    hint: "Created → completed, for tasks completed in the range",
  },
  {
    key: "issue_age",
    label: "Issue age",
    time: true,
    hint: "Created → now, for tasks still open",
  },
  {
    key: "triage_time",
    label: "Triage time",
    time: true,
    hint: "Entered triage → triaged, for tasks triaged in the range",
  },
] as const

export const isTimeMeasure = (m: string) =>
  MEASURES.some((x) => x.key === m && x.time)

/** Days as "4.5 d", or hours under a day. */
export const fmtDays = (d: number) =>
  d < 1 ? `${Math.round(d * 24 * 10) / 10} h` : `${Math.round(d * 10) / 10} d`

// Categorical palette, fixed order (validated default palette, light theme).
export const PALETTE = [
  "#2a78d6",
  "#eb6834",
  "#1baf7a",
  "#eda100",
  "#e87ba4",
  "#008300",
  "#6250d6",
  "#e34948",
]
export const OTHER_COLOR = "#b4b8bf"

const FIXED_COLORS: Record<string, Record<string, string>> = {
  segment: {
    B2B: "var(--seg-b2b)",
    B2G: "var(--seg-b2g)",
    B2BNG: "var(--seg-b2bng)",
    B2C: "var(--seg-b2c)",
    _none: OTHER_COLOR,
  },
  status: {
    done: "var(--st-done)",
    in_progress: "var(--st-progress)",
    not_started: "var(--st-not)",
  },
  work_type: { bug: "#e34948", other: "#2a78d6" },
}

/** Color for a segment value: fixed for segments and statuses, palette order otherwise. */
export function colorFor(dim: string, key: string, index: number): string {
  return (
    FIXED_COLORS[dim]?.[key] ??
    (key === "_none" || key === "_other"
      ? OTHER_COLOR
      : PALETTE[index % PALETTE.length])
  )
}

// Linear project states in workflow order. The untagged page hides canceled until picked.
export const PROJECT_STATES = [
  "backlog",
  "planned",
  "started",
  "paused",
  "completed",
  "canceled",
]
export const DEFAULT_PROJECT_STATES = PROJECT_STATES.filter(
  (s) => s !== "canceled"
)
