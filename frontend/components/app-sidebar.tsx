"use client"

import * as React from "react"
import Link from "next/link"
import { useRouter } from "next/navigation"
import {
  ChevronRightIcon,
  LayoutDashboardIcon,
  PlusIcon,
  CircleDashedIcon,
  CalendarXIcon,
  FolderOpenIcon,
  FolderXIcon,
  LayersIcon,
  ListXIcon,
  SearchIcon,
  TagIcon,
  TargetIcon,
  TriangleAlertIcon,
  UserXIcon,
} from "lucide-react"
import {
  Sidebar,
  SidebarContent,
  SidebarGroup,
  SidebarGroupAction,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarInput,
  SidebarMenu,
  SidebarMenuAction,
  SidebarMenuBadge,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarMenuSub,
  SidebarMenuSubButton,
  SidebarMenuSubItem,
} from "@/components/ui/sidebar"
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible"
import {
  PlanStatusIcon,
  SegmentDot,
  planStatusLabel,
} from "@/components/linear-icons"
import { SEG } from "@/lib/constants"
import type { DashboardSummary, MenuResponse } from "@/lib/types"
import { clientFetch } from "@/lib/api"
import { cn } from "@/lib/utils"
import { useDash } from "@/hooks/use-dash"

export function AppSidebar({
  menu,
  dashboards,
}: {
  menu: MenuResponse | null
  dashboards: DashboardSummary[]
}) {
  const { scope, via, href } = useDash()
  const router = useRouter()
  const [creating, setCreating] = React.useState(false)
  const newDashboard = async () => {
    setCreating(true)
    try {
      const r = await clientFetch<{ dashboard: { id: string } }>(
        "dashboards",
        {},
        { method: "POST", body: JSON.stringify({ name: "Untitled dashboard" }) }
      )
      router.push(
        href({
          scope: `dashboard:${r.dashboard.id}`,
          via: "",
          mscope: "",
          all: "",
        })
      )
      router.refresh()
    } finally {
      setCreating(false)
    }
  }
  const [query, setQuery] = React.useState("")
  const [open, setOpen] = React.useState<Record<string, boolean>>(() =>
    initialOpen(scope, via, menu)
  )
  const q = query.trim().toLowerCase()
  const isOpen = (id: string) => (q ? true : !!open[id])
  const toggle = (id: string, v: boolean) => setOpen((o) => ({ ...o, [id]: v }))
  // Menu links keep the date range but drop view-specific state.
  const link = (s: string, extra: Record<string, string> = {}) =>
    href({
      scope: s === "all" ? "" : s,
      via: "",
      mscope: "",
      all: "",
      ...extra,
    })

  const housekeeping = [
    {
      scope: "missing",
      label: "Missing estimates",
      icon: CircleDashedIcon,
      count: menu?.missingCount,
      warn: false,
    },
    {
      scope: "labelwarn",
      label: "Segment label on task",
      icon: TriangleAlertIcon,
      count: menu?.warningCount,
      warn: true,
    },
    {
      scope: "untagged",
      label: "Untagged projects",
      icon: TagIcon,
      count: menu?.untaggedCount,
      warn: false,
    },
    {
      scope: "noinit",
      label: "Projects without initiative",
      icon: FolderXIcon,
      count: menu?.noInitiativeCount,
      warn: false,
    },
    {
      scope: "emptyinit",
      label: "Initiatives without projects",
      icon: TargetIcon,
      count: menu?.emptyInitiativeCount,
      warn: false,
    },
    {
      scope: "noproject",
      label: "Tasks without project",
      icon: ListXIcon,
      count: menu?.noProjectTaskCount,
      warn: false,
    },
    {
      scope: "unassigned",
      label: "Unassigned with effort",
      icon: UserXIcon,
      count: menu?.unassignedTaskCount,
      warn: false,
    },
    {
      scope: "notasks",
      label: "Projects without tasks",
      icon: FolderOpenIcon,
      count: menu?.emptyProjectCount,
      warn: false,
    },
    {
      scope: "notimeline",
      label: "Missing timeline",
      icon: CalendarXIcon,
      count: menu?.missingTimelineCount,
      warn: false,
    },
  ]

  return (
    <Sidebar>
      <SidebarHeader className="gap-3">
        <div className="flex items-center gap-2 px-2 pt-1">
          <div className="flex size-5 items-center justify-center rounded-md bg-primary text-[10px] font-semibold text-primary-foreground">
            P
          </div>
          <span className="font-medium">Linear Dashboard</span>
        </div>
        <div className="relative">
          <SearchIcon className="pointer-events-none absolute top-1/2 left-2 size-3.5 -translate-y-1/2 text-muted-foreground" />
          <SidebarInput
            placeholder="Filter initiatives & projects"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            className="pl-7"
            aria-label="Filter initiatives and projects"
          />
        </div>
      </SidebarHeader>
      <SidebarContent>
        <SidebarGroup>
          <SidebarMenu>
            <SidebarMenuItem>
              <SidebarMenuButton
                isActive={scope === "all"}
                render={<Link href={link("all")} />}
              >
                <LayersIcon />
                <span>All tagged work</span>
              </SidebarMenuButton>
            </SidebarMenuItem>
          </SidebarMenu>
        </SidebarGroup>

        <SidebarGroup>
          <SidebarGroupLabel>Segments</SidebarGroupLabel>
          <SidebarMenu>
            {(menu?.segments ?? []).map((seg) => {
              const inits = seg.initiatives
                .map((i) => ({
                  ...i,
                  projects:
                    q && !i.name.toLowerCase().includes(q)
                      ? i.projects.filter((p) =>
                          p.name.toLowerCase().includes(q)
                        )
                      : i.projects,
                }))
                .filter((i) => i.projects.length)
              if (q && !inits.length) return null
              const id = `segment:${seg.key}`
              return (
                <SidebarMenuItem key={seg.key}>
                  <Collapsible
                    open={isOpen(id)}
                    onOpenChange={(v) => toggle(id, v)}
                  >
                    <SidebarMenuButton
                      isActive={scope === id}
                      render={<Link href={link(id)} />}
                    >
                      <SegmentDot
                        color={SEG[seg.key]?.color ?? "var(--muted-foreground)"}
                      />
                      <span>{seg.label}</span>
                    </SidebarMenuButton>
                    {seg.projectCount > 0 ? (
                      <CollapsibleTrigger
                        render={
                          <SidebarMenuAction
                            aria-label={`Expand ${seg.label}`}
                          />
                        }
                      >
                        <ChevronRightIcon
                          className={cn(
                            "transition-transform",
                            isOpen(id) && "rotate-90"
                          )}
                        />
                      </CollapsibleTrigger>
                    ) : (
                      <SidebarMenuBadge>0</SidebarMenuBadge>
                    )}
                    <CollapsibleContent>
                      <SidebarMenuSub className="mr-0 pr-0">
                        {inits.map((ini) => {
                          const key = `${seg.key}/${ini.id}`
                          const iid = `initiative:${ini.id}`
                          return (
                            <SidebarMenuSubItem key={ini.id || "_none"}>
                              <Collapsible
                                open={isOpen(key)}
                                onOpenChange={(v) => toggle(key, v)}
                              >
                                <div className="flex items-center">
                                  <CollapsibleTrigger
                                    className="flex size-5 shrink-0 items-center justify-center rounded text-muted-foreground hover:text-foreground"
                                    aria-label={`Expand ${ini.name}`}
                                  >
                                    <ChevronRightIcon
                                      className={cn(
                                        "size-3 transition-transform",
                                        isOpen(key) && "rotate-90"
                                      )}
                                    />
                                  </CollapsibleTrigger>
                                  {ini.id === "" ? (
                                    <span className="truncate px-1 text-xs text-muted-foreground">
                                      No initiative
                                    </span>
                                  ) : (
                                    <SidebarMenuSubButton
                                      size="sm"
                                      isActive={scope === iid}
                                      render={
                                        <Link
                                          href={link(iid, { via: id })}
                                          title={
                                            ini.status
                                              ? `${ini.name} · ${planStatusLabel(ini.status)}`
                                              : ini.name
                                          }
                                        />
                                      }
                                      className="flex-1"
                                    >
                                      <PlanStatusIcon
                                        status={ini.status ?? ""}
                                      />
                                      <span>{ini.name}</span>
                                    </SidebarMenuSubButton>
                                  )}
                                </div>
                                <CollapsibleContent>
                                  <SidebarMenuSub className="mr-0 pr-0">
                                    {ini.projects.map((p) => (
                                      <SidebarMenuSubItem key={p.id}>
                                        <SidebarMenuSubButton
                                          size="sm"
                                          isActive={scope === `project:${p.id}`}
                                          render={
                                            <Link
                                              href={link(`project:${p.id}`, {
                                                via: ini.id ? iid : id,
                                              })}
                                              title={p.name}
                                            />
                                          }
                                        >
                                          <span>{p.name}</span>
                                        </SidebarMenuSubButton>
                                      </SidebarMenuSubItem>
                                    ))}
                                  </SidebarMenuSub>
                                </CollapsibleContent>
                              </Collapsible>
                            </SidebarMenuSubItem>
                          )
                        })}
                      </SidebarMenuSub>
                    </CollapsibleContent>
                  </Collapsible>
                </SidebarMenuItem>
              )
            })}
          </SidebarMenu>
        </SidebarGroup>

        <SidebarGroup>
          <SidebarGroupLabel>Dashboards</SidebarGroupLabel>
          <SidebarGroupAction
            title="New dashboard"
            aria-label="New dashboard"
            onClick={newDashboard}
            disabled={creating}
          >
            <PlusIcon />
          </SidebarGroupAction>
          <SidebarMenu>
            {dashboards.map((d) => (
              <SidebarMenuItem key={d.id}>
                <SidebarMenuButton
                  isActive={scope === `dashboard:${d.id}`}
                  render={<Link href={link(`dashboard:${d.id}`)} />}
                >
                  <LayoutDashboardIcon />
                  <span>{d.name}</span>
                </SidebarMenuButton>
                <SidebarMenuBadge>{d.insights}</SidebarMenuBadge>
              </SidebarMenuItem>
            ))}
            {!dashboards.length && (
              <SidebarMenuItem>
                <SidebarMenuButton
                  onClick={newDashboard}
                  disabled={creating}
                  className="text-muted-foreground"
                >
                  <PlusIcon />
                  <span>New dashboard</span>
                </SidebarMenuButton>
              </SidebarMenuItem>
            )}
          </SidebarMenu>
        </SidebarGroup>

        <SidebarGroup>
          <SidebarGroupLabel>Housekeeping</SidebarGroupLabel>
          <SidebarMenu>
            {housekeeping.map((h) => (
              <SidebarMenuItem key={h.scope}>
                <SidebarMenuButton
                  isActive={scope === h.scope}
                  render={<Link href={link(h.scope)} />}
                >
                  <h.icon
                    style={
                      h.warn && h.count ? { color: "var(--warn)" } : undefined
                    }
                  />
                  <span>{h.label}</span>
                </SidebarMenuButton>
                {h.count !== undefined && (
                  <SidebarMenuBadge
                    style={
                      h.warn && h.count ? { color: "var(--warn)" } : undefined
                    }
                  >
                    {h.count}
                  </SidebarMenuBadge>
                )}
              </SidebarMenuItem>
            ))}
          </SidebarMenu>
        </SidebarGroup>
      </SidebarContent>
    </Sidebar>
  )
}

// Expand the path to the current selection on first load.
function initialOpen(scope: string, via: string, menu: MenuResponse | null) {
  const o: Record<string, boolean> = {}
  const [kind, id] = scope.split(":")
  if (kind === "segment") o[scope] = true
  if (!menu || (kind !== "project" && kind !== "initiative")) return o
  for (const seg of menu.segments) {
    for (const ini of seg.initiatives) {
      const hit =
        kind === "initiative"
          ? ini.id === id
          : ini.projects.some((p) => p.id === id)
      const viaHit =
        !via || via === `segment:${seg.key}` || via === `initiative:${ini.id}`
      if (hit && viaHit) {
        o[`segment:${seg.key}`] = true
        if (kind === "project") o[`${seg.key}/${ini.id}`] = true
      }
    }
  }
  return o
}
