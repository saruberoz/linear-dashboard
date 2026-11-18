// Types mirroring the backend's JSON (protojson: camelCase, zero values included).
// Source of truth: backend/api/dashboard/v1/dashboard.proto.

export type Status = "done" | "in_progress" | "not_started" | "canceled"

export interface SyncInfo {
  running: boolean
  lastSuccessAt: string
  lastAttemptAt: string
  lastStatus: "" | "running" | "succeeded" | "failed"
  lastError: string
  phase?: "fetching" | "saving" // while running
  progress?: number // while running: 0–100, or -1 when the size is unknown
}

export interface StatusSplit {
  done: number
  inProgress: number
  notStarted: number
}

export interface Breakdown {
  key: string
  label: string
  pts: number
  pct: number
}

export interface ContributorCell {
  pts: number
  count: number
}

export interface ContributorRow {
  assigneeId: string
  assignee: string
  total: number
  count: number
  cells: Record<string, ContributorCell>
  missing: number
}

export interface ContributorTable {
  dim: "priority" | "status" | "segment"
  columns: Breakdown[]
  rows: ContributorRow[]
  total: number
  missing: number
}

export interface Summary {
  total: number
  byStatus: StatusSplit
  byPriority: Breakdown[]
  byLabel: Breakdown[]
  unlabeled: number
  tasksTotal: number
  tasksMissing: number
  contributors: ContributorTable[]
}

export interface ProjectRow {
  id: string
  name: string
  url: string
  state: string
  segments: string[]
  pts: number
  sharePts: number
  splitOf: number
  status: StatusSplit
  tasks: number
  missing: number
  hiddenInitiatives: string[]
}

export interface TaskRow {
  id: string
  identifier: string
  title: string
  url: string
  stateName: string
  status: Status
  priority: number
  assignee: string
  labels: string[]
  hasEstimate: boolean
  estimate: number
  countedPts: number
  sharePts: number
  deferred: boolean
  missing: boolean
  projectId: string
  projectName: string
  depth: number
  taskSegments: string[]
  createdAt: string
  hasDays: boolean
  days: number
  splitCount?: number // insight drill-downs: groups this task's value is shared with (e.g. 2 labels)
}

export interface InitiativeRef {
  id: string
  name: string
  url: string
}

export interface MenuSegment {
  key: string
  label: string
  projectCount: number
  initiatives: {
    id: string
    name: string
    projects: { id: string; name: string }[]
  }[]
}

export interface MenuResponse {
  segments: MenuSegment[]
  missingCount: number
  warningCount: number
  untaggedCount: number
  noInitiativeCount: number
  emptyInitiativeCount: number
  sync: SyncInfo
}

export interface SegmentShare {
  key: string
  label: string
  pts: number
  pct: number
  projects: number
  shared: number
  status: StatusSplit
}

export interface OverviewResponse {
  segments: SegmentShare[]
  taggedTotal: number
  summary: Summary
  warnings: { tasks: number; pts: number; uncounted: number }
  untaggedProjects: number
}

export interface InitiativeGroup {
  id: string
  name: string
  url: string
  sharePts: number
  projects: ProjectRow[]
}

export interface SegmentResponse {
  key: string
  label: string
  projectCount: number
  summary: Summary
  groups: InitiativeGroup[]
}

export interface InitiativeResponse {
  initiative: InitiativeRef
  segments: string[]
  summary: Summary
  projects: ProjectRow[]
}

export interface ProjectResponse {
  project: ProjectRow
  initiatives: InitiativeRef[]
  summary: Summary
  tasks: TaskRow[]
}

export interface TasksResponse {
  tasks: TaskRow[]
  total: number
}

export interface MissingResponse {
  scopeTitle: string
  total: number
  groups: { assigneeId: string; assignee: string; tasks: TaskRow[] }[]
}

export interface WarningsResponse {
  total: number
  groups: {
    projectId: string
    projectName: string
    projectUrl: string
    projectSegments: string[]
    counted: boolean
    pts: number
    tasks: TaskRow[]
  }[]
}

export interface ProjectListResponse {
  projects: ProjectRow[]
}

export interface EmptyInitiative {
  id: string
  name: string
  url: string
  status: string
  visible: boolean
  createdAt: string
}

export interface EmptyInitiativesResponse {
  initiatives: EmptyInitiative[]
}

export interface ApiError {
  code: number
  reason: string
  message: string
}

// ---------- custom dashboards (insights.proto) ----------

export interface InsightFilters {
  segments?: string[]
  statuses?: string[]
  priorities?: number[]
  teamIds?: string[]
  includeUntagged?: boolean
  labels?: string[]
  labelMatch?: string
}

export type Measure =
  "effort" | "count" | "cycle_time" | "lead_time" | "issue_age" | "triage_time"
export type InsightView = "bar" | "table" | "metric" | "scatter"

export interface TimeStats {
  n: number
  p25: number
  p50: number
  p75: number
  p95: number
  max: number
}

export interface InsightPoint {
  taskId: string
  identifier: string
  title: string
  url: string
  date: string
  days: number
  groupKey: string
  groupLabel: string
  segmentKey: string
}

export interface LabelOption {
  name: string
  tasks: number
}

export interface InsightSpec {
  measure: Measure
  slice: string
  segment: string
  view: InsightView
  filters?: InsightFilters
}

export interface Insight {
  id: string
  title: string
  spec: InsightSpec
  position: number
}

export interface SavedDashboard {
  id: string
  name: string
  filters: InsightFilters
  insights: Insight[]
  createdAt: string
  updatedAt: string
}

export interface DashboardSummary {
  id: string
  name: string
  insights: number
}

export interface InsightValue {
  key: string
  label: string
  value: number
  tasks: number
  stats?: TimeStats
  sharedTasks?: number // tasks here that also sit in another group of the same dimension
}

export interface InsightGroup extends InsightValue {
  segments: InsightValue[]
}

export interface InsightResult {
  measure: Measure
  slice: string
  segment: string
  groups: InsightGroup[]
  segmentValues: InsightValue[]
  total: number
  tasks: number
  multiValued: boolean
  timeMeasure: boolean
  stats?: TimeStats
  points: InsightPoint[]
  pointsOmitted: number
}

export interface TeamOption {
  id: string
  key: string
  name: string
}
