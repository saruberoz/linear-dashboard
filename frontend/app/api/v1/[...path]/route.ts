// Passes /api/v1/* through to the backend. BACKEND_URL is read on every request,
// so the same image works in any environment (Next.js rewrites are fixed at build time).
import type { NextRequest } from "next/server"

export const dynamic = "force-dynamic"

async function forward(
  req: NextRequest,
  { params }: { params: Promise<{ path: string[] }> }
) {
  const backend = process.env.BACKEND_URL ?? "http://localhost:8000"
  const { path } = await params
  const url = `${backend}/v1/${path.map(encodeURIComponent).join("/")}${req.nextUrl.search}`
  const res = await fetch(url, {
    method: req.method,
    headers: { "content-type": "application/json" },
    body:
      req.method === "GET" || req.method === "DELETE"
        ? undefined
        : await req.text(),
    cache: "no-store",
  })
  return new Response(res.body, {
    status: res.status,
    headers: { "content-type": "application/json" },
  })
}

export { forward as DELETE, forward as GET, forward as POST, forward as PUT }
