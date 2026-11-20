"use client"

import * as React from "react"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Avatar, AvatarFallback } from "@/components/ui/avatar"
import {
  Card,
  CardAction,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet"
import { Skeleton } from "@/components/ui/skeleton"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group"
import { PriorityIcon, SegmentDot, StatusIcon } from "@/components/linear-icons"
import { TaskList } from "@/components/task-list"
import { clientFetch } from "@/lib/api"
import { DIMENSIONS, SEG, fmt } from "@/lib/constants"
import type { ContributorTable, TasksResponse } from "@/lib/types"
import { cn } from "@/lib/utils"
import { useDash } from "@/hooks/use-dash"

type Drill = { title: string; column: string; assigneeId: string; dim: string }

const initials = (name: string) =>
  name
    .split(/[.\s_-]+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((s) => s[0]?.toUpperCase())
    .join("")

function ColumnIcon({ dim, k }: { dim: string; k: string }) {
  if (dim === "priority") return <PriorityIcon priority={Number(k)} />
  if (dim === "status") return <StatusIcon status={k} />
  return <SegmentDot color={SEG[k]?.color ?? "var(--muted-foreground)"} />
}

/** Linear Insights-style pivot: assignees × a chosen dimension. Every number opens the tasks behind it. */
export function ContributorsPivot({
  tables,
  scope,
  range,
}: {
  tables: ContributorTable[]
  scope: string
  range: { from: string; to: string }
}) {
  const { dim: urlDim, set } = useDash()
  const dims = DIMENSIONS.filter((d) => tables.some((t) => t.dim === d.key))
  const dim = tables.some((t) => t.dim === urlDim) ? urlDim : "priority"
  const table = tables.find((t) => t.dim === dim)
  const [drill, setDrill] = React.useState<Drill | null>(null)
  const [result, setResult] = React.useState<TasksResponse | null>(null)
  const [error, setError] = React.useState("")

  const openDrill = (d: Drill) => {
    setResult(null)
    setError("")
    setDrill(d)
  }

  React.useEffect(() => {
    if (!drill) return
    let live = true
    clientFetch<TasksResponse>("tasks", {
      scope,
      from: range.from,
      to: range.to,
      dim: drill.dim,
      column: drill.column,
      assignee_id: drill.assigneeId,
    })
      .then((r) => live && setResult(r))
      .catch((e: Error) => live && setError(e.message))
    return () => {
      live = false
    }
  }, [drill, scope, range.from, range.to])

  if (!table) return null
  const people = table.rows.filter((r) => r.assigneeId !== "_unassigned").length

  const cell = (
    value: number,
    count: number,
    d: Drill,
    opts: { className?: string; showCount?: boolean } = {}
  ) =>
    count ? (
      <TableCell
        className={cn("border-l p-0 text-right tabular-nums", opts.className)}
      >
        <button
          type="button"
          onClick={() => openDrill(d)}
          className="w-full px-3 py-2 text-right hover:bg-accent focus-visible:outline-2 focus-visible:outline-ring"
        >
          {fmt(value)}
          {opts.showCount !== false && (
            <span className="ml-1.5 text-[11px] text-muted-foreground">
              {count}
            </span>
          )}
        </button>
      </TableCell>
    ) : (
      <TableCell
        className={cn(
          "border-l text-right text-muted-foreground/50",
          opts.className
        )}
      >
        –
      </TableCell>
    )

  return (
    <Card size="sm" className="gap-0 pb-0">
      <CardHeader className="pb-3">
        <CardTitle>
          Contributors{" "}
          <span className="font-normal text-muted-foreground tabular-nums">
            {people}
          </span>
        </CardTitle>
        <CardDescription>
          Effort in pts by assignee · the count next to the title leaves out
          Unassigned · small number = task count · click any number to see the
          tasks
        </CardDescription>
        <CardAction>
          <ToggleGroup
            size="sm"
            variant="outline"
            value={[dim]}
            onValueChange={(v) => v[0] && set({ dim: v[0] as string })}
          >
            {dims.map((d) => (
              <ToggleGroupItem key={d.key} value={d.key}>
                {d.label}
              </ToggleGroupItem>
            ))}
          </ToggleGroup>
        </CardAction>
      </CardHeader>
      <div className="overflow-x-auto">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="pl-4">Assignee</TableHead>
              <TableHead className="border-l text-right">
                Total effort
              </TableHead>
              {table.columns.map((c) => (
                <TableHead key={c.key} className="border-l text-right">
                  <span className="inline-flex items-center gap-1.5">
                    <ColumnIcon dim={dim} k={c.key} />
                    {c.label}
                  </span>
                </TableHead>
              ))}
              <TableHead className="border-l pr-4 text-right">
                Unestimated
              </TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {table.rows.map((r) => (
              <TableRow key={r.assigneeId}>
                <TableCell className="pl-4">
                  <span className="flex items-center gap-2">
                    <Avatar className="size-5">
                      <AvatarFallback className="text-[9px]">
                        {r.assigneeId === "_unassigned"
                          ? "–"
                          : initials(r.assignee)}
                      </AvatarFallback>
                    </Avatar>
                    <span
                      className={cn(
                        r.assigneeId === "_unassigned" &&
                          "text-muted-foreground"
                      )}
                    >
                      {r.assignee}
                    </span>
                  </span>
                </TableCell>
                {cell(
                  r.total,
                  r.count,
                  {
                    title: `${r.assignee} · all tasks`,
                    column: "_total",
                    assigneeId: r.assigneeId,
                    dim,
                  },
                  { className: "font-medium" }
                )}
                {table.columns.map((c) => (
                  <React.Fragment key={c.key}>
                    {cell(
                      r.cells[c.key]?.pts ?? 0,
                      r.cells[c.key]?.count ?? 0,
                      {
                        title: `${r.assignee} · ${c.label}`,
                        column: c.key,
                        assigneeId: r.assigneeId,
                        dim,
                      }
                    )}
                  </React.Fragment>
                ))}
                {r.missing ? (
                  cell(
                    r.missing,
                    r.missing,
                    {
                      title: `${r.assignee} · missing estimates`,
                      column: "_missing",
                      assigneeId: r.assigneeId,
                      dim,
                    },
                    {
                      className: "pr-1 [&_button]:text-(--warn)",
                      showCount: false,
                    }
                  )
                ) : (
                  <TableCell className="border-l pr-4 text-right text-muted-foreground/50">
                    0
                  </TableCell>
                )}
              </TableRow>
            ))}
            <TableRow className="bg-muted/40 font-medium hover:bg-muted/40">
              <TableCell className="pl-4">Total</TableCell>
              <TableCell className="border-l px-3 text-right tabular-nums">
                {fmt(table.total)}
              </TableCell>
              {table.columns.map((c) => (
                <TableCell
                  key={c.key}
                  className="border-l px-3 text-right tabular-nums"
                >
                  {fmt(c.pts)}
                </TableCell>
              ))}
              <TableCell className="border-l pr-4 text-right tabular-nums">
                {table.missing}
              </TableCell>
            </TableRow>
          </TableBody>
        </Table>
      </div>

      <Sheet open={!!drill} onOpenChange={(o) => !o && setDrill(null)}>
        <SheetContent className="w-full data-[side=right]:sm:max-w-5xl">
          <SheetHeader>
            <SheetTitle>{drill?.title}</SheetTitle>
            <SheetDescription>
              {result
                ? `${result.tasks.length} tasks · ${fmt(result.total)} pts counted toward this number`
                : "Loading tasks…"}
            </SheetDescription>
          </SheetHeader>
          <div className="overflow-y-auto px-4 pb-4">
            {error && (
              <Alert variant="destructive">
                <AlertDescription>{error}</AlertDescription>
              </Alert>
            )}
            {!result && !error && (
              <div className="flex flex-col gap-2">
                {Array.from({ length: 6 }, (_, i) => (
                  <Skeleton key={i} className="h-8 w-full" />
                ))}
              </div>
            )}
            {result && <TaskList tasks={result.tasks} showProject showShare />}
          </div>
        </SheetContent>
      </Sheet>
    </Card>
  )
}
