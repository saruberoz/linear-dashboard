"use client"

import Link from "next/link"
import { Badge } from "@/components/ui/badge"
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
import { PriorityIcon, StatusIcon } from "@/components/linear-icons"
import { cn } from "@/lib/utils"
import { PRIORITY_LABEL, fmt, fmtDays } from "@/lib/constants"
import type { TaskRow } from "@/lib/types"
import { useDash } from "@/hooks/use-dash"

function Hint({ text, label }: { text: string; label: string }) {
  return (
    <Tooltip>
      <TooltipTrigger
        render={<span className="cursor-help text-xs text-muted-foreground" />}
      >
        {label}
      </TooltipTrigger>
      <TooltipContent>{text}</TooltipContent>
    </Tooltip>
  )
}

function Counted({ t, share }: { t: TaskRow; share: boolean }) {
  if (t.status === "canceled")
    return <Hint text="Canceled tasks are excluded from effort" label="—" />
  if (t.deferred)
    return (
      <Hint
        text="An issue above this sub-task is estimated, so this estimate isn't counted"
        label="in parent"
      />
    )
  if (!t.hasEstimate) return <span className="text-muted-foreground/60">0</span>
  if (share && Math.abs(t.sharePts - t.countedPts) > 1e-9) {
    return (
      <span className="font-medium">
        {fmt(t.sharePts)}
        <span className="ml-1 text-[11px] font-normal text-muted-foreground">
          of {fmt(t.countedPts)}
        </span>
      </span>
    )
  }
  return (
    <span className="font-medium">
      {fmt(share ? t.sharePts : t.countedPts)}
    </span>
  )
}

/**
 * Linear-style task rows: priority · ID · status + title · [project] · assignee · estimate · counted.
 * With showShare, "Counted" shows the task's share of the clicked number ("3 of 6" for a split project).
 */
export function TaskList({
  tasks,
  showProject,
  showShare,
  showDays,
  splitNoun = "groups",
}: {
  tasks: TaskRow[]
  showProject?: boolean
  showShare?: boolean
  showDays?: boolean
  /** What a split is across, e.g. "labels": tasks with splitCount > 1 get a "2 labels" badge. */
  splitNoun?: string
}) {
  const { href } = useDash()
  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead className="w-6 pl-4" />
          <TableHead className="w-20">ID</TableHead>
          <TableHead>Task</TableHead>
          {showProject && <TableHead>Project</TableHead>}
          <TableHead>Assignee</TableHead>
          <TableHead className="text-right">Estimate</TableHead>
          <TableHead className="pr-4 text-right">
            {showDays ? "Duration" : "Counted"}
          </TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {tasks.map((t) => (
          <TableRow
            key={t.id}
            className={cn(t.status === "canceled" && "opacity-50")}
          >
            <TableCell className="pl-4">
              <Tooltip>
                <TooltipTrigger render={<span className="flex" />}>
                  <PriorityIcon priority={t.priority} />
                </TooltipTrigger>
                <TooltipContent>{PRIORITY_LABEL[t.priority]}</TooltipContent>
              </Tooltip>
            </TableCell>
            <TableCell className="font-mono text-xs text-muted-foreground">
              <a
                href={t.url}
                target="_blank"
                rel="noopener"
                className="hover:text-foreground"
              >
                {t.identifier}
              </a>
            </TableCell>
            <TableCell className="w-full max-w-0">
              <span
                className="flex items-center gap-2"
                style={{ paddingLeft: t.depth * 18 }}
              >
                {t.depth > 0 && (
                  <span className="text-muted-foreground/60">↳</span>
                )}
                <Tooltip>
                  <TooltipTrigger render={<span className="flex" />}>
                    <StatusIcon status={t.status} />
                  </TooltipTrigger>
                  <TooltipContent>{t.stateName}</TooltipContent>
                </Tooltip>
                <span
                  className={cn(
                    "truncate",
                    t.status === "canceled" && "line-through"
                  )}
                >
                  {t.title}
                </span>
                {(t.splitCount ?? 0) > 1 && (
                  <Tooltip>
                    <TooltipTrigger
                      render={
                        <Badge
                          variant="outline"
                          className="shrink-0 cursor-help"
                        />
                      }
                    >
                      {t.splitCount} {splitNoun}
                    </TooltipTrigger>
                    <TooltipContent>
                      Counted under {t.splitCount} {splitNoun}
                      {t.labels.length ? ` (${t.labels.join(", ")})` : ""}; its
                      effort is split equally between them.
                    </TooltipContent>
                  </Tooltip>
                )}
              </span>
            </TableCell>
            {showProject && (
              <TableCell className="max-w-48 truncate text-muted-foreground">
                {t.projectId ? (
                  <Link
                    href={href({
                      scope: `project:${t.projectId}`,
                      via: "",
                      mscope: "",
                      all: "",
                    })}
                    title={t.projectName}
                    className="hover:text-foreground hover:underline"
                  >
                    {t.projectName}
                  </Link>
                ) : (
                  <span className="text-muted-foreground/60">No project</span>
                )}
              </TableCell>
            )}
            <TableCell className="whitespace-nowrap text-muted-foreground">
              {t.assignee || "Unassigned"}
            </TableCell>
            <TableCell className="text-right tabular-nums">
              {t.hasEstimate ? (
                fmt(t.estimate)
              ) : t.missing ? (
                <Badge variant="outline" style={{ color: "var(--warn)" }}>
                  missing
                </Badge>
              ) : (
                <span className="text-muted-foreground/60">—</span>
              )}
            </TableCell>
            <TableCell className="pr-4 text-right tabular-nums">
              {showDays && t.hasDays ? (
                <span className="font-medium">{fmtDays(t.days)}</span>
              ) : (
                <Counted t={t} share={!!showShare} />
              )}
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  )
}
