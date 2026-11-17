import type { ApiError } from "@/lib/types"

/** Thrown for non-2xx backend replies; carries the backend's reason code. */
export class BackendError extends Error {
  constructor(
    public status: number,
    public body: ApiError
  ) {
    super(body.message || `backend returned ${status}`)
  }
}

async function parse<T>(res: Response): Promise<T> {
  const body = await res
    .json()
    .catch(() => ({ code: res.status, reason: "", message: res.statusText }))
  if (!res.ok) throw new BackendError(res.status, body as ApiError)
  return body as T
}

const query = (params: Record<string, string | boolean | undefined>) => {
  const q = new URLSearchParams()
  for (const [k, v] of Object.entries(params))
    if (v !== undefined && v !== "" && v !== false) q.set(k, String(v))
  const s = q.toString()
  return s ? `?${s}` : ""
}

/** Server-side call straight to the backend (used by the page's server component). */
export async function serverGet<T>(
  path: string,
  params: Record<string, string | boolean | undefined> = {}
): Promise<T> {
  const backend = process.env.BACKEND_URL ?? "http://localhost:8000"
  return parse<T>(
    await fetch(`${backend}/v1/${path}${query(params)}`, { cache: "no-store" })
  )
}

/** Browser-side call through the frontend's /api/v1 pass-through. */
export async function clientFetch<T>(
  path: string,
  params: Record<string, string | boolean | undefined> = {},
  init?: RequestInit
): Promise<T> {
  return parse<T>(
    await fetch(`/api/v1/${path}${query(params)}`, {
      cache: "no-store",
      ...init,
    })
  )
}
