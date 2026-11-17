"use client"

import * as React from "react"
import { Bar, BarChart, CartesianGrid, XAxis, YAxis } from "recharts"
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
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { PriorityIcon, StatusIcon } from "@/components/linear-icons"
import { STATUS, fmt, pct, pctFmt } from "@/lib/constants"
import type { Summary } from "@/lib/types"

export function Panel({
  title,
  description,
  action,
  children,
  className,
}: {
  title: React.ReactNode
  description?: React.ReactNode
  action?: React.ReactNode
  children: React.ReactNode
  className?: string
}) {
  return (
    <Card size="sm" className={className}>
      <CardHeader>
        <CardTitle>{title}</CardTitle>
        {description && <CardDescription>{description}</CardDescription>}
        {action && <CardAction>{action}</CardAction>}
      </CardHeader>
      <CardContent>{children}</CardContent>
    </Card>
  )
}

export function Kpi({
  label,
  value,
  unit,
  note,
  tone,
}: {
  label: string
  value: string
  unit?: string
  note: React.ReactNode
  tone?: "warn"
}) {
  return (
    <Card size="sm" className="h-full">
      <CardHeader>
        <CardDescription>{label}</CardDescription>
        <CardTitle
          className="text-2xl font-medium tabular-nums"
          style={tone === "warn" ? { color: "var(--warn)" } : undefined}
        >
          {value}
          {unit && (
            <span className="ml-1 text-xs font-normal text-muted-foreground">
              {unit}
            </span>
          )}
        </CardTitle>
        <CardDescription className="text-xs">{note}</CardDescription>
      </CardHeader>
    </Card>
  )
}

/** Thin part-to-whole bar with a 2px gap between parts, plus a direct-labelled legend. */
export function ShareBar({
  parts,
  total,
  height = 8,
}: {
  parts: { label: string; value: number; color: string }[]
  total: number
  height?: number
}) {
  return (
    <div className="flex flex-col gap-3">
      <div
        className="flex gap-0.5 overflow-hidden rounded-full bg-muted"
        style={{ height }}
      >
        {parts
          .filter((p) => p.value > 0)
          .map((p) => (
            <Tooltip key={p.label}>
              <TooltipTrigger
                render={
                  <div
                    style={{ flex: p.value, background: p.color, minWidth: 2 }}
                  />
                }
              />
              <TooltipContent>
                {p.label}: {fmt(p.value)} pts ({pct(p.value, total)}%)
              </TooltipContent>
            </Tooltip>
          ))}
      </div>
      <div className="flex flex-wrap gap-x-4 gap-y-1 text-xs text-muted-foreground">
        {parts.map((p) => (
          <span key={p.label} className="flex items-center gap-1.5">
            <span
              className="size-2 rounded-full"
              style={{ background: p.color }}
            />
            {p.label}
            <span className="font-medium text-foreground tabular-nums">
              {fmt(p.value)}
            </span>{" "}
            pts · {pct(p.value, total)}%
          </span>
        ))}
      </div>
    </div>
  )
}

const statusIconKey = {
  done: "done",
  inProgress: "in_progress",
  notStarted: "not_started",
} as const

export function StatusPanel({ s }: { s: Summary }) {
  return (
    <Panel
      title="Current status"
      description="As it is in Linear today, not at the end of the range"
    >
      <ShareBar
        total={s.total}
        parts={STATUS.map((x) => ({
          label: x.label,
          value: s.byStatus[x.key],
          color: x.color,
        }))}
      />
      <Table className="mt-3">
        <TableBody>
          {STATUS.map((x) => (
            <TableRow key={x.key}>
              <TableCell className="flex items-center gap-2">
                <StatusIcon status={statusIconKey[x.key]} />
                {x.label}
              </TableCell>
              <TableCell className="text-right tabular-nums">
                {fmt(s.byStatus[x.key])} pts
              </TableCell>
              <TableCell className="w-14 text-right text-muted-foreground tabular-nums">
                {pct(s.byStatus[x.key], s.total)}%
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </Panel>
  )
}

export function PriorityPanel({ s }: { s: Summary }) {
  return (
    <Panel title="By priority">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Priority</TableHead>
            <TableHead className="text-right">Effort</TableHead>
            <TableHead className="text-right">Share</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {s.byPriority.map((p) => (
            <TableRow key={p.key}>
              <TableCell className="flex items-center gap-2">
                <PriorityIcon priority={Number(p.key)} />
                {p.label}
              </TableCell>
              <TableCell className="text-right tabular-nums">
                {fmt(p.pts)} pts
              </TableCell>
              <TableCell className="text-right text-muted-foreground tabular-nums">
                {pctFmt(p.pct)}
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </Panel>
  )
}

const labelChart = {
  pts: { label: "Effort (pts)", color: "var(--chart-1)" },
} satisfies ChartConfig

export function LabelPanel({ s }: { s: Summary }) {
  let rows = s.byLabel.map((b) => [b.label, b.pts] as [string, number])
  if (rows.length > 8) {
    const rest = rows.slice(7)
    rows = [
      ...rows.slice(0, 7),
      [`Other (${rest.length})`, rest.reduce((a, r) => a + r[1], 0)],
    ]
  }
  const labeled = s.total - s.unlabeled
  const data = rows.map(([label, v]) => ({
    label: label.length > 28 ? label.slice(0, 27) + "…" : label,
    full: label,
    pts: Math.round(v * 10) / 10,
  }))
  return (
    <Panel
      title="By label"
      description={
        <>
          Share of labeled effort ({fmt(labeled)} pts) · a task with several
          labels splits its effort evenly
        </>
      }
      action={
        <Tooltip>
          <TooltipTrigger
            render={
              <span className="cursor-help rounded-md border px-2 py-0.5 text-xs text-muted-foreground" />
            }
          >
            No label{" "}
            <span className="font-medium text-foreground tabular-nums">
              {fmt(s.unlabeled)}
            </span>{" "}
            pts · {pct(s.unlabeled, s.total)}%
          </TooltipTrigger>
          <TooltipContent>
            Effort on tasks with no label. Kept out of the chart so the label
            shares cover labeled work only.
          </TooltipContent>
        </Tooltip>
      }
    >
      {data.length ? (
        <ChartContainer
          config={labelChart}
          className="aspect-auto w-full"
          style={{ height: data.length * 28 + 8 }}
        >
          <BarChart
            data={data}
            layout="vertical"
            margin={{ left: 0, right: 8 }}
          >
            <CartesianGrid horizontal={false} />
            <XAxis type="number" hide />
            <YAxis
              dataKey="label"
              type="category"
              width={150}
              tickLine={false}
              axisLine={false}
              tick={{ fontSize: 11 }}
            />
            <ChartTooltip
              cursor={false}
              content={<ChartTooltipContent labelKey="full" />}
            />
            <Bar
              dataKey="pts"
              fill="var(--color-pts)"
              radius={4}
              barSize={12}
            />
          </BarChart>
        </ChartContainer>
      ) : (
        <p className="py-6 text-center text-muted-foreground">
          No labeled effort in this range
        </p>
      )}
    </Panel>
  )
}
