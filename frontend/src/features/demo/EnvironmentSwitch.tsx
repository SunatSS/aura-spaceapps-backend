import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useCatalog } from '../../shared/api/catalog.ts'
import { api } from '../../shared/api/client.ts'
import { useMission } from '../../shared/api/mission.ts'

/**
 * Demo control (FR-4): start a new mission phase in another environment now.
 * Gravity, Earth delay and environment-specific limits follow; Reset demo data
 * restores the scripted mission.
 */
export function EnvironmentSwitch() {
  const catalog = useCatalog()
  const mission = useMission()
  const qc = useQueryClient()
  const change = useMutation({
    mutationFn: (environment: string) => api('/demo/environment', { method: 'POST', body: JSON.stringify({ environment }) }),
    onSuccess: () => void qc.invalidateQueries(),
  })
  const current = mission.data?.currentPhase?.environment
  if (!catalog.data || !current) return null
  return (
    <label className="flex items-center gap-2">
      <span>Environment (demo)</span>
      <select
        value={current}
        disabled={change.isPending}
        onChange={(e) => change.mutate(e.target.value)}
        className="h-8 rounded-md border-[1.5px] border-line-strong bg-surface-raised px-2 text-sm font-semibold text-ink"
      >
        {catalog.data.environments.map((env) => (
          <option key={env.code} value={env.code}>
            {env.name}
          </option>
        ))}
      </select>
    </label>
  )
}
