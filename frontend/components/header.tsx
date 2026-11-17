"use client"

import * as React from "react"
import Link from "next/link"
import { useRouter } from "next/navigation"
import { CalendarIcon, RefreshCwIcon, TriangleAlertIcon } from "lucide-react"
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "@/components/ui/breadcrumb"
import { Button } from "@/components/ui/button"
import { Calendar } from "@/components/ui/calendar"
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover"
import { Separator } from "@/components/ui/separator"
import { SidebarTrigger } from "@/components/ui/sidebar"
import { Spinner } from "@/components/ui/spinner"
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group"
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip"
import { BackendError, clientFetch } from "@/lib/api"
import type { SyncInfo } from "@/lib/types"
import { currentMonth, type Range } from "@/lib/view"
import { useDash } from "@/hooks/use-dash"

const wib = (iso: string) =>
  iso
    ? new Date(iso).toLocaleString("en-GB", {
        timeZone: "Asia/Jakarta",
        day: "numeric",
        month: "short",
        year: "numeric",
        hour: "2-digit",
        minute: "2-digit",
      }) + " WIB"
    : ""

const ymd = (d: Date) =>
  `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`
const parse = (s: string) => new Date(`${s}T00:00:00`)
const nice = (s: string) =>
  parse(s).toLocaleDateString("en-GB", {
    day: "numeric",
    month: "short",
    year: "numeric",
  })

function presets(): [string, string, Range][] {
  const now = new Date()
  const month = currentMonth(now)
  const last = currentMonth(new Date(now.getFullYear(), now.getMonth() - 1, 15))
  const quarterStart = currentMonth(
    new Date(now.getFullYear(), now.getMonth() - 2, 15)
  ).from
  return [
    ["month", "This month", month],
    ["last", "Last month", last],
    ["quarter", "Last 3 months", { from: quarterStart, to: month.to }],
    ["all", "All time", { from: "2020-01-01", to: month.to }],
  ]
}

export function DateRange({ range }: { range: Range }) {
  const { set, pending } = useDash()
  const list = presets()
  const active = list.find(
    ([, , r]) => r.from === range.from && r.to === range.to
  )?.[0]
  return (
    <div className="flex flex-wrap items-center gap-2">
      <ToggleGroup
        size="sm"
        variant="outline"
        value={active ? [active] : []}
        onValueChange={(v) => {
          const p = list.find(([k]) => k === v[0])
          if (p) set({ from: p[2].from, to: p[2].to })
        }}
      >
        {list.map(([k, label]) => (
          <ToggleGroupItem key={k} value={k}>
            {label}
          </ToggleGroupItem>
        ))}
      </ToggleGroup>
      <Popover>
        <PopoverTrigger render={<Button variant="outline" size="sm" />}>
          {pending ? (
            <Spinner data-icon="inline-start" />
          ) : (
            <CalendarIcon data-icon="inline-start" />
          )}
          {nice(range.from)} – {nice(range.to)}
        </PopoverTrigger>
        <PopoverContent className="w-auto p-0" align="end">
          <Calendar
            mode="range"
            numberOfMonths={2}
            defaultMonth={parse(range.from)}
            selected={{ from: parse(range.from), to: parse(range.to) }}
            onSelect={(r) =>
              r?.from && r?.to && set({ from: ymd(r.from), to: ymd(r.to) })
            }
          />
        </PopoverContent>
      </Popover>
    </div>
  )
}

/** Shows the last sync result and runs a manual sync, polling until it finishes. */
function SyncStatus({ initial }: { initial: SyncInfo | null }) {
  const router = useRouter()
  // A locally fetched status overrides the server's until the page re-renders with a new one.
  const [local, setLocal] = React.useState<{
    from: SyncInfo | null
    value: SyncInfo
  } | null>(null)
  const sync = local && local.from === initial ? local.value : initial
  const setSync = (value: SyncInfo) => setLocal({ from: initial, value })
  const [notice, setNotice] = React.useState("")

  React.useEffect(() => {
    if (!sync?.running) return
    const id = setInterval(async () => {
      try {
        const s = (await clientFetch<{ sync: SyncInfo }>("sync")).sync
        setLocal({ from: initial, value: s })
        if (!s.running) router.refresh()
      } catch {
        // keep polling; the next tick may succeed
      }
    }, 5000)
    return () => clearInterval(id)
  }, [sync?.running, router, initial])

  const refresh = async () => {
    setNotice("")
    try {
      setSync(
        (
          await clientFetch<{ sync: SyncInfo }>(
            "sync",
            {},
            { method: "POST", body: "{}" }
          )
        ).sync
      )
    } catch (e) {
      setNotice(
        e instanceof BackendError ? e.body.message : "Could not start a sync"
      )
    }
  }

  let label: React.ReactNode = "Not synced yet"
  if (sync?.running) label = <>Syncing… started {wib(sync.lastAttemptAt)}</>
  else if (sync?.lastStatus === "failed")
    label = (
      <Tooltip>
        <TooltipTrigger
          render={
            <span
              className="flex cursor-help items-center gap-1"
              style={{ color: "var(--warn)" }}
            />
          }
        >
          <TriangleAlertIcon className="size-3.5" />
          Last sync failed {wib(sync.lastAttemptAt)}
          {sync.lastSuccessAt && <> · data from {wib(sync.lastSuccessAt)}</>}
        </TooltipTrigger>
        <TooltipContent className="max-w-sm">
          {sync.lastError || "Unknown error"}
        </TooltipContent>
      </Tooltip>
    )
  else if (sync?.lastSuccessAt) label = <>Synced {wib(sync.lastSuccessAt)}</>

  return (
    <div className="flex items-center gap-2 text-xs text-muted-foreground">
      <span className="hidden lg:inline">{label}</span>
      {notice && <span style={{ color: "var(--warn)" }}>{notice}</span>}
      <Tooltip>
        <TooltipTrigger
          render={
            <Button
              variant="ghost"
              size="sm"
              onClick={refresh}
              disabled={!!sync?.running}
            />
          }
        >
          {sync?.running ? (
            <Spinner data-icon="inline-start" />
          ) : (
            <RefreshCwIcon data-icon="inline-start" />
          )}
          Refresh
        </TooltipTrigger>
        <TooltipContent>
          Copy the latest data from Linear now (at most once every 15 minutes)
        </TooltipContent>
      </Tooltip>
    </div>
  )
}

export function Header({
  crumbs,
  sync,
}: {
  crumbs: [string, string | null][]
  sync: SyncInfo | null
}) {
  return (
    <header className="sticky top-0 z-10 flex h-12 shrink-0 items-center gap-2 border-b bg-background/95 px-3 backdrop-blur">
      <SidebarTrigger />
      <Separator orientation="vertical" className="mx-1 h-4" />
      <Breadcrumb className="min-w-0">
        <BreadcrumbList className="flex-nowrap">
          {crumbs.map(([label, to], n) => (
            <React.Fragment key={`${n}-${label}`}>
              {n > 0 && <BreadcrumbSeparator />}
              <BreadcrumbItem className="min-w-0">
                {to ? (
                  <BreadcrumbLink render={<Link href={to} />}>
                    {label}
                  </BreadcrumbLink>
                ) : (
                  <BreadcrumbPage className="truncate">{label}</BreadcrumbPage>
                )}
              </BreadcrumbItem>
            </React.Fragment>
          ))}
        </BreadcrumbList>
      </Breadcrumb>
      <div className="ml-auto">
        <SyncStatus initial={sync} />
      </div>
    </header>
  )
}
