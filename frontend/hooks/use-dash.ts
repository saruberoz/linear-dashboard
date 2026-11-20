"use client"

import { usePathname, useRouter, useSearchParams } from "next/navigation"
import { useTransition } from "react"
import { DEFAULT_PROJECT_STATES } from "@/lib/constants"

/**
 * All dashboard state lives in the URL so any view can be shared as a link.
 *   scope  all | segment:KEY | initiative:ID | project:ID | missing | labelwarn | untagged | noinit | emptyinit | noproject | unassigned | notasks | notimeline
 *   from, to  YYYY-MM-DD (Asia/Jakarta), default: the current month
 *   dim    contributor-table columns: priority | status | segment
 *   via    breadcrumb path: segment:KEY or initiative:ID (never changes numbers)
 *   mscope Missing-estimates filter scope; all=1 adds untagged projects
 *   pstate Untagged-projects state filter, comma-separated; default: all but canceled
 */
export function useDash() {
  const sp = useSearchParams()
  const router = useRouter()
  const path = usePathname()
  const [pending, startTransition] = useTransition()
  const get = (k: string) => sp.get(k) ?? ""
  const href = (patch: Record<string, string>) => {
    const q = new URLSearchParams(sp.toString())
    for (const [k, v] of Object.entries(patch)) {
      if (v) q.set(k, v)
      else q.delete(k)
    }
    const s = q.toString()
    return s ? `${path}?${s}` : path
  }
  const set = (patch: Record<string, string>) =>
    startTransition(() => router.replace(href(patch), { scroll: false }))
  return {
    scope: get("scope") || "all",
    from: get("from"),
    to: get("to"),
    dim: get("dim") || "priority",
    via: get("via"),
    mscope: get("mscope"),
    all: get("all") === "1",
    pstate: get("pstate") ? get("pstate").split(",") : DEFAULT_PROJECT_STATES,
    href,
    set,
    pending,
  }
}
