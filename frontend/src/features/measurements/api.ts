import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api, type Schemas } from '../../shared/api/client.ts'

export type MeasurementView = Schemas['MeasurementView']
export type NewMeasurement = Schemas['NewMeasurement']
export type Activity = Schemas['Activity']

/** Reading history for one indicator since `from` (chronological). */
export function useMeasurements(astronautId: number | undefined, indicator: string, from: Date) {
  const fromIso = from.toISOString().slice(0, 13) + ':00:00Z' // hour precision keeps the query key stable
  return useQuery({
    queryKey: ['measurements', astronautId, indicator, fromIso],
    queryFn: () =>
      api<{ measurements: MeasurementView[] }>(
        `/astronauts/${astronautId}/measurements?indicator=${encodeURIComponent(indicator)}&from=${encodeURIComponent(fromIso)}`,
      ).then((r) => r.measurements),
    enabled: astronautId !== undefined,
  })
}

export function useActivities(astronautId: number | undefined, from: Date) {
  const fromIso = from.toISOString().slice(0, 13) + ':00:00Z'
  return useQuery({
    queryKey: ['activities', astronautId, fromIso],
    queryFn: () => api<{ activities: Activity[] }>(`/astronauts/${astronautId}/activities?from=${encodeURIComponent(fromIso)}`).then((r) => r.activities),
    enabled: astronautId !== undefined,
  })
}

/** Save a batch of readings; refreshes every view that depends on them. */
export function useCreateMeasurements(astronautId: number | undefined) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (measurements: NewMeasurement[]) =>
      api<{ measurements: MeasurementView[] }>(`/astronauts/${astronautId}/measurements`, {
        method: 'POST',
        body: JSON.stringify({ measurements }),
      }).then((r) => r.measurements),
    onSuccess: () => {
      for (const key of ['readiness', 'guidance', 'measurements', 'tasks', 'report']) {
        void qc.invalidateQueries({ queryKey: [key, astronautId] })
      }
      void qc.invalidateQueries({ queryKey: ['crew-overview'] })
    },
  })
}
