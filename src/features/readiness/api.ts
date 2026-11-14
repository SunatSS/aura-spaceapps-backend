import { useQuery } from '@tanstack/react-query'
import { api, type Schemas } from '../../shared/api/client.ts'

export type Readiness = Schemas['Readiness']
export type SystemResult = Schemas['SystemResult']
export type IndicatorResult = Schemas['IndicatorResult']

export function useReadiness(astronautId: number | undefined) {
  return useQuery({
    queryKey: ['readiness', astronautId],
    queryFn: () => api<Readiness>(`/astronauts/${astronautId}/readiness`),
    enabled: astronautId !== undefined,
    refetchInterval: 60_000,
  })
}
