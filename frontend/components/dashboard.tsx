"use client"

import * as React from "react"
import Link from "next/link"
import { ChevronRightIcon, InfoIcon, TriangleAlertIcon } from "lucide-react"
import {
  Alert,
  AlertAction,
  AlertDescription,
  AlertTitle,
} from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardAction,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyTitle,
} from "@/components/ui/empty"
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Switch } from "@/components/ui/switch"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip"
import { ContributorsPivot } from "@/components/contributors-pivot"
import { DateRange, Header } from "@/components/header"
import { SegmentDot } from "@/components/linear-icons"
import {
  Kpi,
  LabelPanel,
  Panel,
  PriorityPanel,
  ShareBar,
  StatusPanel,
} from "@/components/panels"
import { TaskList } from "@/components/task-list"
import { OpenInLinear } from "@/components/open-in-linear"
import { CustomDashboardView } from "@/components/custom-dashboard"
import { SEG, SEGMENTS, STATUS, fmt, pct, pctFmt } from "@/lib/constants"
import type * as T from "@/lib/types"
import type { Range, View } from "@/lib/view"
import { useDash } from "@/hooks/use-dash"
import { cn } from "@/lib/utils"

function Title({
  children,
  meta,
  action,
}: {
  children: React.ReactNode
  meta?: React.ReactNode
  action?: React.ReactNode
}) {
  return (
    <div className="flex flex-col gap-2">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <h1 className="flex min-w-0 items-center gap-2 text-xl font-medium">
          {children}
        </h1>
        {action}
      </div>
      {meta && (
        <div className="flex flex-wrap items-center gap-2 text-muted-foreground">
          {meta}
        </div>
      )}
    </div>
  )
}

function SegBadges({ segs }: { segs: string[] }) {
  if (!segs.length) return <Badge variant="outline">Untagged</Badge>
  return (
    <span className="flex flex-wrap items-center gap-1">
      {segs.map((k) => (
        <Badge key={k} variant="outline">
          <SegmentDot color={SEG[k]?.color ?? "var(--muted-foreground)"} />
          {SEG[k]?.label ?? k}
        </Badge>
      ))}
      {segs.length > 1 && (
        <Tooltip>
          <TooltipTrigger
            render={<Badge variant="secondary" className="cursor-help" />}
          >
            multi-segment
          </TooltipTrigger>
          <TooltipContent>
            Effort is split evenly between {segs.length} segments on segment
            views
          </TooltipContent>
        </Tooltip>
      )}
    </span>
  )
}

function MiniStatus({ status }: { status: T.StatusSplit }) {
  const t = status.done + status.inProgress + status.notStarted
  if (!t) return <span className="text-muted-foreground/60">—</span>
  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <div className="flex h-1.5 w-20 gap-0.5 overflow-hidden rounded-full bg-muted" />
        }
      >
        {STATUS.map(
          (s) =>
            status[s.key] > 0 && (
              <div
                key={s.key}
                style={{ flex: status[s.key], background: s.color }}
              />
            )
        )}
      </TooltipTrigger>
      <TooltipContent>
        Done {fmt(status.done)} · In progress {fmt(status.inProgress)} · Not
        started {fmt(status.notStarted)} pts
      </TooltipContent>
    </Tooltip>
  )
}

/** Disclosure required by the design: how to read the numbers (decisions D4, D6). */
function HowToRead() {
  return (
    <Alert>
      <InfoIcon />
      <AlertTitle>How to read these numbers</AlertTitle>
      <AlertDescription>
        <ul className="list-disc pl-4">
          <li>
            Effort is Linear estimate points, not hours. Teams estimate on
            different scales, so points are compared, not measured.
          </li>
          <li>
            Segment shares cover tagged projects only. Projects with no B2B /
            B2G / B2B NG / B2C label are left out.
          </li>
          <li>
            A task open across several months counts in each of them, so monthly
            totals don&apos;t add up to a quarter.
          </li>
          <li>
            Past months can change when someone re-estimates, moves or deletes a
            task in Linear.
          </li>
        </ul>
      </AlertDescription>
    </Alert>
  )
}

function SummaryBlocks({
  s,
  missingScope,
  scope,
  range,
}: {
  s: T.Summary
  missingScope: string
  scope: string
  range: Range
}) {
  const { href } = useDash()
  return (
    <>
      <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
        <Kpi
          label="Total effort"
          value={fmt(s.total)}
          unit="pts"
          note="Estimate points, not hours"
        />
        <Kpi
          label="Done"
          value={fmt(s.byStatus.done)}
          unit="pts"
          note={`${pct(s.byStatus.done, s.total)}% of effort`}
        />
        <Kpi
          label="In progress"
          value={fmt(s.byStatus.inProgress)}
          unit="pts"
          note={`${pct(s.byStatus.inProgress, s.total)}% of effort`}
        />
        <Link
          href={href({
            scope: "missing",
            mscope: missingScope,
            via: "",
            all: "",
          })}
          className="rounded-xl focus-visible:outline-2 focus-visible:outline-ring"
        >
          <Kpi
            label="Unestimated tasks"
            value={String(s.tasksMissing)}
            unit={`of ${s.tasksTotal}`}
            note="Effort is understated · view list →"
            tone={s.tasksMissing ? "warn" : undefined}
          />
        </Link>
      </div>
      <div className="grid gap-3 lg:grid-cols-3">
        <StatusPanel s={s} />
        <PriorityPanel s={s} />
        <LabelPanel s={s} />
      </div>
      <ContributorsPivot tables={s.contributors} scope={scope} range={range} />
    </>
  )
}

function ItemsCard({
  title,
  description,
  action,
  children,
}: {
  title: React.ReactNode
  description?: React.ReactNode
  action?: React.ReactNode
  children: React.ReactNode
}) {
  return (
    <Card size="sm" className="gap-0 pb-0">
      <CardHeader className="pb-3">
        <CardTitle>{title}</CardTitle>
        {description && <CardDescription>{description}</CardDescription>}
        {action && <CardAction>{action}</CardAction>}
      </CardHeader>
      <div className="overflow-x-auto">{children}</div>
    </Card>
  )
}

function ProjectTable({
  rows,
  via,
  split,
  extra,
}: {
  rows: T.ProjectRow[]
  via?: string
  split?: boolean
  extra?: "hidden"
}) {
  const { href } = useDash()
  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead className="pl-4">Project</TableHead>
          <TableHead>Segments</TableHead>
          {extra === "hidden" && <TableHead>Only in</TableHead>}
          <TableHead className="text-right">Effort</TableHead>
          <TableHead>Status</TableHead>
          <TableHead className="text-right">Tasks</TableHead>
          <TableHead className="pr-4 text-right">Unestimated</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {rows.map((p) => (
          <TableRow key={p.id}>
            <TableCell className="w-full max-w-0 pl-4">
              <span className="flex items-center gap-2">
                <Link
                  href={href({
                    scope: `project:${p.id}`,
                    via: via ?? "",
                    mscope: "",
                    all: "",
                  })}
                  className="truncate hover:underline"
                >
                  {p.name}
                </Link>
                <OpenInLinear href={p.url} className="ml-auto" />
              </span>
            </TableCell>
            <TableCell>
              <SegBadges segs={p.segments} />
            </TableCell>
            {extra === "hidden" && (
              <TableCell
                className="max-w-56 truncate text-xs text-muted-foreground"
                title={p.hiddenInitiatives.join(", ")}
              >
                {p.hiddenInitiatives.length ? (
                  p.hiddenInitiatives.join(", ")
                ) : (
                  <span className="text-muted-foreground/60">
                    no initiative at all
                  </span>
                )}
              </TableCell>
            )}
            <TableCell className="text-right whitespace-nowrap tabular-nums">
              <span className="font-medium">
                {fmt(split ? p.sharePts : p.pts)}
              </span>{" "}
              pts
              {split && p.splitOf > 1 && (
                <div className="text-[11px] text-muted-foreground">
                  {p.splitOf === 2 ? "half" : `1/${p.splitOf}`} of {fmt(p.pts)}
                </div>
              )}
            </TableCell>
            <TableCell>
              <MiniStatus status={p.status} />
            </TableCell>
            <TableCell className="text-right tabular-nums">{p.tasks}</TableCell>
            <TableCell className="pr-4 text-right tabular-nums">
              {p.missing || <span className="text-muted-foreground/60">0</span>}
            </TableCell>
          </TableRow>
        ))}
        {!rows.length && (
          <TableRow>
            <TableCell
              colSpan={7}
              className="py-6 text-center text-muted-foreground"
            >
              Nothing in this date range
            </TableCell>
          </TableRow>
        )}
      </TableBody>
    </Table>
  )
}

function LabelWarningBanner({ w }: { w: T.OverviewResponse["warnings"] }) {
  const { href } = useDash()
  if (!w.tasks) return null
  return (
    <Alert>
      <TriangleAlertIcon style={{ color: "var(--warn)" }} />
      <AlertTitle>
        {w.tasks} tasks ({fmt(w.pts)} pts) have a segment label on the task, not
        the project
      </AlertTitle>
      <AlertDescription>
        Segments come from project labels only, so {w.uncounted} of these tasks
        are not counted in any segment. Tag their project in Linear to count
        them.
      </AlertDescription>
      <AlertAction>
        <Button
          variant="outline"
          size="sm"
          nativeButton={false}
          render={
            <Link
              href={href({ scope: "labelwarn", via: "", mscope: "", all: "" })}
            />
          }
        >
          Review
        </Button>
      </AlertAction>
    </Alert>
  )
}

function OverviewView({ d, range }: { d: T.OverviewResponse; range: Range }) {
  const { href } = useDash()
  return (
    <>
      <Title meta="Share of estimated effort across the four segments. Multi-segment projects are split evenly.">
        Segments overview
      </Title>
      <LabelWarningBanner w={d.warnings} />
      <Panel
        title="Effort share by segment"
        description={
          <>
            {fmt(d.taggedTotal)} pts of tagged work = 100% ·{" "}
            <Link href={href({ scope: "untagged" })} className="underline">
              {d.untaggedProjects} untagged projects
            </Link>{" "}
            are not included
          </>
        }
      >
        <ShareBar
          height={12}
          total={d.taggedTotal}
          parts={d.segments.map((s) => ({
            label: s.label,
            value: s.pts,
            color: SEG[s.key]?.color ?? "",
          }))}
        />
        <Table className="mt-3">
          <TableHeader>
            <TableRow>
              <TableHead>Segment</TableHead>
              <TableHead className="text-right">Share</TableHead>
              <TableHead className="text-right">Effort</TableHead>
              <TableHead>Status</TableHead>
              <TableHead className="text-right">Projects</TableHead>
              <TableHead className="text-right">
                Shared with another segment
              </TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {d.segments.map((s) => (
              <TableRow key={s.key}>
                <TableCell>
                  <Link
                    href={href({ scope: `segment:${s.key}`, via: "" })}
                    className="flex items-center gap-2 font-medium hover:underline"
                  >
                    <SegmentDot color={SEG[s.key]?.color ?? ""} />
                    {s.label}
                  </Link>
                </TableCell>
                <TableCell className="text-right text-base font-medium tabular-nums">
                  {pctFmt(s.pct)}
                </TableCell>
                <TableCell className="text-right tabular-nums">
                  {fmt(s.pts)} pts
                </TableCell>
                <TableCell>
                  <MiniStatus status={s.status} />
                </TableCell>
                <TableCell className="text-right tabular-nums">
                  {s.projects}
                </TableCell>
                <TableCell className="text-right text-muted-foreground tabular-nums">
                  {s.shared}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </Panel>
      <SummaryBlocks s={d.summary} missingScope="" scope="all" range={range} />
      <HowToRead />
    </>
  )
}

function SegmentView({
  id,
  d,
  range,
}: {
  id: string
  d: T.SegmentResponse
  range: Range
}) {
  const { href } = useDash()
  // Initiatives start collapsed so the list reads as an outline; open one to see its projects.
  const [open, setOpen] = React.useState<Set<string>>(() => new Set())
  const keyOf = (g: (typeof d.groups)[number]) => g.id || "_none"
  const allOpen =
    d.groups.length > 0 && d.groups.every((g) => open.has(keyOf(g)))
  const toggle = (k: string) =>
    setOpen((prev) => {
      const next = new Set(prev)
      if (next.has(k)) next.delete(k)
      else next.add(k)
      return next
    })
  return (
    <>
      <Title
        meta={`${d.projectCount} projects in range · shared projects count at their split share here`}
      >
        <SegmentDot color={SEG[id]?.color ?? ""} />
        {d.label}
      </Title>
      <SummaryBlocks
        s={d.summary}
        missingScope={`segment:${id}`}
        scope={`segment:${id}`}
        range={range}
      />
      <ItemsCard
        title="Initiatives and projects"
        description="Only [W] / [WS] initiatives · a project in more than one initiative counts fully in each, so initiative totals can add up to more than the total above"
        action={
          d.groups.length > 0 && (
            <Button
              variant="ghost"
              size="sm"
              onClick={() =>
                setOpen(allOpen ? new Set() : new Set(d.groups.map(keyOf)))
              }
            >
              {allOpen ? "Collapse all" : "Expand all"}
            </Button>
          )
        }
      >
        <Table>
          <TableBody>
            {d.groups.map((g) => {
              const k = keyOf(g)
              const isOpen = open.has(k)
              return (
                <React.Fragment key={k}>
                  <TableRow className="bg-muted/40 hover:bg-muted/40">
                    <TableCell className="pl-2 font-medium">
                      <Button
                        variant="ghost"
                        size="icon-xs"
                        className="mr-1 align-middle"
                        aria-expanded={isOpen}
                        aria-label={`${isOpen ? "Collapse" : "Expand"} ${g.name || "No initiative"}`}
                        onClick={() => toggle(k)}
                      >
                        <ChevronRightIcon
                          className={cn(
                            "transition-transform",
                            isOpen && "rotate-90"
                          )}
                        />
                      </Button>
                      {g.id ? (
                        <Link
                          href={href({
                            scope: `initiative:${g.id}`,
                            via: `segment:${id}`,
                          })}
                          className="hover:underline"
                        >
                          {g.name}
                        </Link>
                      ) : (
                        <span className="text-muted-foreground">
                          No initiative
                        </span>
                      )}
                      <button
                        type="button"
                        onClick={() => toggle(k)}
                        className="ml-2 font-normal text-muted-foreground hover:text-foreground"
                      >
                        {g.projects.length} project
                        {g.projects.length > 1 ? "s" : ""} · {fmt(g.sharePts)}{" "}
                        pts
                      </button>
                    </TableCell>
                  </TableRow>
                  {isOpen && (
                    <TableRow className="hover:bg-transparent">
                      <TableCell className="p-0">
                        <ProjectTable
                          rows={g.projects}
                          via={g.id ? `initiative:${g.id}` : `segment:${id}`}
                          split
                        />
                      </TableCell>
                    </TableRow>
                  )}
                </React.Fragment>
              )
            })}
          </TableBody>
        </Table>
      </ItemsCard>
    </>
  )
}

function InitiativeView({
  id,
  d,
  range,
}: {
  id: string
  d: T.InitiativeResponse
  range: Range
}) {
  return (
    <>
      <Title
        meta={
          <>
            <SegBadges segs={d.segments} />
            {d.projects.length} tagged projects in range · full project totals,
            not split by segment
          </>
        }
        action={<OpenInLinear href={d.initiative.url} variant="primary" />}
      >
        {d.initiative.name}
      </Title>
      <SummaryBlocks
        s={d.summary}
        missingScope={`initiative:${id}`}
        scope={`initiative:${id}`}
        range={range}
      />
      <ItemsCard
        title="Projects"
        description="A project in more than one initiative counts fully in each."
      >
        <ProjectTable rows={d.projects} via={`initiative:${id}`} />
      </ItemsCard>
    </>
  )
}

function ProjectView({
  id,
  d,
  range,
}: {
  id: string
  d: T.ProjectResponse
  range: Range
}) {
  const { href } = useDash()
  const p = d.project
  return (
    <>
      <Title
        meta={
          <>
            <SegBadges segs={p.segments} />
            <Badge variant="secondary" className="capitalize">
              {p.state}
            </Badge>
            {d.initiatives.map((i) => (
              <Link
                key={i.id}
                href={href({ scope: `initiative:${i.id}`, via: "" })}
                className="hover:text-foreground"
              >
                · {i.name}
              </Link>
            ))}
          </>
        }
        action={<OpenInLinear href={p.url} variant="primary" />}
      >
        {p.name}
      </Title>
      <SummaryBlocks
        s={d.summary}
        missingScope={`project:${id}`}
        scope={`project:${id}`}
        range={range}
      />
      <ItemsCard
        title="Tasks in range"
        description={`${d.tasks.length} tasks · sub-tasks nested under their parent · real project total${p.segments.length > 1 ? " (not split)" : ""}`}
      >
        <TaskList tasks={d.tasks} />
      </ItemsCard>
    </>
  )
}

function MissingView({ d }: { d: T.MissingResponse }) {
  const { mscope, all, set } = useDash()
  const [kind] = mscope.split(":")
  const value = !mscope ? "all" : kind === "segment" ? mscope : "current"
  return (
    <>
      <Title
        meta={
          <>
            {d.total} tasks in <b>{d.scopeTitle}</b> have no estimate, so their
            effort counts as 0. Grouped by assignee so you can remind each
            person.
          </>
        }
      >
        Missing estimates
      </Title>
      <div className="flex flex-wrap items-center gap-4 text-sm">
        <Select
          items={[
            { value: "all", label: "All tagged projects" },
            ...SEGMENTS.map((x) => ({
              value: `segment:${x.key}`,
              label: x.label,
            })),
            { value: "current", label: d.scopeTitle },
          ]}
          value={value}
          onValueChange={(v) =>
            v !== "current" && set({ mscope: v === "all" ? "" : String(v) })
          }
        >
          <SelectTrigger size="sm" className="w-56" aria-label="Scope">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectGroup>
              <SelectItem value="all">All tagged projects</SelectItem>
              {SEGMENTS.map((s) => (
                <SelectItem key={s.key} value={`segment:${s.key}`}>
                  {s.label}
                </SelectItem>
              ))}
              {value === "current" && (
                <SelectItem value="current">{d.scopeTitle}</SelectItem>
              )}
            </SelectGroup>
          </SelectContent>
        </Select>
        {!mscope && (
          <label className="flex items-center gap-2 text-muted-foreground">
            <Switch
              checked={all}
              onCheckedChange={(v) => set({ all: v ? "1" : "" })}
            />
            All projects (include untagged and tasks without a project)
          </label>
        )}
      </div>
      <p className="-mt-2 text-xs text-muted-foreground">
        Excluded: canceled tasks, teams with estimates turned off, and parents
        whose sub-tasks are estimated.
      </p>
      {d.groups.map((g) => (
        <ItemsCard
          key={g.assigneeId}
          title={g.assignee}
          description={`${g.tasks.length} task${g.tasks.length > 1 ? "s" : ""}`}
        >
          <TaskList tasks={g.tasks} showProject />
        </ItemsCard>
      ))}
      {!d.groups.length && (
        <Empty className="border">
          <EmptyHeader>
            <EmptyTitle>Nothing to chase</EmptyTitle>
            <EmptyDescription>
              Every task in this range has an estimate.
            </EmptyDescription>
          </EmptyHeader>
        </Empty>
      )}
    </>
  )
}

function WarningsView({ d }: { d: T.WarningsResponse }) {
  return (
    <>
      <Title
        meta={`${d.total} tasks in range carry a segment label (B2B, B2G, B2B NG or B2C, any case) on the task itself. Task labels never count; tagging the project is the fix.`}
      >
        <TriangleAlertIcon
          className="size-5"
          style={{ color: "var(--warn)" }}
        />
        Segment label on task
      </Title>
      {d.groups.map((g) => (
        <ItemsCard
          key={g.projectId || "_none"}
          title={g.projectName}
          description={
            <>
              {g.tasks.length} task{g.tasks.length > 1 ? "s" : ""} ·{" "}
              {fmt(g.pts)} pts ·{" "}
              {g.counted
                ? `already counted through the project's own segment (${g.projectSegments.map((k) => SEG[k]?.label ?? k).join(" + ")}); the task label is ignored`
                : g.projectId
                  ? "project has no segment label → not counted"
                  : "not in a project → not counted"}
              {g.projectUrl && (
                <>
                  {" "}
                  <OpenInLinear
                    href={g.projectUrl}
                    label="Tag this project in Linear"
                    className="ml-1 align-middle"
                  />
                </>
              )}
            </>
          }
        >
          <TaskList tasks={g.tasks} />
        </ItemsCard>
      ))}
      {!d.groups.length && (
        <Empty className="border">
          <EmptyHeader>
            <EmptyTitle>No warnings</EmptyTitle>
            <EmptyDescription>
              No task in this range carries a segment label of its own.
            </EmptyDescription>
          </EmptyHeader>
        </Empty>
      )}
    </>
  )
}

function ProjectListView({
  title,
  meta,
  rows,
  extra,
}: {
  title: string
  meta: string
  rows: T.ProjectRow[]
  extra?: "hidden"
}) {
  return (
    <>
      <Title meta={meta}>{title}</Title>
      <ItemsCard title={`${rows.length} projects`}>
        <ProjectTable rows={rows} extra={extra} />
      </ItemsCard>
    </>
  )
}

function EmptyInitiativesView({ d }: { d: T.EmptyInitiativesResponse }) {
  return (
    <>
      <Title meta="Initiatives with no linked project. Not filtered by date: they have no tasks. Link a project in Linear or archive the initiative.">
        Initiatives without projects
      </Title>
      <ItemsCard title={`${d.initiatives.length} initiatives`}>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="pl-4">Initiative</TableHead>
              <TableHead>Shown on dashboard</TableHead>
              <TableHead>Status</TableHead>
              <TableHead className="pr-4 text-right">Created</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {d.initiatives.map((i) => (
              <TableRow key={i.id}>
                <TableCell className="w-full max-w-0 pl-4">
                  <span className="flex items-center gap-2">
                    <span className="truncate">{i.name}</span>
                    <OpenInLinear href={i.url} className="ml-auto" />
                  </span>
                </TableCell>
                <TableCell>
                  {i.visible ? (
                    <Badge variant="secondary">[W] / [WS]</Badge>
                  ) : (
                    <span className="text-xs text-muted-foreground">
                      hidden (sprint or bucket)
                    </span>
                  )}
                </TableCell>
                <TableCell className="text-muted-foreground">
                  {i.status}
                </TableCell>
                <TableCell className="pr-4 text-right text-muted-foreground tabular-nums">
                  {i.createdAt}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </ItemsCard>
    </>
  )
}

function ErrorView({ error }: { error: T.ApiError }) {
  const noData = error.reason === "NO_DATA"
  return (
    <Empty className="border">
      <EmptyHeader>
        <EmptyTitle>
          {noData
            ? "The first Linear sync is still running"
            : "Could not load this view"}
        </EmptyTitle>
        <EmptyDescription>
          {noData
            ? "This page refreshes on its own when the data is ready."
            : error.message}
        </EmptyDescription>
      </EmptyHeader>
    </Empty>
  )
}

/** Breadcrumb from the view and the URL's via (checked against the item, so it can't lie). */
function useCrumbs(
  view: View,
  menu: T.MenuResponse | null
): [string, string | null][] {
  const { via, href } = useDash()
  const home = (link: boolean): [string, string | null] => [
    "All tagged work",
    link ? href({ scope: "", via: "", mscope: "", all: "" }) : null,
  ]
  const segCrumb = (k: string): [string, string] => [
    SEG[k]?.label ?? k,
    href({ scope: `segment:${k}`, via: "" }),
  ]
  const iniName = (id: string) =>
    menu?.segments.flatMap((s) => s.initiatives).find((i) => i.id === id)
      ?.name ?? "Initiative"
  const [vkind, vid] = via.split(":")
  switch (view.kind) {
    case "all":
      return [home(false)]
    case "segment":
      return [home(true), [view.data.label, null]]
    case "initiative": {
      const c: [string, string | null][] = [home(true)]
      if (vkind === "segment" && view.data.segments.includes(vid))
        c.push(segCrumb(vid))
      c.push([view.data.initiative.name, null])
      return c
    }
    case "project": {
      const p = view.data.project
      const c: [string, string | null][] = [home(true)]
      const ini =
        vkind === "initiative"
          ? view.data.initiatives.find((i) => i.id === vid)
          : undefined
      if (vkind === "segment" && p.segments.includes(vid)) c.push(segCrumb(vid))
      else if (ini) {
        const seg = menu?.segments.find((s) =>
          s.initiatives.some(
            (i) => i.id === ini.id && i.projects.some((x) => x.id === p.id)
          )
        )
        if (seg) c.push(segCrumb(seg.key))
        c.push([
          iniName(ini.id),
          href({
            scope: `initiative:${ini.id}`,
            via: seg ? `segment:${seg.key}` : "",
          }),
        ])
      } else if (p.segments.length === 1) c.push(segCrumb(p.segments[0]))
      c.push([p.name, null])
      return c
    }
    default: {
      const titles: Record<string, string> = {
        missing: "Missing estimates",
        labelwarn: "Segment label on task",
        untagged: "Untagged projects",
        noinit: "Projects without initiative",
        emptyinit: "Initiatives without projects",
        dashboard:
          view.kind === "dashboard"
            ? `Dashboards · ${view.data.name}`
            : "Dashboard",
        error: "Error",
      }
      return [home(true), [titles[view.kind] ?? "", null]]
    }
  }
}

export function Dashboard({
  view,
  menu,
  range,
}: {
  view: View
  menu: T.MenuResponse | null
  range: Range
}) {
  const crumbs = useCrumbs(view, menu)
  let body: React.ReactNode
  switch (view.kind) {
    case "all":
      body = <OverviewView d={view.data} range={range} />
      break
    case "segment":
      body = (
        <SegmentView key={view.id} id={view.id} d={view.data} range={range} />
      )
      break
    case "initiative":
      body = <InitiativeView id={view.id} d={view.data} range={range} />
      break
    case "project":
      body = <ProjectView id={view.id} d={view.data} range={range} />
      break
    case "missing":
      body = <MissingView d={view.data} />
      break
    case "labelwarn":
      body = <WarningsView d={view.data} />
      break
    case "untagged":
      body = (
        <ProjectListView
          title="Untagged projects"
          meta="Projects active in this range with no B2B / B2G / B2B NG / B2C label. Not counted in segment shares. Largest effort first."
          rows={view.data.projects}
        />
      )
      break
    case "noinit":
      body = (
        <ProjectListView
          title="Projects without initiative"
          meta="Projects active in this range that are in no [W] / [WS] initiative, tagged or not. 'Only in' lists the sprint or bucket initiatives they do belong to."
          rows={view.data.projects}
          extra="hidden"
        />
      )
      break
    case "emptyinit":
      body = <EmptyInitiativesView d={view.data} />
      break
    case "dashboard":
      body = (
        <CustomDashboardView key={view.id} initial={view.data} range={range} />
      )
      break
    default:
      body = <ErrorView error={view.error} />
  }
  return (
    <>
      <Header crumbs={crumbs} sync={menu?.sync ?? null} />
      <div className="mx-auto flex w-full max-w-7xl flex-col gap-4 p-4 md:p-6">
        <div className="flex justify-end">
          <DateRange range={range} />
        </div>
        {body}
      </div>
    </>
  )
}
