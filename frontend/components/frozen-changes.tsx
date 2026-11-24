"use client"

import * as React from "react"
import { useRouter } from "next/navigation"
import { CheckIcon, Undo2Icon } from "lucide-react"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
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

/** Pending changes to completed work, each with Approve and Reject. */
export function PendingChanges({ rows }: { rows: T.FrozenChange[] }) {
  const router = useRouter()
  const [busy, setBusy] = React.useState<string>("")
  const [notice, setNotice] = React.useState("")
  const [rejecting, setRejecting] = React.useState<T.FrozenChange | null>(null)
  const [reason, setReason] = React.useState("")

  const decide = async (
    c: T.FrozenChange,
    action: "approve" | "reject",
    body: object = {}
  ) => {
    setBusy(`${c.id}:${action}`)
    setNotice("")
    try {
      await clientFetch(
        `frozen-changes/${c.id}/${action}`,
        {},
        { method: "POST", body: JSON.stringify(body) }
      )
      setRejecting(null)
      setReason("")
    } catch (e) {
      setNotice(
        e instanceof BackendError
          ? e.body.message
          : "Could not reach the dashboard"
      )
    } finally {
      setBusy("")
      router.refresh()
    }
  }

  if (!rows.length)
    return (
      <p className="px-4 pb-4 text-sm text-muted-foreground">
        Nothing waiting. Changes Linear makes to completed work show up here
        after a sync.
      </p>
    )

  return (
    <>
      {notice && (
        <p className="px-4 pb-3 text-sm" style={{ color: "var(--warn)" }}>
          {notice}
        </p>
      )}
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead className="pl-4">Task or project</TableHead>
            <TableHead>Change</TableHead>
            <TableHead>Frozen because</TableHead>
            <TableHead>Seen</TableHead>
            <TableHead className="pr-4 text-right">Decision</TableHead>
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
                    onClick={() => decide(c, "approve")}
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
                    onClick={() => setRejecting(c)}
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
            <DialogTitle>Reject and undo in Linear?</DialogTitle>
            <DialogDescription>
              {rejecting && (
                <>
                  {rejecting.what} of{" "}
                  {rejecting.itemIdentifier || rejecting.itemTitle} goes back to{" "}
                  {rejecting.oldValue} in Linear, and a comment there says why:{" "}
                  {rejecting.frozenBy}, so it is frozen.
                </>
              )}
            </DialogDescription>
          </DialogHeader>
          <label className="flex flex-col gap-1.5 text-sm">
            Reason (optional, added to the comment)
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
              {busy.endsWith(":reject") && <Spinner data-icon="inline-start" />}
              Reject and undo
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
