import { useQuery } from '@tanstack/react-query'
import { api, type Schemas } from './client.ts'

export function useHealth() {
  return useQuery({
    queryKey: ['health'],
    queryFn: () => api<Schemas['Health']>('/health'),
    refetchInterval: 15_000,
  })
}
