import { useQuery } from '@tanstack/react-query'
import { api, type Schemas } from '../../shared/api/client.ts'

export type Astronaut = Schemas['Astronaut']

export function useAstronauts() {
  return useQuery({
    queryKey: ['astronauts'],
    queryFn: () => api<{ astronauts: Astronaut[] }>('/astronauts').then((r) => r.astronauts),
    staleTime: 5 * 60_000,
  })
}

export const roleLabel: Record<Astronaut['role'], string> = {
  commander: 'Commander',
  deputy_commander: 'Deputy commander',
  crew: 'Crew',
  flight_surgeon: 'Flight surgeon',
}
