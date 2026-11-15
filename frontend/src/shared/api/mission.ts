import { useQuery } from '@tanstack/react-query'
import { api, type Schemas } from './client.ts'

export function useMission() {
  return useQuery({
    queryKey: ['mission', 'current'],
    queryFn: () => api<Schemas['CurrentMission']>('/missions/current'),
    staleTime: 5 * 60_000,
  })
}
