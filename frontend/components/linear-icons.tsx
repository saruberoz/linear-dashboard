// Small Linear-style glyphs for priority and status.

export function PriorityIcon({ priority }: { priority: number }) {
  if (priority === 1) {
    return (
      <svg viewBox="0 0 16 16" className="size-3.5 shrink-0" aria-hidden>
        <rect x="1" y="1" width="14" height="14" rx="3" fill="#f2994a" />
        <rect x="7.1" y="4" width="1.8" height="5" rx=".9" fill="#fff" />
        <rect x="7.1" y="10.4" width="1.8" height="1.8" rx=".9" fill="#fff" />
      </svg>
    )
  }
  if (priority === 0) {
    return (
      <svg
        viewBox="0 0 16 16"
        className="size-3.5 shrink-0 text-muted-foreground"
        aria-hidden
      >
        {[3, 8, 13].map((x) => (
          <rect
            key={x}
            x={x - 1}
            y="7.25"
            width="2"
            height="1.5"
            rx=".5"
            fill="currentColor"
          />
        ))}
      </svg>
    )
  }
  const filled = 5 - priority // High 3, Medium 2, Low 1
  return (
    <svg viewBox="0 0 16 16" className="size-3.5 shrink-0" aria-hidden>
      {[0, 1, 2].map((n) => (
        <rect
          key={n}
          x={2 + n * 4.5}
          y={10 - n * 3.5}
          width="3"
          height={4 + n * 3.5}
          rx="1"
          className={
            n < filled ? "fill-foreground/80" : "fill-muted-foreground/30"
          }
        />
      ))}
    </svg>
  )
}

export function StatusIcon({ status }: { status: string }) {
  if (status === "done") {
    return (
      <svg viewBox="0 0 14 14" className="size-3.5 shrink-0" aria-hidden>
        <circle cx="7" cy="7" r="6" fill="var(--st-done)" />
        <path
          d="M4.3 7.2l1.8 1.8 3.6-3.8"
          stroke="#fff"
          strokeWidth="1.4"
          fill="none"
          strokeLinecap="round"
          strokeLinejoin="round"
        />
      </svg>
    )
  }
  if (status === "in_progress") {
    return (
      <svg viewBox="0 0 14 14" className="size-3.5 shrink-0" aria-hidden>
        <circle
          cx="7"
          cy="7"
          r="5.5"
          fill="none"
          stroke="var(--st-progress)"
          strokeWidth="1.5"
        />
        <path d="M7 3.5a3.5 3.5 0 0 1 0 7z" fill="var(--st-progress)" />
      </svg>
    )
  }
  if (status === "planned") {
    return (
      <svg
        viewBox="0 0 14 14"
        className="size-3.5 shrink-0 text-muted-foreground"
        aria-hidden
      >
        <circle
          cx="7"
          cy="7"
          r="5.5"
          fill="none"
          stroke="currentColor"
          strokeWidth="1.5"
        />
      </svg>
    )
  }
  if (status === "paused") {
    return (
      <svg viewBox="0 0 14 14" className="size-3.5 shrink-0" aria-hidden>
        <circle
          cx="7"
          cy="7"
          r="5.5"
          fill="none"
          stroke="var(--st-progress)"
          strokeWidth="1.5"
        />
        <path
          d="M5.6 4.8v4.4M8.4 4.8v4.4"
          stroke="var(--st-progress)"
          strokeWidth="1.5"
          strokeLinecap="round"
        />
      </svg>
    )
  }
  if (status === "canceled") {
    return (
      <svg
        viewBox="0 0 14 14"
        className="size-3.5 shrink-0 text-muted-foreground"
        aria-hidden
      >
        <circle cx="7" cy="7" r="6" fill="currentColor" opacity=".5" />
        <path
          d="M5 5l4 4M9 5l-4 4"
          stroke="var(--background)"
          strokeWidth="1.4"
          strokeLinecap="round"
        />
      </svg>
    )
  }
  return (
    <svg
      viewBox="0 0 14 14"
      className="size-3.5 shrink-0 text-muted-foreground"
      aria-hidden
    >
      <circle
        cx="7"
        cy="7"
        r="5.5"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.5"
        strokeDasharray="2.2 1.8"
      />
    </svg>
  )
}

// Linear initiative statuses and project states, keyed lower-case, with the
// StatusIcon glyph each one uses.
const PLAN_STATUS: Record<string, { label: string; glyph: string }> = {
  proposed: { label: "Proposed", glyph: "backlog" },
  backlog: { label: "Backlog", glyph: "backlog" },
  planned: { label: "Planned", glyph: "planned" },
  active: { label: "Active", glyph: "in_progress" },
  started: { label: "In progress", glyph: "in_progress" },
  paused: { label: "Paused", glyph: "paused" },
  completed: { label: "Completed", glyph: "done" },
  canceled: { label: "Canceled", glyph: "canceled" },
}

/** The display name of an initiative status or project state. */
export function planStatusLabel(status: string): string {
  return PLAN_STATUS[status.toLowerCase()]?.label ?? status
}

/** The status glyph of an initiative or project, named for screen readers. */
export function PlanStatusIcon({ status }: { status: string }) {
  if (!status) return null
  const glyph = PLAN_STATUS[status.toLowerCase()]?.glyph ?? "backlog"
  return (
    <span role="img" aria-label={planStatusLabel(status)} className="flex">
      <StatusIcon status={glyph} />
    </span>
  )
}

export function SegmentDot({ color }: { color: string }) {
  return (
    <span
      className="inline-block size-2 shrink-0 rounded-full"
      style={{ background: color }}
    />
  )
}
