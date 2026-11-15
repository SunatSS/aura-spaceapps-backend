import { useQuery } from '@tanstack/react-query'
import { api, type Schemas } from '../../shared/api/client.ts'

export type Guidance = Schemas['Guidance']
export type Recommendation = Schemas['Recommendation']

export function useGuidance(astronautId: number | undefined) {
  return useQuery({
    queryKey: ['guidance', astronautId],
    queryFn: () => api<Guidance>(`/astronauts/${astronautId}/guidance`),
    enabled: astronautId !== undefined,
    refetchInterval: 60_000,
  })
}

/** Alert counts for the status bar: Warning = critical, Caution = caution (App. F F.6.4). */
export function alertCounts(g: Guidance | undefined): { warning: number; caution: number } {
  const recs = g?.recommendations ?? []
  return {
    warning: recs.filter((r) => r.severity === 'critical').length,
    caution: recs.filter((r) => r.severity === 'caution').length,
  }
}
