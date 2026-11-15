import { Legend } from '../features/legend/components/Legend.tsx'
import { useSearchParams } from 'react-router'
import { BodyFigure, type Zone } from '../features/body-map/components/BodyFigure.tsx'
import { SystemDetail } from '../features/body-map/components/SystemDetail.tsx'
import { roleLabel } from '../features/crew/api.ts'
import { useCurrentAstronaut } from '../features/crew/context.ts'
import { useGuidance } from '../features/guidance/api.ts'
import { RecommendationCard } from '../features/guidance/components/RecommendationCard.tsx'
import { useAcknowledged } from '../features/guidance/useAcknowledged.ts'
import { useReadiness, type SystemResult } from '../features/readiness/api.ts'
import { systemCode } from '../features/readiness/codes.ts'
import { useCatalog } from '../shared/api/catalog.ts'
import { useMission } from '../shared/api/mission.ts'
import { alertClassOf } from '../shared/lib/status.ts'
import { Notice } from '../shared/ui/Notice.tsx'

const severity = { critical: 3, caution: 2, unknown: 1, nominal: 0 } as const

/** The system that most needs attention, to open by default. */
function worstSystem(systems: SystemResult[]): string | undefined {
  const withData = systems.filter((s) => s.indicators.length > 0)
  return [...withData].sort((a, b) => severity[b.status] - severity[a.status])[0]?.code ?? systems[0]?.code
}

/** Body map display: schematic figure + detail of the selected system (`?system=` in the URL). */
export function BodyMapPage() {
  const { current, isError: crewError } = useCurrentAstronaut()
  const catalog = useCatalog()
  const mission = useMission()
  const readiness = useReadiness(current?.id)
  const guidance = useGuidance(current?.id)
  const ack = useAcknowledged(current?.id)
  const [params, setParams] = useSearchParams()

  const systems = readiness.data?.systems ?? []
  const selected = systems.some((s) => s.code === params.get('system')) ? params.get('system')! : worstSystem(systems)
  const selectedIndex = systems.findIndex((s) => s.code === selected)
  const selectedSystem = systems[selectedIndex]
  const missionStart = mission.data ? new Date(mission.data.startedAt) : undefined

  const zones: Zone[] = systems.map((s, i) => ({
    code: s.code,
    sysCode: systemCode(i),
    name: catalog.data?.systems.find((m) => m.code === s.code)?.name ?? s.code,
    state: s.indicators.length === 0 ? 'unavailable' : alertClassOf[s.status],
  }))

  const recs = (guidance.data?.recommendations ?? []).filter((r) => r.system === selected)
  const error = crewError || catalog.isError || readiness.isError || guidance.isError

  return (
    <div className="flex flex-col gap-8">
      <header>
        <h1 className="text-[40px] leading-[48px] font-semibold tracking-[-0.01em]">Body map</h1>
        <p className="text-ink-muted">
          {current ? `${current.name} · ${roleLabel[current.role]}` : 'Loading crew…'} · select a body system to see its readings, NASA risks and
          recommendations.
        </p>
      </header>

      <Legend sections={['classes', 'values', 'zones', 'recommendations', 'statusbar']} />

      {error && <Notice alert="warning">Cannot load data from the AURA server. Check the link status in the top bar.</Notice>}

      {readiness.data && catalog.data && guidance.data && selectedSystem && (
        <div className="grid items-start gap-8 xl:grid-cols-[minmax(0,640px)_1fr]">
          <BodyFigure zones={zones} selected={selected} onSelect={(code) => setParams({ system: code }, { replace: true })} />
          <SystemDetail
            sysCode={systemCode(selectedIndex)}
            system={selectedSystem}
            catalog={catalog.data}
            missionStart={missionStart}
            recommendations={
              recs.length === 0 ? (
                <p className="text-ink-muted">No active recommendations for this system.</p>
              ) : (
                recs.map((r) => (
                  <RecommendationCard
                    key={r.id}
                    rec={r}
                    guidance={guidance.data}
                    acknowledged={ack.isAcknowledged(r.id)}
                    onAcknowledge={() => ack.acknowledge(r.id)}
                  />
                ))
              )
            }
          />
        </div>
      )}
    </div>
  )
}
