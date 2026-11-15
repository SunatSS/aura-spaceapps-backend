import { Legend } from '../features/legend/components/Legend.tsx'
import { Link } from 'react-router'
import { roleLabel } from '../features/crew/api.ts'
import { systemCode } from '../features/readiness/codes.ts'
import { useCurrentAstronaut } from '../features/crew/context.ts'
import { useGuidance } from '../features/guidance/api.ts'
import { RecommendationCard } from '../features/guidance/components/RecommendationCard.tsx'
import { useAcknowledged } from '../features/guidance/useAcknowledged.ts'
import { useReadiness } from '../features/readiness/api.ts'
import { ReadinessScore } from '../features/readiness/components/ReadinessScore.tsx'
import { SystemTile } from '../features/readiness/components/SystemTile.tsx'
import { useCatalog } from '../shared/api/catalog.ts'
import { useMission } from '../shared/api/mission.ts'
import { Notice } from '../shared/ui/Notice.tsx'
import { TodayPanel } from '../features/tasks/components/TodayPanel.tsx'

/** Readiness display: score, recommendations, body systems (App. F: answer first, details on demand). */
export function DashboardPage() {
  const { me, current, isError: crewError } = useCurrentAstronaut()
  const catalog = useCatalog()
  const mission = useMission()
  const readiness = useReadiness(current?.id)
  const guidance = useGuidance(current?.id)
  const ack = useAcknowledged(current?.id)

  const error = crewError || catalog.isError || readiness.isError || guidance.isError
  const missionStart = mission.data ? new Date(mission.data.startedAt) : undefined

  // Unacknowledged first, then by the server's severity order.
  const recs = [...(guidance.data?.recommendations ?? [])].sort(
    (a, b) => Number(ack.isAcknowledged(a.id)) - Number(ack.isAcknowledged(b.id)),
  )

  return (
    <div className="flex flex-col gap-8">
      <header>
        <h1 className="text-[40px] leading-[48px] font-semibold tracking-[-0.01em]">Readiness</h1>
        <p className="text-ink-muted">
          {current ? `${current.name} · ${roleLabel[current.role]}` : 'Loading crew…'}
          {mission.data?.currentPhase && ` · ${mission.data.currentPhase.name}`}
          {mission.data?.environment && ` · ${mission.data.environment.gravityG} g`}
        </p>
      </header>

      <Legend sections={['classes', 'values', 'tiles', 'score', 'recommendations', 'tasks', 'statusbar']} />

      {error && <Notice alert="warning">Cannot load data from the AURA server. Check the link status in the top bar.</Notice>}

      {readiness.data && catalog.data && guidance.data && (
        <>
          <div className="grid items-start gap-6 lg:grid-cols-[320px_1fr]">
            <ReadinessScore readiness={readiness.data} indicatorCount={catalog.data.indicators.length} />
            <section aria-labelledby="recs-title" className="flex flex-col gap-4">
              <h2 id="recs-title" className="text-2xl leading-8 font-semibold">
                Recommendations
              </h2>
              {recs.length === 0 ? (
                <Notice alert="nominal">No active recommendations — all fresh readings are nominal.</Notice>
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
              )}
            </section>
          </div>

          {current && <TodayPanel astronautId={current.id} canMark={current.id === me.id} catalog={catalog.data} missionStart={missionStart} />}

          <section aria-labelledby="systems-title" className="flex flex-col gap-4">
            <h2 id="systems-title" className="text-2xl leading-8 font-semibold">
              Body systems
            </h2>
            <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
              {readiness.data.systems.map((s, i) => (
                <Link
                  key={s.code}
                  to={`/body?system=${s.code}`}
                  className="flex rounded-md [&>article]:flex-1 hover:[&>article]:border-ink"
                  aria-label={`Open ${systemCode(i)} on the body map`}
                >
                  <SystemTile code={systemCode(i)} system={s} catalog={catalog.data} missionStart={missionStart} />
                </Link>
              ))}
            </div>
          </section>

          <p className="text-sm leading-5 text-ink-muted">{catalog.data.thresholdsNote}</p>
        </>
      )}
    </div>
  )
}
