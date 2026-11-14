import { useMutation, useQuery, useQueryClient, type QueryClient } from '@tanstack/react-query'
import { api, ApiError, type Schemas } from '../../shared/api/client.ts'

export type Session = Schemas['Session']
export type DemoAccounts = Schemas['DemoAccounts']

export const SESSION_KEY = ['session'] as const

/** Who is signed in; `null` when there is no valid session (401). */
export function useSession() {
  return useQuery({
    queryKey: SESSION_KEY,
    queryFn: async (): Promise<Session | null> => {
      try {
        return await api<Session>('/auth/me')
      } catch (e) {
        if (e instanceof ApiError && e.status === 401) return null
        throw e
      }
    },
    staleTime: Infinity,
  })
}

/** Demo logins for the sign-in screen; `null` when the server has no demo mission. */
export function useDemoAccounts() {
  return useQuery({
    queryKey: ['demo-accounts'],
    queryFn: async (): Promise<DemoAccounts | null> => {
      try {
        return await api<DemoAccounts>('/auth/demo-accounts')
      } catch (e) {
        if (e instanceof ApiError && e.status === 404) return null
        throw e
      }
    },
    staleTime: Infinity,
  })
}

/**
 * Drops every cached query (they belong to the previous user) and stores the
 * new session. The session query itself is kept, so the gate observing it
 * re-renders (clear() would orphan that observer).
 */
function switchSession(qc: QueryClient, s: Session | null) {
  qc.removeQueries({ predicate: (q) => q.queryKey[0] !== SESSION_KEY[0] })
  qc.setQueryData(SESSION_KEY, s)
}

export function useLogin() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (body: Schemas['LoginRequest']) => api<Session>('/auth/login', { method: 'POST', body: JSON.stringify(body) }),
    onSuccess: (s) => switchSession(qc, s),
  })
}

export function useLogout() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: () => api<null>('/auth/logout', { method: 'POST' }),
    onSettled: () => switchSession(qc, null),
  })
}

/** Restores the demo mission (FR-62). Local acknowledgements go too, so the demo starts clean. */
export function useResetDemo() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: () => api<Session>('/demo/reset', { method: 'POST' }),
    onSuccess: (s) => {
      for (const key of Object.keys(localStorage)) {
        if (key.startsWith('aura.ack.')) localStorage.removeItem(key)
      }
      switchSession(qc, s)
    },
  })
}

/** Mirrors the backend policy (auth.SeesWholeCrew): who may open other members and the crew display. */
export function seesWholeCrew(role: Session['user']['role']): boolean {
  return role === 'commander' || role === 'deputy_commander' || role === 'flight_surgeon'
}
