"use client"

import * as React from "react"
import { useRouter } from "next/navigation"
import { CheckIcon, Undo2Icon } from "lucide-react"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Spinner } from "@/components/ui/spinner"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { Textarea } from "@/components/ui/textarea"
import { OpenInLinear } from "@/components/open-in-linear"
import { BackendError, clientFetch } from "@/lib/api"
import type * as T from "@/lib/types"

const STATUS_LABEL: Record<string, string> = {
  approved: "Approved",
  rejected: "Rejected, undone in Linear",
  withdrawn: "Changed back in Linear",
}

const when = (iso: string) =>
  iso
    ? new Date(iso).toLocaleString("en-GB", {
        timeZone: "Asia/Jakarta",
        day: "numeric",
        month: "short",
        hour: "2-digit",
        minute: "2-digit",
      })
    : ""

function Item({ c }: { c: T.FrozenChange }) {
  return (
    <span className="flex min-w-0 items-center gap-2">
      {c.itemIdentifier && (
        <span className="shrink-0 font-mono text-xs text-muted-foreground">
          {c.itemIdentifier}
        </span>
      )}
      <span className="truncate">{c.itemTitle || "(no longer in Linear)"}</span>
      {c.itemUrl && <OpenInLinear href={c.itemUrl} className="ml-auto" />}
    </span>
  )
}

function Change({ c }: { c: T.FrozenChange }) {
  return (
    <span className="flex flex-col gap-0.5">
      <span className="font-medium">{c.what}</span>
      <span className="text-xs text-muted-foreground">
        {c.oldValue} → <span className="text-foreground">{c.newValue}</span>
      </span>
    </span>
  )
}

type Action = "approve" | "reject"

const ACTION_LABEL: Record<Action, [string, string]> = {
  approve: ["Approving", "Approved"],
  reject: ["Rejecting", "Rejected"],
}

const name = (c: T.FrozenChange) =>
  c.itemIdentifier || c.itemTitle || "a change no longer in Linear"

/**
 * Pending changes to completed work. Each row has Approve and Reject, and
 * selected rows can be decided together.
 */
export function PendingChanges({ rows }: { rows: T.FrozenChange[] }) {
  const router = useRouter()
  const [busy, setBusy] = React.useState("")
  const [notice, setNotice] = React.useState("")
  const [picked, setPicked] = React.useState<Set<string>>(new Set())
  const [rejecting, setRejecting] = React.useState<T.FrozenChange[] | null>(
    null
  )
  const [reason, setReason] = React.useState("")

  const selected = rows.filter((c) => picked.has(c.id))
  const allPicked = rows.length > 0 && selected.length === rows.length

  const toggle = (id: string, on: boolean) =>
    setPicked((prev) => {
      const next = new Set(prev)
      if (on) next.add(id)
      else next.delete(id)
      return next
    })

  // The server decides one change at a time, so a batch goes in order.
  const decide = async (
    changes: T.FrozenChange[],
    action: Action,
    body: object = {}
  ) => {
    const [doing, done] = ACTION_LABEL[action]
    const failed: string[] = []
    setNotice("")
    for (const [i, c] of changes.entries()) {
      setBusy(
        changes.length > 1
          ? `${doing} ${i + 1} of ${changes.length}`
          : `${c.id}:${action}`
      )
      try {
        await clientFetch(
          `frozen-changes/${c.id}/${action}`,
          {},
          { method: "POST", body: JSON.stringify(body) }
        )
        toggle(c.id, false)
      } catch (e) {
        failed.push(
          `${name(c)}: ${
            e instanceof BackendError
              ? e.body.message
              : "could not reach the dashboard"
          }`
        )
      }
    }
    if (failed.length)
      setNotice(
        (changes.length > 1
          ? `${done} ${changes.length - failed.length} of ${changes.length}. `
          : "") + failed.join("; ")
      )
    setRejecting(null)
    setReason("")
    setBusy("")
    router.refresh()
  }

  if (!rows.length)
    return (
      <p className="px-4 pb-4 text-sm text-muted-foreground">
        Nothing waiting. Changes Linear makes to completed work show up here
        after a sync.
      </p>
    )

  const one = rejecting?.length === 1 ? rejecting[0] : null

  return (
    <>
      {notice && (
        <p className="px-4 pb-3 text-sm" style={{ color: "var(--warn)" }}>
          {notice}
        </p>
      )}
      <div className="flex min-h-9 flex-wrap items-center gap-2 px-4 pb-3 text-sm">
        <span className="text-muted-foreground tabular-nums">
          {busy && !busy.includes(":")
            ? `${busy}…`
            : selected.length
              ? `${selected.length} selected`
              : "Select changes to decide them together"}
        </span>
        {selected.length > 0 && (
          <span className="flex flex-wrap gap-1.5">
            <Button
              size="sm"
              disabled={!!busy}
              onClick={() => decide(selected, "approve")}
            >
              <CheckIcon data-icon="inline-start" />
              Approve {selected.length}
            </Button>
            <Button
              size="sm"
              variant="outline"
              disabled={!!busy}
              onClick={() => setRejecting(selected)}
            >
              <Undo2Icon data-icon="inline-start" />
              Reject {selected.length}
            </Button>
            <Button
              size="sm"
              variant="ghost"
              disabled={!!busy}
              onClick={() => setPicked(new Set())}
            >
              Clear
            </Button>
          </span>
        )}
      </div>
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead className="w-8 pl-4">
              <Checkbox
                aria-label="Select all changes"
                checked={allPicked}
                indeterminate={selected.length > 0 && !allPicked}
                disabled={!!busy}
                onCheckedChange={() =>
                  setPicked(
                    allPicked ? new Set() : new Set(rows.map((c) => c.id))
                  )
                }
              />
            </TableHead>
            <TableHead>Task or project</TableHead>
            <TableHead>Change</TableHead>
            <TableHead>Frozen because</TableHead>
            <TableHead>Seen</TableHead>
            <TableHead className="pr-4 text-right">Decision</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.map((c) => (
            <TableRow
              key={c.id}
              data-state={picked.has(c.id) ? "selected" : undefined}
            >
              <TableCell className="pl-4">
                <Checkbox
                  aria-label={`Select ${name(c)}`}
                  checked={picked.has(c.id)}
                  disabled={!!busy}
                  onCheckedChange={(on) => toggle(c.id, on)}
                />
              </TableCell>
              <TableCell className="w-full max-w-0">
                <Item c={c} />
              </TableCell>
              <TableCell className="max-w-64 min-w-40 whitespace-normal">
                <Change c={c} />
              </TableCell>
              <TableCell className="max-w-56 min-w-40 whitespace-normal text-muted-foreground">
                {c.frozenBy}
              </TableCell>
              <TableCell className="whitespace-nowrap text-muted-foreground tabular-nums">
                {when(c.detectedAt)}
              </TableCell>
              <TableCell className="pr-4">
                <span className="flex flex-wrap justify-end gap-1.5">
                  <Button
                    size="sm"
                    variant="outline"
                    disabled={!!busy}
                    onClick={() => decide([c], "approve")}
                  >
                    {busy === `${c.id}:approve` ? (
                      <Spinner data-icon="inline-start" />
                    ) : (
                      <CheckIcon data-icon="inline-start" />
                    )}
                    Approve
                  </Button>
                  <Button
                    size="sm"
                    variant="outline"
                    disabled={!!busy}
                    onClick={() => setRejecting([c])}
                  >
                    <Undo2Icon data-icon="inline-start" />
                    Reject
                  </Button>
                </span>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
      <Dialog
        open={!!rejecting}
        onOpenChange={(open) => !open && !busy && setRejecting(null)}
      >
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>
              {one
                ? "Reject and undo in Linear?"
                : `Reject ${rejecting?.length} changes and undo them in Linear?`}
            </DialogTitle>
            <DialogDescription>
              {one ? (
                <>
                  {one.what} of {name(one)} goes back to {one.oldValue} in
                  Linear, and a comment there says why: {one.frozenBy}, so it is
                  frozen.
                </>
              ) : (
                <>
                  Each change goes back to its old value in Linear, and a
                  comment on each says why it is frozen:{" "}
                  {rejecting?.map(name).join(", ")}.
                </>
              )}
            </DialogDescription>
          </DialogHeader>
          <label className="flex flex-col gap-1.5 text-sm">
            {one
              ? "Reason (optional, added to the comment)"
              : "Reason (optional, added to every comment)"}
            <Textarea
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              placeholder="e.g. Scope was agreed at sprint start; open a new task instead."
              rows={3}
            />
          </label>
          <DialogFooter>
            <Button
              variant="outline"
              disabled={!!busy}
              onClick={() => setRejecting(null)}
            >
              Cancel
            </Button>
            <Button
              disabled={!!busy}
              onClick={() =>
                rejecting && decide(rejecting, "reject", { reason })
              }
            >
              {busy && <Spinner data-icon="inline-start" />}
              {busy && !busy.includes(":") ? `${busy}…` : "Reject and undo"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}

/** Changes already decided, or withdrawn because Linear changed back. */
export function DecidedChanges({ rows }: { rows: T.FrozenChange[] }) {
  if (!rows.length)
    return (
      <p className="px-4 pb-4 text-sm text-muted-foreground">
        No decisions yet.
      </p>
    )
  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead className="pl-4">Task or project</TableHead>
          <TableHead>Change</TableHead>
          <TableHead>Outcome</TableHead>
          <TableHead className="pr-4 text-right">When</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {rows.map((c) => (
          <TableRow key={c.id}>
            <TableCell className="w-full max-w-0 pl-4">
              <Item c={c} />
            </TableCell>
            <TableCell className="max-w-64 min-w-40 whitespace-normal">
              <Change c={c} />
            </TableCell>
            <TableCell>
              <span className="flex flex-col gap-0.5">
                <Badge variant="secondary">
                  {STATUS_LABEL[c.status] ?? c.status}
                </Badge>
                {c.reason && (
                  <span
                    className="max-w-64 truncate text-xs text-muted-foreground"
                    title={c.reason}
                  >
                    {c.reason}
                  </span>
                )}
              </span>
            </TableCell>
            <TableCell className="pr-4 text-right whitespace-nowrap text-muted-foreground tabular-nums">
              {when(c.decidedAt ?? "")}
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  )
}
