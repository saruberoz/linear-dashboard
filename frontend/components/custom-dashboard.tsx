"use client"

import * as React from "react"
import { useRouter } from "next/navigation"
import {
  Bar,
  BarChart,
  CartesianGrid,
  ReferenceLine,
  Scatter,
  ScatterChart,
  XAxis,
  YAxis,
} from "recharts"
import { EllipsisIcon, FilterIcon, PlusIcon, Trash2Icon } from "lucide-react"
import { Alert, AlertDescription } from "@/components/ui/alert"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from "@/components/ui/chart"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyTitle,
} from "@/components/ui/empty"
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet"
import { Skeleton } from "@/components/ui/skeleton"
import { Switch } from "@/components/ui/switch"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group"
import { PriorityIcon, StatusIcon } from "@/components/linear-icons"
import { TaskList } from "@/components/task-list"
import { MultiSelect } from "@/components/multi-select"
import { BackendError, clientFetch } from "@/lib/api"
import {
  DIMENSION_LABEL,
  INSIGHT_DIMENSIONS,
  MEASURES,
  OTHER_COLOR,
  PRIORITY_LABEL,
  SEGMENTS,
  colorFor,
  fmt,
  fmtDays,
  isTimeMeasure,
} from "@/lib/constants"
import type * as T from "@/lib/types"
import type { Range } from "@/lib/view"
import { cn } from "@/lib/utils"

const MAX_GROUPS = 15
const MAX_SEGMENTS = 8
const STATUSES = [
  { key: "done", label: "Done" },
  { key: "in_progress", label: "In progress" },
  { key: "not_started", label: "Not started" },
]

const send = <R,>(path: string, method: string, body?: unknown) =>
  clientFetch<R>(
    path,
    {},
    { method, body: body === undefined ? undefined : JSON.stringify(body) }
  )

const errText = (e: unknown) =>
  e instanceof BackendError ? e.body.message : "Something went wrong"

const fmtValue = (measure: T.Measure, v: number) =>
  isTimeMeasure(measure)
    ? fmtDays(v)
    : measure === "count"
      ? String(Math.round(v))
      : `${fmt(v)} pts`

const measureLabel = (m: string) =>
  MEASURES.find((x) => x.key === m)?.label.replace(" (pts)", "") ?? m

// ---------- filters ----------

function countFilters(f: T.InsightFilters | undefined) {
  if (!f) return 0
  return (
    (f.segments?.length ?? 0) +
    (f.statuses?.length ?? 0) +
    (f.priorities?.length ?? 0) +
    (f.teamIds?.length ?? 0) +
    (f.labels?.length ?? 0) +
    (f.labelMatch?.trim() ? 1 : 0) +
    (f.includeUntagged ? 1 : 0)
  )
}

/** Filter pickers shared by dashboard-level and insight-level filters. */
function FiltersEditor({
  value,
  onChange,
  options,
}: {
  value: T.InsightFilters
  onChange: (f: T.InsightFilters) => void
  options: Options
}) {
  const { teams, labels } = options
  // Stable option objects: the combobox matches selections by reference and resets when they change.
  const teamOptions = React.useMemo(
    () => teams.map((t) => ({ value: t.id, label: t.name, hint: t.key })),
    [teams]
  )
  const labelOptions = React.useMemo(
    () =>
      labels
        .slice(0, 300)
        .map((l) => ({ value: l.name, label: l.name, hint: String(l.tasks) })),
    [labels]
  )
  const set = (patch: Partial<T.InsightFilters>) =>
    onChange({ ...value, ...patch })
  return (
    <FieldGroup>
      <Field>
        <FieldLabel>Segments</FieldLabel>
        <ToggleGroup
          multiple
          size="sm"
          variant="outline"
          value={value.segments ?? []}
          onValueChange={(v) => set({ segments: v as string[] })}
        >
          {SEGMENTS.map((s) => (
            <ToggleGroupItem key={s.key} value={s.key}>
              {s.label}
            </ToggleGroupItem>
          ))}
        </ToggleGroup>
      </Field>
      <Field>
        <FieldLabel>Status</FieldLabel>
        <ToggleGroup
          multiple
          size="sm"
          variant="outline"
          value={value.statuses ?? []}
          onValueChange={(v) => set({ statuses: v as string[] })}
        >
          {STATUSES.map((s) => (
            <ToggleGroupItem key={s.key} value={s.key}>
              <StatusIcon status={s.key} />
              {s.label}
            </ToggleGroupItem>
          ))}
        </ToggleGroup>
      </Field>
      <Field>
        <FieldLabel>Priority</FieldLabel>
        <ToggleGroup
          multiple
          size="sm"
          variant="outline"
          value={(value.priorities ?? []).map(String)}
          onValueChange={(v) =>
            set({ priorities: (v as string[]).map(Number) })
          }
        >
          {[1, 2, 3, 4, 0].map((p) => (
            <ToggleGroupItem
              key={p}
              value={String(p)}
              aria-label={PRIORITY_LABEL[p]}
            >
              <PriorityIcon priority={p} />
              {PRIORITY_LABEL[p]}
            </ToggleGroupItem>
          ))}
        </ToggleGroup>
      </Field>
      <Field>
        <FieldLabel>Teams</FieldLabel>
        <MultiSelect
          ariaLabel="Teams"
          placeholder="All teams — type to pick one or more"
          emptyText="No team matches"
          options={teamOptions}
          value={value.teamIds ?? []}
          onChange={(teamIds) => set({ teamIds })}
        />
      </Field>
      <Field>
        <FieldLabel htmlFor="label-match">Labels</FieldLabel>
        <div className="flex flex-wrap items-center gap-2">
          <Input
            id="label-match"
            className="w-56"
            placeholder="Label contains, e.g. bug"
            value={value.labelMatch ?? ""}
            onChange={(e) => set({ labelMatch: e.target.value })}
          />
          <div className="min-w-0 flex-1">
            <MultiSelect
              ariaLabel="Labels"
              placeholder="Any label — type to pick one or more"
              emptyText="No label matches"
              options={labelOptions}
              value={value.labels ?? []}
              onChange={(labels) => set({ labels })}
            />
          </div>
        </div>
      </Field>
      <Field orientation="horizontal">
        <Switch
          checked={value.includeUntagged ?? false}
          onCheckedChange={(c) => set({ includeUntagged: c })}
          id="untagged"
        />
        <FieldLabel htmlFor="untagged">
          Include untagged projects and tasks without a project
        </FieldLabel>
      </Field>
    </FieldGroup>
  )
}

type Options = { teams: T.TeamOption[]; labels: T.LabelOption[] }

function useOptions(): Options {
  const [options, setOptions] = React.useState<Options>({
    teams: [],
    labels: [],
  })
  React.useEffect(() => {
    clientFetch<{ teams?: T.TeamOption[]; labels?: T.LabelOption[] }>(
      "insights/options"
    )
      .then((r) => setOptions({ teams: r.teams ?? [], labels: r.labels ?? [] }))
      .catch(() => setOptions({ teams: [], labels: [] }))
  }, [])
  return options
}

// ---------- insight editor ----------

const emptySpec: T.InsightSpec = {
  measure: "effort",
  slice: "assignee",
  segment: "",
  view: "bar",
  filters: {},
}

function InsightDialog({
  open,
  onOpenChange,
  initial,
  onSave,
}: {
  open: boolean
  onOpenChange: (o: boolean) => void
  initial?: T.Insight
  onSave: (title: string, spec: T.InsightSpec) => Promise<void>
}) {
  const options = useOptions()
  const [title, setTitle] = React.useState(initial?.title ?? "")
  const [spec, setSpec] = React.useState<T.InsightSpec>(
    initial?.spec ?? emptySpec
  )
  const [saving, setSaving] = React.useState(false)
  const [error, setError] = React.useState("")
  const set = (patch: Partial<T.InsightSpec>) =>
    setSpec((s) => ({ ...s, ...patch }))
  const time = isTimeMeasure(spec.measure)
  const autoTitle = `${measureLabel(spec.measure)} by ${DIMENSION_LABEL[spec.slice]?.toLowerCase()}${
    spec.segment ? ` and ${DIMENSION_LABEL[spec.segment]?.toLowerCase()}` : ""
  }`

  const save = async () => {
    setSaving(true)
    setError("")
    try {
      await onSave(title.trim() || autoTitle, spec)
      onOpenChange(false)
    } catch (e) {
      setError(errText(e))
    } finally {
      setSaving(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-xl">
        <DialogHeader>
          <DialogTitle>{initial ? "Edit insight" : "New insight"}</DialogTitle>
          <DialogDescription>
            Plot a measure sliced by one dimension, optionally split by a second
            one.
          </DialogDescription>
        </DialogHeader>
        <FieldGroup>
          <Field>
            <FieldLabel htmlFor="title">Title</FieldLabel>
            <Input
              id="title"
              placeholder={autoTitle}
              value={title}
              onChange={(e) => setTitle(e.target.value)}
            />
          </Field>
          <Field>
            <FieldLabel>Measure</FieldLabel>
            <Select
              items={MEASURES.map((m) => ({ value: m.key, label: m.label }))}
              value={spec.measure}
              onValueChange={(v) => {
                const measure = String(v) as T.Measure
                set({
                  measure,
                  view:
                    !isTimeMeasure(measure) && spec.view === "scatter"
                      ? "bar"
                      : spec.view,
                })
              }}
            >
              <SelectTrigger aria-label="Measure" className="w-56">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  {MEASURES.map((m) => (
                    <SelectItem key={m.key} value={m.key}>
                      {m.label}
                    </SelectItem>
                  ))}
                </SelectGroup>
              </SelectContent>
            </Select>
            <p className="text-xs text-muted-foreground">
              {MEASURES.find((m) => m.key === spec.measure)?.hint}
              {time && " · shown as the median per group, in days"}
            </p>
          </Field>
          <div className="grid gap-4 sm:grid-cols-2">
            <Field>
              <FieldLabel>Slice by</FieldLabel>
              <Select
                items={INSIGHT_DIMENSIONS.map((d) => ({
                  value: d.key,
                  label: d.label,
                }))}
                value={spec.slice}
                onValueChange={(v) =>
                  set({
                    slice: String(v),
                    segment: spec.segment === v ? "" : spec.segment,
                  })
                }
              >
                <SelectTrigger aria-label="Slice by">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectGroup>
                    {INSIGHT_DIMENSIONS.map((d) => (
                      <SelectItem key={d.key} value={d.key}>
                        {d.label}
                      </SelectItem>
                    ))}
                  </SelectGroup>
                </SelectContent>
              </Select>
            </Field>
            <Field>
              <FieldLabel>Segment by</FieldLabel>
              <Select
                items={[
                  { value: "none", label: "No segment" },
                  ...INSIGHT_DIMENSIONS.map((d) => ({
                    value: d.key,
                    label: d.label,
                  })),
                ]}
                value={spec.segment || "none"}
                onValueChange={(v) =>
                  set({ segment: v === "none" ? "" : String(v) })
                }
              >
                <SelectTrigger aria-label="Segment by">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectGroup>
                    <SelectItem value="none">No segment</SelectItem>
                    {INSIGHT_DIMENSIONS.filter((d) => d.key !== spec.slice).map(
                      (d) => (
                        <SelectItem key={d.key} value={d.key}>
                          {d.label}
                        </SelectItem>
                      )
                    )}
                  </SelectGroup>
                </SelectContent>
              </Select>
            </Field>
          </div>
          <Field>
            <FieldLabel>Show as</FieldLabel>
            <ToggleGroup
              size="sm"
              variant="outline"
              value={[spec.view]}
              onValueChange={(v) =>
                v[0] && set({ view: v[0] as T.InsightView })
              }
            >
              <ToggleGroupItem value="bar">Bar chart</ToggleGroupItem>
              <ToggleGroupItem value="table">Table</ToggleGroupItem>
              <ToggleGroupItem value="metric">Metric</ToggleGroupItem>
              {time && (
                <ToggleGroupItem value="scatter">Scatter</ToggleGroupItem>
              )}
            </ToggleGroup>
          </Field>
          <details className="rounded-lg border p-3">
            <summary className="cursor-pointer text-sm font-medium">
              Insight filters ({countFilters(spec.filters)})
            </summary>
            <div className="pt-3">
              <FiltersEditor
                value={spec.filters ?? {}}
                onChange={(filters) => set({ filters })}
                options={options}
              />
            </div>
          </details>
          {error && (
            <Alert variant="destructive">
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          )}
        </FieldGroup>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button onClick={save} disabled={saving}>
            {initial ? "Save insight" : "Add insight"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

// ---------- one insight ----------

type Drill = { title: string; sliceKey: string; segmentKey: string }

/** Keeps the largest groups and folds the rest into one "Other" group that can't be drilled. */
function topGroups(res: T.InsightResult) {
  const fixedOrder = [
    "created_month",
    "completed_month",
    "priority",
    "status",
    "segment",
    "work_type",
    "estimate",
  ].includes(res.slice)
  if (fixedOrder || res.groups.length <= MAX_GROUPS)
    return { groups: res.groups, other: null as T.InsightGroup | null }
  if (res.timeMeasure)
    return { groups: res.groups.slice(0, MAX_GROUPS), other: null } // medians can't be folded together
  const kept = res.groups.slice(0, MAX_GROUPS - 1)
  const rest = res.groups.slice(MAX_GROUPS - 1)
  const other: T.InsightGroup = {
    key: "_other",
    label: `Other (${rest.length})`,
    value: rest.reduce((a, g) => a + g.value, 0),
    tasks: rest.reduce((a, g) => a + g.tasks, 0),
    segments: [],
  }
  return { groups: kept, other }
}

function segmentLegend(res: T.InsightResult) {
  const values = res.segmentValues.slice(0, MAX_SEGMENTS)
  return values.map((v, i) => ({
    ...v,
    color: colorFor(res.segment, v.key, i),
  }))
}

function InsightChart({
  res,
  onDrill,
}: {
  res: T.InsightResult
  onDrill: (d: Drill) => void
}) {
  const { groups, other } = topGroups(res)
  const legend = res.segment ? segmentLegend(res) : []
  const legendKeys = new Set(legend.map((l) => l.key))
  const rows = [...groups, ...(other ? [other] : [])].map((g) => {
    const shared = g.sharedTasks
      ? ` · ${g.sharedTasks} of ${g.tasks} tasks also under another ${DIMENSION_LABEL[res.slice]?.toLowerCase()}`
      : ""
    const row: Record<string, string | number> = {
      key: g.key,
      label: g.label.length > 26 ? g.label.slice(0, 25) + "…" : g.label,
      full: g.label + shared,
    }
    if (!res.segment) row.value = g.value
    else {
      for (const s of g.segments) {
        if (!legendKeys.has(s.key) && res.timeMeasure) continue
        const k = legendKeys.has(s.key) ? `s_${s.key}` : "s__other"
        row[k] = ((row[k] as number) ?? 0) + s.value
      }
    }
    if (g.key === "_other") row[res.segment ? "s__other" : "value"] = g.value
    return row
  })
  const series = res.segment
    ? [
        ...legend.map((l) => ({
          dataKey: `s_${l.key}`,
          key: l.key,
          label: l.label,
          color: l.color,
        })),
        ...(res.timeMeasure
          ? []
          : [
              {
                dataKey: "s__other",
                key: "_other",
                label: "Other",
                color: OTHER_COLOR,
              },
            ]),
      ]
    : [
        {
          dataKey: "value",
          key: "",
          label: res.timeMeasure
            ? "Median (days)"
            : res.measure === "effort"
              ? "Effort (pts)"
              : "Issues",
          color: "var(--chart-1)",
        },
      ]
  // Medians are shown side by side; sums are stacked.
  const stack = res.timeMeasure ? undefined : "a"
  const config = Object.fromEntries(
    series.map((s) => [s.dataKey, { label: s.label, color: s.color }])
  ) satisfies ChartConfig
  const horizontal = !["created_month", "completed_month"].includes(res.slice)
  const click = (row: Record<string, string | number>, seg: string) => {
    if (row.key === "_other" || seg === "_other") return
    onDrill({
      title: `${row.full}${seg ? ` · ${legend.find((l) => l.key === seg)?.label ?? seg}` : ""}`,
      sliceKey: String(row.key),
      segmentKey: seg,
    })
  }
  if (!rows.length)
    return (
      <p className="py-8 text-center text-muted-foreground">
        No tasks match in this range
      </p>
    )
  return (
    <div className="flex flex-col gap-2">
      <ChartContainer
        config={config}
        className="aspect-auto w-full"
        style={{ height: horizontal ? rows.length * 26 + 30 : 240 }}
      >
        <BarChart
          data={rows}
          layout={horizontal ? "vertical" : "horizontal"}
          margin={{ left: 0, right: 8 }}
        >
          <CartesianGrid horizontal={!horizontal} vertical={horizontal} />
          {horizontal ? (
            <>
              <XAxis type="number" hide />
              <YAxis
                dataKey="label"
                type="category"
                width={140}
                tickLine={false}
                axisLine={false}
                tick={{ fontSize: 11 }}
              />
            </>
          ) : (
            <>
              <XAxis
                dataKey="label"
                tickLine={false}
                axisLine={false}
                tick={{ fontSize: 11 }}
              />
              <YAxis
                tickLine={false}
                axisLine={false}
                width={40}
                tick={{ fontSize: 11 }}
              />
            </>
          )}
          <ChartTooltip
            cursor={false}
            content={<ChartTooltipContent labelKey="full" />}
          />
          {series.map((s, i) => (
            <Bar
              key={s.dataKey}
              dataKey={s.dataKey}
              stackId={stack}
              fill={`var(--color-${s.dataKey})`}
              radius={!stack || i === series.length - 1 ? 4 : 0}
              barSize={horizontal ? 12 : 24}
              className="cursor-pointer"
              onClick={(d: { payload?: Record<string, string | number> }) =>
                d.payload && click(d.payload, s.key)
              }
            />
          ))}
        </BarChart>
      </ChartContainer>
      {res.timeMeasure && res.groups.length > MAX_GROUPS && (
        <p className="text-xs text-muted-foreground">
          Showing the {MAX_GROUPS} groups with the longest median out of{" "}
          {res.groups.length}.
        </p>
      )}
      {res.segment && (
        <div className="flex flex-wrap gap-x-4 gap-y-1 text-xs text-muted-foreground">
          {legend.map((l) => (
            <span key={l.key} className="flex items-center gap-1.5">
              <span
                className="size-2 rounded-full"
                style={{ background: l.color }}
              />
              {l.label}
            </span>
          ))}
          {!res.timeMeasure && res.segmentValues.length > MAX_SEGMENTS && (
            <span className="flex items-center gap-1.5">
              <span
                className="size-2 rounded-full"
                style={{ background: OTHER_COLOR }}
              />
              Other
            </span>
          )}
        </div>
      )}
    </div>
  )
}

function InsightTable({
  res,
  onDrill,
}: {
  res: T.InsightResult
  onDrill: (d: Drill) => void
}) {
  const legend = res.segment ? segmentLegend(res) : []
  const cell = (value: number, tasks: number, d: Drill | null) =>
    tasks && d ? (
      <TableCell className="border-l p-0 text-right tabular-nums">
        <button
          type="button"
          onClick={() => onDrill(d)}
          className="w-full px-3 py-2 text-right hover:bg-accent"
        >
          {fmtValue(res.measure, value)}
        </button>
      </TableCell>
    ) : (
      <TableCell className="border-l text-right text-muted-foreground/50">
        –
      </TableCell>
    )
  if (!res.groups.length)
    return (
      <p className="py-8 text-center text-muted-foreground">
        No tasks match in this range
      </p>
    )
  const percentiles = res.timeMeasure && !res.segment
  return (
    <div className="max-h-96 overflow-auto">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>{DIMENSION_LABEL[res.slice]}</TableHead>
            <TableHead className="border-l text-right">
              {res.timeMeasure ? "Median" : "Total"}
            </TableHead>
            {percentiles && (
              <>
                <TableHead className="border-l text-right">75th pct</TableHead>
                <TableHead className="border-l text-right">95th pct</TableHead>
                <TableHead className="border-l text-right">Tasks</TableHead>
              </>
            )}
            {legend.map((l) => (
              <TableHead key={l.key} className="border-l text-right">
                <span className="inline-flex items-center gap-1.5">
                  <span
                    className="size-2 rounded-full"
                    style={{ background: l.color }}
                  />
                  {l.label}
                </span>
              </TableHead>
            ))}
          </TableRow>
        </TableHeader>
        <TableBody>
          {res.groups.map((g) => (
            <TableRow key={g.key}>
              <TableCell className="max-w-64" title={g.label}>
                <span className="flex items-center gap-2">
                  <span className="truncate">{g.label}</span>
                  {!!g.sharedTasks && (
                    <Badge
                      variant="outline"
                      className="shrink-0"
                      title={`${g.sharedTasks} of ${g.tasks} tasks here also sit under another ${DIMENSION_LABEL[res.slice]?.toLowerCase()}; their effort is split between them`}
                    >
                      {g.sharedTasks} shared
                    </Badge>
                  )}
                </span>
              </TableCell>
              {cell(g.value, g.tasks, {
                title: g.label,
                sliceKey: g.key,
                segmentKey: "",
              })}
              {percentiles && (
                <>
                  <TableCell className="border-l text-right text-muted-foreground tabular-nums">
                    {fmtDays(g.stats?.p75 ?? 0)}
                  </TableCell>
                  <TableCell className="border-l text-right text-muted-foreground tabular-nums">
                    {fmtDays(g.stats?.p95 ?? 0)}
                  </TableCell>
                  <TableCell className="border-l text-right text-muted-foreground tabular-nums">
                    {g.tasks}
                  </TableCell>
                </>
              )}
              {legend.map((l) => {
                const c = g.segments.find((s) => s.key === l.key)
                return (
                  <React.Fragment key={l.key}>
                    {cell(c?.value ?? 0, c?.tasks ?? 0, {
                      title: `${g.label} · ${l.label}`,
                      sliceKey: g.key,
                      segmentKey: l.key,
                    })}
                  </React.Fragment>
                )
              })}
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  )
}

const dayMs = 86_400_000

/** Each task is a point: when it finished (or was created, for age) against how long it took. Click opens it in Linear. */
function InsightScatter({ res }: { res: T.InsightResult }) {
  const legend = res.segment ? segmentLegend(res) : []
  const colorOf = (key: string) =>
    legend.find((l) => l.key === key)?.color ??
    (res.segment ? OTHER_COLOR : "var(--chart-1)")
  const series = res.segment
    ? [
        ...legend.map((l) => ({ key: l.key, label: l.label, color: l.color })),
        { key: "_other", label: "Other", color: OTHER_COLOR },
      ]
    : [{ key: "", label: "Tasks", color: "var(--chart-1)" }]
  const data = res.points.map((p) => ({
    ...p,
    x: Date.parse(`${p.date}T00:00:00`),
    y: Math.round(p.days * 10) / 10,
  }))
  const bySeries = series.map((s) => ({
    ...s,
    points: data.filter((p) =>
      !res.segment
        ? true
        : s.key === "_other"
          ? !legend.some((l) => l.key === p.segmentKey)
          : p.segmentKey === s.key
    ),
  }))
  const config = Object.fromEntries(
    series.map((s) => [s.key || "tasks", { label: s.label, color: s.color }])
  ) satisfies ChartConfig
  if (!data.length)
    return (
      <p className="py-8 text-center text-muted-foreground">
        No tasks match in this range
      </p>
    )
  const xs = data.map((p) => p.x)
  const fmtDate = (v: number) =>
    new Date(v).toLocaleDateString("en-GB", { day: "numeric", month: "short" })
  const date =
    res.measure === "issue_age"
      ? "created"
      : res.measure === "triage_time"
        ? "triaged"
        : "completed"
  return (
    <div className="flex flex-col gap-2">
      <ChartContainer config={config} className="aspect-auto h-72 w-full">
        <ScatterChart margin={{ left: 0, right: 16, top: 8 }}>
          <CartesianGrid />
          <XAxis
            dataKey="x"
            type="number"
            domain={[Math.min(...xs) - dayMs, Math.max(...xs) + dayMs]}
            tickFormatter={fmtDate}
            tickLine={false}
            axisLine={false}
            tick={{ fontSize: 11 }}
            name={`Date ${date}`}
          />
          <YAxis
            dataKey="y"
            type="number"
            tickLine={false}
            axisLine={false}
            width={40}
            tick={{ fontSize: 11 }}
            unit=" d"
            name="Days"
          />
          {res.stats &&
            (
              [
                ["50%", res.stats.p50],
                ["75%", res.stats.p75],
                ["95%", res.stats.p95],
              ] as const
            ).map(([label, y]) => (
              <ReferenceLine
                key={label}
                y={y}
                stroke="var(--muted-foreground)"
                strokeDasharray="4 4"
                label={{
                  value: `${label} ${fmtDays(y)}`,
                  position: "insideTopRight",
                  fontSize: 10,
                  fill: "var(--muted-foreground)",
                }}
              />
            ))}
          <ChartTooltip
            cursor={false}
            content={({ active, payload }) => {
              const p = active && payload?.[0]?.payload
              if (!p) return null
              return (
                <div className="rounded-lg border bg-background px-2.5 py-1.5 text-xs shadow-sm">
                  <div className="font-medium">
                    {p.identifier} · {fmtDays(p.days)}
                  </div>
                  <div className="max-w-64 truncate text-muted-foreground">
                    {p.title}
                  </div>
                  <div className="text-muted-foreground">
                    {p.groupLabel} · {date} {p.date}
                  </div>
                </div>
              )
            }}
          />
          {bySeries.map((s) => (
            <Scatter
              key={s.key || "tasks"}
              data={s.points}
              fill={colorOf(s.key)}
              fillOpacity={0.75}
              className="cursor-pointer"
              onClick={(d: { payload?: { url?: string } }) =>
                d.payload?.url &&
                window.open(d.payload.url, "_blank", "noopener")
              }
            />
          ))}
        </ScatterChart>
      </ChartContainer>
      <div className="flex flex-wrap gap-x-4 gap-y-1 text-xs text-muted-foreground">
        {res.segment &&
          legend.map((l) => (
            <span key={l.key} className="flex items-center gap-1.5">
              <span
                className="size-2 rounded-full"
                style={{ background: l.color }}
              />
              {l.label}
            </span>
          ))}
        <span>
          {data.length} tasks · dashed lines are the 50th, 75th and 95th
          percentiles · click a point to open the task
          {res.pointsOmitted > 0 && ` · ${res.pointsOmitted} more not plotted`}
        </span>
      </div>
    </div>
  )
}

function InsightCard({
  insight,
  dashboard,
  range,
  onEdit,
  onDelete,
}: {
  insight: T.Insight
  dashboard: T.SavedDashboard
  range: Range
  onEdit: () => void
  onDelete: () => void
}) {
  const [res, setRes] = React.useState<T.InsightResult | null>(null)
  const [error, setError] = React.useState("")
  const [drill, setDrill] = React.useState<Drill | null>(null)
  const [tasks, setTasks] = React.useState<{
    tasks: T.TaskRow[]
    total: number
  } | null>(null)
  const body = React.useMemo(
    () => ({
      spec: insight.spec,
      dashboardFilters: dashboard.filters,
      from: range.from,
      to: range.to,
    }),
    [insight.spec, dashboard.filters, range.from, range.to]
  )

  React.useEffect(() => {
    let live = true
    send<{ result: T.InsightResult }>("insights/run", "POST", body)
      .then((r) => live && (setRes(r.result), setError("")))
      .catch((e) => live && setError(errText(e)))
    return () => {
      live = false
    }
  }, [body])

  const openDrill = (d: Drill) => {
    setTasks(null)
    setDrill(d)
    send<{ tasks: T.TaskRow[]; total: number }>("insights/tasks", "POST", {
      ...body,
      sliceKey: d.sliceKey,
      segmentKey: d.segmentKey,
    })
      .then(setTasks)
      .catch((e) => setError(errText(e)))
  }

  const s = insight.spec
  const filterCount = countFilters(s.filters)
  return (
    <Card size="sm" className={cn(s.view === "metric" ? "" : "lg:col-span-2")}>
      <CardHeader>
        <CardTitle>{insight.title}</CardTitle>
        <CardDescription className="flex flex-wrap items-center gap-1">
          {measureLabel(s.measure)} by {DIMENSION_LABEL[s.slice]?.toLowerCase()}
          {s.segment && (
            <> · split by {DIMENSION_LABEL[s.segment]?.toLowerCase()}</>
          )}
          {filterCount > 0 && (
            <Badge variant="secondary">
              <FilterIcon data-icon="inline-start" />
              {filterCount}
            </Badge>
          )}
        </CardDescription>
        <CardAction>
          <DropdownMenu>
            <DropdownMenuTrigger
              render={
                <Button
                  variant="ghost"
                  size="icon-sm"
                  aria-label="Insight options"
                />
              }
            >
              <EllipsisIcon />
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuGroup>
                <DropdownMenuItem onClick={onEdit}>Edit</DropdownMenuItem>
                <DropdownMenuItem variant="destructive" onClick={onDelete}>
                  Delete
                </DropdownMenuItem>
              </DropdownMenuGroup>
            </DropdownMenuContent>
          </DropdownMenu>
        </CardAction>
      </CardHeader>
      <CardContent>
        {error && (
          <Alert variant="destructive">
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        )}
        {!res && !error && <Skeleton className="h-40 w-full" />}
        {res && s.view === "metric" && (
          <button
            type="button"
            onClick={() =>
              openDrill({ title: insight.title, sliceKey: "", segmentKey: "" })
            }
            className="w-full text-left"
          >
            <div className="text-3xl font-medium tabular-nums">
              {fmtValue(res.measure, res.total)}
            </div>
            <div className="text-xs text-muted-foreground">
              {res.timeMeasure && res.stats
                ? `median of ${res.tasks} tasks · 75th pct ${fmtDays(res.stats.p75)} · 95th pct ${fmtDays(res.stats.p95)} · click to see them`
                : `${res.tasks} tasks · click to see them`}
            </div>
          </button>
        )}
        {res && s.view === "bar" && (
          <InsightChart res={res} onDrill={openDrill} />
        )}
        {res && s.view === "table" && (
          <InsightTable res={res} onDrill={openDrill} />
        )}
        {res && s.view === "scatter" && <InsightScatter res={res} />}
        {res?.multiValued && s.view !== "metric" && (
          <p className="mt-2 text-[11px] text-muted-foreground">
            A task can sit in several groups here (segments, labels or
            initiatives), so the groups can add up to more than the total.
          </p>
        )}
      </CardContent>
      <Sheet open={!!drill} onOpenChange={(o) => !o && setDrill(null)}>
        <SheetContent className="w-full data-[side=right]:sm:max-w-5xl">
          <SheetHeader>
            <SheetTitle>{drill?.title}</SheetTitle>
            <SheetDescription>
              {!tasks
                ? "Loading tasks…"
                : isTimeMeasure(s.measure)
                  ? `${tasks.tasks.length} tasks · median ${fmtDays(tasks.total)} · longest first`
                  : `${tasks.tasks.length} tasks · ${fmtValue(s.measure, tasks.total)} counted toward this number`}
            </SheetDescription>
          </SheetHeader>
          <div className="overflow-y-auto px-4 pb-4">
            {!tasks ? (
              <div className="flex flex-col gap-2">
                {Array.from({ length: 6 }, (_, i) => (
                  <Skeleton key={i} className="h-8 w-full" />
                ))}
              </div>
            ) : (
              <TaskList
                tasks={tasks.tasks}
                showProject
                showShare={s.measure === "effort"}
                showDays={isTimeMeasure(s.measure)}
                splitNoun={splitNoun(s)}
              />
            )}
          </div>
        </SheetContent>
      </Sheet>
    </Card>
  )
}

// ---------- dashboard ----------

export function CustomDashboardView({
  initial,
  range,
}: {
  initial: T.SavedDashboard
  range: Range
}) {
  const router = useRouter()
  const options = useOptions()
  const [dash, setDash] = React.useState(initial)
  const [name, setName] = React.useState(initial.name)
  const [editing, setEditing] = React.useState<T.Insight | "new" | null>(null)
  const [showFilters, setShowFilters] = React.useState(false)
  const [error, setError] = React.useState("")

  const update = async (patch: {
    name?: string
    filters?: T.InsightFilters
  }) => {
    try {
      const r = await send<{ dashboard: T.SavedDashboard }>(
        `dashboards/${dash.id}`,
        "PUT",
        {
          name: patch.name ?? dash.name,
          filters: patch.filters ?? dash.filters,
        }
      )
      setDash(r.dashboard)
      if (patch.name) router.refresh() // the sidebar lists dashboard names
    } catch (e) {
      setError(errText(e))
    }
  }

  const saveInsight = async (title: string, spec: T.InsightSpec) => {
    if (editing && editing !== "new") {
      const r = await send<{ insight: T.Insight }>(
        `dashboards/${dash.id}/insights/${editing.id}`,
        "PUT",
        { title, spec, position: editing.position }
      )
      setDash((d) => ({
        ...d,
        insights: d.insights.map((i) =>
          i.id === r.insight.id ? r.insight : i
        ),
      }))
    } else {
      const r = await send<{ insight: T.Insight }>(
        `dashboards/${dash.id}/insights`,
        "POST",
        { title, spec }
      )
      setDash((d) => ({ ...d, insights: [...d.insights, r.insight] }))
      router.refresh()
    }
  }

  const deleteInsight = async (id: string) => {
    try {
      await send(`dashboards/${dash.id}/insights/${id}`, "DELETE")
      setDash((d) => ({
        ...d,
        insights: d.insights.filter((i) => i.id !== id),
      }))
      router.refresh()
    } catch (e) {
      setError(errText(e))
    }
  }

  const deleteDashboard = async () => {
    try {
      await send(`dashboards/${dash.id}`, "DELETE")
      router.push("/")
      router.refresh()
    } catch (e) {
      setError(errText(e))
    }
  }

  const filterCount = countFilters(dash.filters)
  return (
    <>
      <div className="flex flex-wrap items-center gap-2">
        <Input
          aria-label="Dashboard name"
          className="h-9 max-w-md border-transparent text-xl font-medium shadow-none hover:border-input focus-visible:border-input"
          value={name}
          onChange={(e) => setName(e.target.value)}
          onBlur={() =>
            name.trim() &&
            name.trim() !== dash.name &&
            update({ name: name.trim() })
          }
          onKeyDown={(e) =>
            e.key === "Enter" && (e.target as HTMLInputElement).blur()
          }
        />
        <div className="ml-auto flex items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={() => setShowFilters((v) => !v)}
          >
            <FilterIcon data-icon="inline-start" />
            Dashboard filters{filterCount ? ` (${filterCount})` : ""}
          </Button>
          <Button size="sm" onClick={() => setEditing("new")}>
            <PlusIcon data-icon="inline-start" />
            Add insight
          </Button>
          <AlertDialog>
            <AlertDialogTrigger
              render={
                <Button
                  variant="ghost"
                  size="icon-sm"
                  aria-label="Delete dashboard"
                />
              }
            >
              <Trash2Icon />
            </AlertDialogTrigger>
            <AlertDialogContent>
              <AlertDialogHeader>
                <AlertDialogTitle>
                  Delete &ldquo;{dash.name}&rdquo;?
                </AlertDialogTitle>
                <AlertDialogDescription>
                  This removes the dashboard and its {dash.insights.length}{" "}
                  insights for everyone. It can&apos;t be undone.
                </AlertDialogDescription>
              </AlertDialogHeader>
              <AlertDialogFooter>
                <AlertDialogCancel>Cancel</AlertDialogCancel>
                <AlertDialogAction
                  variant="destructive"
                  onClick={deleteDashboard}
                >
                  Delete dashboard
                </AlertDialogAction>
              </AlertDialogFooter>
            </AlertDialogContent>
          </AlertDialog>
        </div>
      </div>
      <p className="-mt-2 text-xs text-muted-foreground">
        Shared dashboard: everyone who can open this app sees and can edit it.
        Uses the date range above and the dashboard&apos;s own rules for effort.
      </p>
      {showFilters && (
        <Card size="sm">
          <CardHeader>
            <CardTitle>Dashboard filters</CardTitle>
            <CardDescription>
              Apply to every insight on this dashboard, together with each
              insight&apos;s own filters.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <FiltersEditor
              value={dash.filters ?? {}}
              onChange={(filters) => update({ filters })}
              options={options}
            />
          </CardContent>
        </Card>
      )}
      {error && (
        <Alert variant="destructive">
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}
      {dash.insights.length ? (
        <div className="grid gap-3 lg:grid-cols-2">
          {dash.insights.map((i) => (
            <InsightCard
              key={i.id}
              insight={i}
              dashboard={dash}
              range={range}
              onEdit={() => setEditing(i)}
              onDelete={() => deleteInsight(i.id)}
            />
          ))}
        </div>
      ) : (
        <Empty className="border">
          <EmptyHeader>
            <EmptyTitle>No insights yet</EmptyTitle>
            <EmptyDescription>
              Add an insight to chart effort or issue count by person, segment,
              label, project and more.
            </EmptyDescription>
          </EmptyHeader>
          <Button size="sm" onClick={() => setEditing("new")}>
            <PlusIcon data-icon="inline-start" />
            Add insight
          </Button>
        </Empty>
      )}
      {editing && (
        <InsightDialog
          key={editing === "new" ? "new" : editing.id}
          open
          onOpenChange={(o) => !o && setEditing(null)}
          initial={editing === "new" ? undefined : editing}
          onSave={saveInsight}
        />
      )}
    </>
  )
}

// splitNoun names what a drill-down task's value is split across, e.g. "labels".
function splitNoun(s: T.InsightSpec): string {
  const names: Record<string, string> = {
    label: "labels",
    segment: "segments",
    initiative: "initiatives",
  }
  const dims = [s.slice, s.segment].filter((d) => d in names)
  return dims.length === 1 ? names[dims[0]] : "groups"
}
