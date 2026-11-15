/**
 * Thin fetch wrapper for the AURA API. All requests go to /api/v1 on the same
 * origin (Vite proxies to the Go backend in dev).
 *
 * Response types come from ./schema.d.ts, generated from
 * backend/api/openapi.yaml by `npm run gen:api` — never hand-edit it.
 */
import type { components } from './schema'

export type Schemas = components['schemas']

const BASE = '/api/v1'

export class ApiError extends Error {
  readonly status: number

  constructor(status: number, message: string) {
    super(message)
    this.status = status
  }
}

export async function api<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    ...init,
    headers: { 'Content-Type': 'application/json', ...init?.headers },
  })
  const body: unknown = await res.json().catch(() => null)
  if (!res.ok) {
    const msg = (body as Schemas['Error'] | null)?.error ?? res.statusText
    throw new ApiError(res.status, msg)
  }
  return body as T
}
