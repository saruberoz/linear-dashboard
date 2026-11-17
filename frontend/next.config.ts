import type { NextConfig } from "next"

// "standalone" emits a self-contained server for the Docker image.
// The backend address is read at runtime by app/api/v1/[...path]/route.ts, not here.
const nextConfig: NextConfig = { output: "standalone" }

export default nextConfig
