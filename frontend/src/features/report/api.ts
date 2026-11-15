import { useQuery } from '@tanstack/react-query'
import { api, ApiError, type Schemas } from '../../shared/api/client.ts'

export type Report = Schemas['Report']

/** Path of the plain-text report (also the download link). */
export const reportTextPath = (astronautId: number) => `/api/v1/astronauts/${astronautId}/report?format=text`

/** Flight-surgeon report for one crew member (FR-32). */
export function useReport(astronautId: number | undefined) {
  return useQuery({
    queryKey: ['report', astronautId],
    queryFn: () => api<Report>(`/astronauts/${astronautId}/report`),
    enabled: astronautId !== undefined,
  })
}

/** The exact text that goes down the link. */
export function useReportText(astronautId: number | undefined, enabled: boolean) {
  return useQuery({
    queryKey: ['report-text', astronautId],
    queryFn: async () => {
      const res = await fetch(reportTextPath(astronautId!))
      if (!res.ok) throw new ApiError(res.status, res.statusText)
      return res.text()
    },
    enabled: enabled && astronautId !== undefined,
  })
}

/** Saves the JSON report as a file (machine-readable export). */
export function downloadJSON(r: Report) {
  const surname = r.astronaut.name.split(' ').at(-1)?.toLowerCase().replace(/[^a-z0-9]/g, '') ?? 'member'
  const url = URL.createObjectURL(new Blob([JSON.stringify(r, null, 2)], { type: 'application/json' }))
  const a = document.createElement('a')
  a.href = url
  a.download = `aura-report-${surname}-md${r.mission.day}.json`
  a.click()
  URL.revokeObjectURL(url)
}
