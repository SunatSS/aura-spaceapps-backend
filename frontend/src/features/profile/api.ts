import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api, type Schemas } from '../../shared/api/client.ts'

export type Profile = Schemas['Profile']
export type PersonalIndicator = Schemas['PersonalIndicator']

/** Known conditions, baselines, personal ranges and weights (FR-6, FR-24). */
export function useProfile(astronautId: number | undefined) {
  return useQuery({
    queryKey: ['profile', astronautId],
    queryFn: () => api<Profile>(`/astronauts/${astronautId}/profile`),
    enabled: astronautId !== undefined,
  })
}

type Update = { conditions?: string; indicators?: Record<string, PersonalIndicator>; weights?: Record<string, number> }

/** Flight surgeon only; readiness and guidance change with it. */
export function useUpdateProfile(astronautId: number | undefined) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (u: Update) => api<Profile>(`/astronauts/${astronautId}/profile`, { method: 'PUT', body: JSON.stringify(u) }),
    onSuccess: (p) => {
      qc.setQueryData(['profile', astronautId], p)
      for (const key of ['readiness', 'guidance', 'crew-overview', 'report', 'measurements']) void qc.invalidateQueries({ queryKey: [key] })
    },
  })
}
