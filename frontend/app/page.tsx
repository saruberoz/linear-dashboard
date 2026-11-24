import { SidebarInset, SidebarProvider } from "@/components/ui/sidebar"
import { AppSidebar } from "@/components/app-sidebar"
import { Dashboard } from "@/components/dashboard"
import { BackendError, serverGet } from "@/lib/api"
import type * as T from "@/lib/types"
import { currentMonth, teamsOf, type View } from "@/lib/view"

export const dynamic = "force-dynamic"

type SearchParams = Promise<Record<string, string | string[] | undefined>>

const one = (v: string | string[] | undefined) =>
  (Array.isArray(v) ? v[0] : v) ?? ""

async function loadView(
  scope: string,
  filter: { from: string; to: string; teams: string[] },
  sp: Record<string, string>
): Promise<View> {
  const [kind, ...rest] = scope.split(":")
  const id = rest.join(":")
  try {
    switch (kind) {
      case "segment":
        return {
          kind,
          id,
          data: await serverGet<T.SegmentResponse>(
            `segments/${encodeURIComponent(id)}`,
            filter
          ),
        }
      case "initiative":
        return {
          kind,
          id,
          data: await serverGet<T.InitiativeResponse>(
            `initiatives/${encodeURIComponent(id)}`,
            filter
          ),
        }
      case "project":
        return {
          kind,
          id,
          data: await serverGet<T.ProjectResponse>(
            `projects/${encodeURIComponent(id)}`,
            filter
          ),
        }
      case "missing":
        return {
          kind,
          data: await serverGet<T.MissingResponse>("missing-estimates", {
            ...filter,
            scope: sp.mscope,
            all: sp.all === "1",
          }),
        }
      case "labelwarn":
        return {
          kind,
          data: await serverGet<T.WarningsResponse>(
            "warnings/segment-labels",
            filter
          ),
        }
      case "untagged":
        return {
          kind,
          data: await serverGet<T.ProjectListResponse>(
            "untagged-projects",
            filter
          ),
        }
      case "noinit":
        return {
          kind,
          data: await serverGet<T.ProjectListResponse>(
            "projects-without-initiative",
            filter
          ),
        }
      case "dashboard": {
        const r = await serverGet<{ dashboard: T.SavedDashboard }>(
          `dashboards/${encodeURIComponent(id)}`
        )
        return { kind, id, data: r.dashboard }
      }
      case "emptyinit":
        return {
          kind,
          data: await serverGet<T.EmptyInitiativesResponse>(
            "initiatives-without-projects"
          ),
        }
      case "noproject":
        return {
          kind,
          data: await serverGet<T.AssigneeGroupsResponse>(
            "tasks-without-project",
            filter
          ),
        }
      case "unassigned":
        return {
          kind,
          data: await serverGet<T.UnassignedTasksResponse>(
            "unassigned-tasks",
            filter
          ),
        }
      case "notimeline":
        return {
          kind,
          data: await serverGet<T.MissingTimelineResponse>(
            "missing-timeline",
            filter
          ),
        }
      case "frozen":
        return {
          kind,
          data: await serverGet<T.FrozenChangesResponse>("frozen-changes"),
        }
      case "notasks":
        return {
          kind,
          data: await serverGet<T.ProjectListResponse>(
            "projects-without-tasks",
            filter
          ),
        }
      default:
        return {
          kind: "all",
          data: await serverGet<T.OverviewResponse>("overview", filter),
        }
    }
  } catch (err) {
    if (err instanceof BackendError) return { kind: "error", error: err.body }
    return {
      kind: "error",
      error: {
        code: 502,
        reason: "BACKEND_UNREACHABLE",
        message: "The backend could not be reached.",
      },
    }
  }
}

export default async function Page({
  searchParams,
}: {
  searchParams: SearchParams
}) {
  const raw = await searchParams
  const sp = Object.fromEntries(
    Object.entries(raw).map(([k, v]) => [k, one(v)])
  )
  const month = currentMonth()
  const range = { from: sp.from || month.from, to: sp.to || month.to }
  const filter = { ...range, teams: teamsOf(sp.teams) }
  const [menu, view, dashboards] = await Promise.all([
    serverGet<T.MenuResponse>("menu", filter).catch(() => null),
    loadView(sp.scope || "all", filter, sp),
    serverGet<{ dashboards?: T.DashboardSummary[] }>("dashboards")
      .then((r) => r.dashboards ?? [])
      .catch(() => []),
  ])
  return (
    <SidebarProvider>
      <AppSidebar menu={menu} dashboards={dashboards} />
      <SidebarInset>
        <Dashboard view={view} menu={menu} range={range} />
      </SidebarInset>
    </SidebarProvider>
  )
}
