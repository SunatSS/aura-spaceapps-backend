import { useQuery } from '@tanstack/react-query'
import { api, type Schemas } from '../../shared/api/client.ts'

export type CrewOverview = Schemas['CrewOverview']
export type CrewMember = Schemas['CrewMember']
export type TrendPoint = Schemas['TrendPoint']

/** Whole crew at a glance (commander, deputy, flight surgeon). */
export function useCrewOverview(enabled: boolean) {
  return useQuery({
    queryKey: ['crew-overview'],
    queryFn: () => api<CrewOverview>('/crew/overview'),
    enabled,
    refetchInterval: 60_000,
  })
}
