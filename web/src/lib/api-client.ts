import { env } from "@/env"

const BASE_URL = env.VITE_VIGIL_API_URL.replace(/\/$/, "")

export class ApiError extends Error {
  constructor(
    public status: number,
    message: string,
  ) {
    super(message)
    this.name = "ApiError"
  }
}

type QueryValue = string | number | boolean | undefined | null

/** Thin typed fetch wrapper around the keelwave API. */
async function request<T>(
  path: string,
  params?: Record<string, QueryValue>,
): Promise<T> {
  const url = new URL(`${BASE_URL}${path}`)
  if (params) {
    for (const [key, value] of Object.entries(params)) {
      if (value !== undefined && value !== null) {
        url.searchParams.set(key, String(value))
      }
    }
  }

  const res = await fetch(url, {
    headers: { Accept: "application/json" },
  })

  if (!res.ok) {
    const body = (await res.json().catch(() => null)) as {
      error?: string
    } | null
    throw new ApiError(res.status, body?.error ?? res.statusText)
  }

  // The Go API wraps every success body in a { "data": ... } envelope.
  const body = (await res.json()) as { data: T }
  return body.data
}

export const apiClient = {
  get: request,
}
