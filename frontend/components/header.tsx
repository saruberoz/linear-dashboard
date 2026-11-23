"use client"

import * as React from "react"
import Link from "next/link"
import { useRouter } from "next/navigation"
import {
  CalendarIcon,
  RefreshCwIcon,
  TriangleAlertIcon,
  UsersIcon,
} from "lucide-react"
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
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
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
import type { SyncInfo, TeamOption } from "@/lib/types"
import { DEFAULT_TEAMS } from "@/lib/constants"
import { currentMonth, jakartaWeek, type Range } from "@/lib/view"
import { useDash } from "@/hooks/use-dash"
import { cn } from "@/lib/utils"

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

const jakartaDay = (d: Date) =>
  d.toLocaleDateString("en-CA", { timeZone: "Asia/Jakarta" })

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
    ["week", "This week", jakartaWeek(now)],
    ["lastweek", "Last week", jakartaWeek(now, -1)],
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
    <div className="flex flex-wrap items-center justify-end gap-2">
      <ToggleGroup
        size="sm"
        variant="outline"
        className="flex-wrap justify-end"
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

const sameKeys = (a: string[], b: string[]) =>
  a.length === b.length && a.every((k) => b.includes(k))

/** Picks the Linear teams every number on the page counts. */
export function TeamFilter({ teams }: { teams: TeamOption[] }) {
  const { teams: selected, set, pending } = useDash()
  const all = selected.length === 0
  const ticked = all ? teams.map((t) => t.key) : selected
  const save = (keys: string[]) =>
    set({
      teams:
        keys.length === teams.length
          ? "all"
          : sameKeys(keys, DEFAULT_TEAMS)
            ? ""
            : keys.join(","),
    })
  const label = all
    ? "All teams"
    : sameKeys(selected, DEFAULT_TEAMS)
      ? `Default teams (${teams.filter((t) => selected.includes(t.key)).length})`
      : selected.length === 1
        ? (teams.find((t) => t.key === selected[0])?.name ?? selected[0])
        : `${selected.length} teams`
  return (
    <DropdownMenu>
      <DropdownMenuTrigger render={<Button variant="outline" size="sm" />}>
        {pending ? (
          <Spinner data-icon="inline-start" />
        ) : (
          <UsersIcon data-icon="inline-start" />
        )}
        {label}
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-64">
        <DropdownMenuItem onClick={() => save(DEFAULT_TEAMS)}>
          Default teams
        </DropdownMenuItem>
        <DropdownMenuItem onClick={() => save(teams.map((t) => t.key))}>
          All teams
        </DropdownMenuItem>
        <DropdownMenuSeparator />
        {teams.map((t) => {
          const on = ticked.includes(t.key)
          return (
            <DropdownMenuCheckboxItem
              key={t.id}
              checked={on}
              disabled={on && ticked.length === 1}
              onCheckedChange={(c) =>
                save(c ? [...ticked, t.key] : ticked.filter((k) => k !== t.key))
              }
            >
              <span className="truncate">{t.name}</span>
              <span className="ml-auto text-xs text-muted-foreground">
                {t.key}
              </span>
            </DropdownMenuCheckboxItem>
          )
        })}
      </DropdownMenuContent>
    </DropdownMenu>
  )
}

/** "Syncing… 42%" with a bar that fills as records arrive; it pulses when the percent is unknown or while saving. */
function SyncProgress({ sync }: { sync: SyncInfo }) {
  const pct = sync.progress ?? -1
  const known = pct >= 0
  const saving = sync.phase === "saving"
  return (
    <span
      className="flex items-center gap-2"
      title={`Started ${wib(sync.lastAttemptAt)}`}
    >
      {saving ? "Saving…" : "Syncing…"}
      <span
        role="progressbar"
        aria-label="Sync progress"
        aria-valuemin={0}
        aria-valuemax={100}
        aria-valuenow={known ? pct : undefined}
        className="relative h-1.5 w-20 overflow-hidden rounded-full bg-white/20"
      >
        <span
          className={cn(
            "absolute inset-y-0 left-0 rounded-full bg-green-400 transition-[width] duration-1000 ease-out motion-reduce:transition-none",
            (!known || saving) && "animate-pulse"
          )}
          style={{ width: `${known ? pct : 100}%` }}
        />
      </span>
      {known && <span className="w-8 text-right tabular-nums">{pct}%</span>}
    </span>
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
    }, 2000)
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
  if (sync?.running) label = <SyncProgress sync={sync} />
  else if (sync?.lastStatus === "failed")
    label = (
      <Tooltip>
        <TooltipTrigger
          render={<span className="flex cursor-help items-center gap-1" />}
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

  // Green when the data is from today (WIB), red when it is older, failed or missing.
  const fresh =
    !!sync?.lastSuccessAt &&
    sync.lastStatus !== "failed" &&
    jakartaDay(new Date(sync.lastSuccessAt)) === jakartaDay(new Date())
  const tone = sync?.running
    ? "text-white"
    : fresh
      ? "text-green-400"
      : "text-red-400"

  return (
    <div className="flex items-center gap-2 text-xs text-muted-foreground">
      <span
        className={cn(
          "hidden h-7 items-center rounded-md bg-black px-2.5 font-medium ring-1 ring-white/10 lg:inline-flex",
          tone
        )}
      >
        {label}
      </span>
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
