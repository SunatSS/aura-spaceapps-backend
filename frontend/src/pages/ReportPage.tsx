import clsx from 'clsx'
import { useState } from 'react'
import { roleLabel } from '../features/crew/api.ts'
import { useCurrentAstronaut } from '../features/crew/context.ts'
import { downloadJSON, reportTextPath, useReport, useReportText, type Report } from '../features/report/api.ts'
import { formatNumber } from '../shared/lib/format.ts'
import { alertClassOf, alertText } from '../shared/lib/status.ts'
import { formatDelay, formatMET } from '../shared/lib/time.ts'
import { Button } from '../shared/ui/Button.tsx'
import { Notice } from '../shared/ui/Notice.tsx'
import { StatusBadge } from '../shared/ui/StatusBadge.tsx'

/**
 * Report display (FR-32): what the flight surgeon receives — readiness,
 * out-of-range readings, recommendations with their basis and 7-day trends.
 * Exported as plain text (for a narrow downlink) and JSON; sending is simulated.
 */
export function ReportPage() {
  const { me, current } = useCurrentAstronaut()
  const report = useReport(current?.id)
  const [showText, setShowText] = useState(false)
  const text = useReportText(current?.id, showText)
  const [sentAt, setSentAt] = useState<{ id: number; at: Date } | null>(null)
  const r = report.data
  const start = r ? new Date(r.mission.startedAt) : undefined
  const sent = sentAt && sentAt.id === current?.id ? sentAt.at : null

  return (
    <div className="flex flex-col gap-8">
      <header>
        <h1 className="text-[40px] leading-[48px] font-semibold tracking-[-0.01em]">Report</h1>
        <p className="text-ink-muted">
          {current ? `${current.name} · ${roleLabel[current.role]}` : 'Loading crew…'} · summary for the flight surgeon: readiness, alerts and the last 7 days.
        </p>
      </header>

      {report.isError && <Notice alert="warning">Cannot build the report. Check the link status in the top bar.</Notice>}

      {r && start && current && (
        <>
          <div className="flex flex-wrap items-center gap-3">
            {me.role !== 'flight_surgeon' && (
              <Button variant="command" onClick={() => setSentAt({ id: current.id, at: new Date() })} disabled={sent !== null}>
                {sent ? 'Queued for downlink' : 'Send to flight surgeon'}
              </Button>
            )}
            <a
              href={reportTextPath(current.id)}
              download
              className="inline-flex h-10 items-center rounded-md border-[1.5px] border-line-strong px-4 text-[15px] leading-5 font-semibold text-ink hover:border-ink"
            >
              Download text
            </a>
            <Button onClick={() => downloadJSON(r)}>Download JSON</Button>
          </div>

          {sent && (
            <Notice alert="advisory">
              Queued for the next downlink at <span className="font-mono">MET {formatMET(start, sent).slice(0, -3)}</span>. Earth receives it in{' '}
              <span className="font-mono">{formatDelay(r.mission.commDelayMinS, r.mission.commDelayMaxS)}</span>, and a reply takes as long again — follow the
              recommendations meanwhile. (Simulated: this prototype has no real link.)
            </Notice>
          )}

          <Summary r={r} start={start} />
          <Alerts r={r} start={start} />
          <Recommendations r={r} />
          <CabinAir r={r} />
          <Trends r={r} />

          <section className="flex flex-col gap-3">
            <Button className="w-fit" aria-expanded={showText} onClick={() => setShowText((v) => !v)}>
              {showText ? 'Hide plain-text version' : 'Show plain-text version'}
            </Button>
            {showText && (
              <pre className="overflow-x-auto rounded-lg border-[1.5px] border-line bg-surface p-4 font-mono text-sm leading-5 whitespace-pre">
                {text.data ?? (text.isError ? 'Cannot load the text version.' : 'Loading…')}
              </pre>
            )}
          </section>

          <p className="text-sm leading-5 text-ink-muted">{r.disclaimer}</p>
        </>
      )}
    </div>
  )
}

function Section({ id, title, children }: { id: string; title: string; children: React.ReactNode }) {
  return (
    <section aria-labelledby={id} className="flex flex-col gap-3">
      <h2 id={id} className="text-2xl leading-8 font-semibold">
        {title}
      </h2>
      {children}
    </section>
  )
}

function Summary({ r, start }: { r: Report; start: Date }) {
  const cls = alertClassOf[r.readiness.status]
  return (
    <Section id="summary-title" title="Summary">
      <dl className="grid grid-cols-[max-content_1fr] gap-x-6 gap-y-2 rounded-lg border-[1.5px] border-line bg-surface-raised p-6">
        <dt className="font-medium text-ink-muted">Readiness</dt>
        <dd className="flex flex-wrap items-center gap-3">
          <span className="font-mono text-2xl font-medium">{r.readiness.score ?? '—'}</span>
          <span className="font-mono text-ink-muted">/100</span>
          <StatusBadge alert={cls} />
          {r.readiness.cappedByCritical && <span className="text-sm text-ink-muted">Capped at 49 by a Warning reading.</span>}
        </dd>
        <dt className="font-medium text-ink-muted">Fresh data</dt>
        <dd className="font-mono">
          {r.readiness.freshIndicators}/{r.readiness.totalIndicators} indicators
        </dd>
        <dt className="font-medium text-ink-muted">Escalation</dt>
        <dd>{r.escalate ? <span className="font-semibold text-warning-ink">Report to flight surgeon requested</span> : 'None'}</dd>
        <dt className="font-medium text-ink-muted">Mission</dt>
        <dd>
          {r.mission.name} · day <span className="font-mono">{r.mission.day}</span> · {r.mission.phase} (<span className="font-mono">{r.mission.gravityG} g</span>)
        </dd>
        <dt className="font-medium text-ink-muted">Generated</dt>
        <dd className="font-mono">MET {formatMET(start, new Date(r.generatedAt)).slice(0, -3)}</dd>
      </dl>
    </Section>
  )
}

function Alerts({ r, start }: { r: Report; start: Date }) {
  return (
    <Section id="alerts-title" title={`Out-of-range readings (${r.alerts.length})`}>
      {r.alerts.length === 0 ? (
        <p className="text-ink-muted">All fresh readings are nominal.</p>
      ) : (
        <div className="overflow-x-auto rounded-lg border-[1.5px] border-line bg-surface-raised">
          <table className="w-full text-left">
            <thead className="text-sm leading-5 text-ink-muted">
              <tr className="border-b border-line">
                <th className="px-4 py-2 font-medium">Class</th>
                <th className="px-4 py-2 font-medium">Indicator</th>
                <th className="px-4 py-2 font-medium">Value</th>
                <th className="px-4 py-2 font-medium">Nominal</th>
                <th className="px-4 py-2 font-medium">Measured</th>
              </tr>
            </thead>
            <tbody>
              {r.alerts.map((a) => {
                const cls = alertClassOf[a.status]
                return (
                  <tr key={a.indicator} className="border-b border-line last:border-0">
                    <td className="px-4 py-2">
                      <StatusBadge alert={cls} />
                    </td>
                    <td className="px-4 py-2">{a.name}</td>
                    <td className="px-4 py-2 font-mono font-medium">
                      {a.display} <span className={alertText[cls]}>{a.direction === 'below' ? '↓' : '↑'}</span>
                    </td>
                    <td className="px-4 py-2 font-mono text-sm text-ink-muted">{a.nominal}</td>
                    <td className="px-4 py-2 font-mono text-sm">MET {formatMET(start, new Date(a.measuredAt)).slice(0, -3)}</td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      )}
      {r.stale.length > 0 && (
        <Notice alert="stale">
          No fresh reading: {r.stale.map((s) => s.name).join(', ')}.
        </Notice>
      )}
    </Section>
  )
}

function Recommendations({ r }: { r: Report }) {
  return (
    <Section id="recs-title" title={`Active recommendations (${r.recommendations.length})`}>
      {r.recommendations.length === 0 ? (
        <p className="text-ink-muted">None.</p>
      ) : (
        <ol className="flex flex-col gap-3">
          {r.recommendations.map((rec) => (
            <li key={rec.id} className="flex flex-col gap-2 rounded-md border-[1.5px] border-line bg-surface-raised p-4">
              <div className="flex flex-wrap items-center gap-2">
                <StatusBadge alert={alertClassOf[rec.severity]} />
                {rec.escalate && (
                  <span className="inline-flex h-6 items-center rounded-sm border-[1.5px] border-warning-ink px-2 text-sm font-semibold text-warning-ink">
                    Report to flight surgeon
                  </span>
                )}
                <h3 className="text-lg leading-[26px] font-semibold">{rec.title}</h3>
              </div>
              <p>{rec.action}</p>
              <dl className="grid grid-cols-[max-content_1fr] gap-x-4 gap-y-1 text-sm leading-5">
                <dt className="font-medium text-ink-muted">Basis</dt>
                <dd className="font-mono">{rec.basis}</dd>
                {rec.recheckMinutes > 0 && (
                  <>
                    <dt className="font-medium text-ink-muted">Re-check</dt>
                    <dd className="font-mono">in {rec.recheckMinutes} min</dd>
                  </>
                )}
                {rec.escalate && (
                  <>
                    <dt className="font-medium text-ink-muted">Why report</dt>
                    <dd>{rec.escalateReason}</dd>
                  </>
                )}
              </dl>
            </li>
          ))}
        </ol>
      )}
    </Section>
  )
}

const n = (v: number | null) => (v === null ? '—' : formatNumber(v))

/** Air in the member's module now and over 7 days (FR-84). */
function CabinAir({ r }: { r: Report }) {
  if (!r.cabin) return null
  return (
    <Section id="cabin-title" title={`Cabin air — ${r.cabin.name}, last 7 days`}>
      <div className="overflow-x-auto rounded-lg border-[1.5px] border-line bg-surface-raised">
        <table className="w-full text-left">
          <thead className="text-sm leading-5 text-ink-muted">
            <tr className="border-b border-line">
              <th className="px-4 py-2 font-medium">Parameter</th>
              <th className="px-4 py-2 font-medium">Now</th>
              <th className="px-4 py-2 font-medium">Class</th>
              <th className="px-4 py-2 font-medium">Nominal</th>
              <th className="px-4 py-2 text-right font-medium">Min</th>
              <th className="px-4 py-2 text-right font-medium">Max</th>
              <th className="px-4 py-2 text-right font-medium">Off-nominal</th>
            </tr>
          </thead>
          <tbody className="font-mono text-sm">
            {r.cabin.parameters.map((c) => (
              <tr key={c.parameter} className="border-b border-line last:border-0">
                <td className="px-4 py-2 font-sans text-base">{c.name}</td>
                <td className="px-4 py-2">
                  {n(c.value)} <span className="text-ink-muted">{c.unit}</span>
                </td>
                <td className="px-4 py-2 font-sans">
                  <StatusBadge alert={alertClassOf[c.status]} quiet />
                </td>
                <td className="px-4 py-2 text-ink-muted">{c.nominal}</td>
                <td className="px-4 py-2 text-right">{n(c.min)}</td>
                <td className="px-4 py-2 text-right">{n(c.max)}</td>
                <td className={clsx('px-4 py-2 text-right', c.offNominal > 0 && 'font-semibold')}>{c.offNominal}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </Section>
  )
}

function Trends({ r }: { r: Report }) {
  return (
    <Section id="trends-title" title="Trends, last 7 days">
      <div className="overflow-x-auto rounded-lg border-[1.5px] border-line bg-surface-raised">
        <table className="w-full text-left">
          <thead className="text-sm leading-5 text-ink-muted">
            <tr className="border-b border-line">
              <th className="px-4 py-2 font-medium">Indicator</th>
              <th className="px-4 py-2 font-medium">Nominal</th>
              <th className="px-4 py-2 text-right font-medium">Readings</th>
              <th className="px-4 py-2 text-right font-medium">Min</th>
              <th className="px-4 py-2 text-right font-medium">Max</th>
              <th className="px-4 py-2 text-right font-medium">Mean</th>
              <th className="px-4 py-2 text-right font-medium">Last</th>
              <th className="px-4 py-2 text-right font-medium">Change</th>
              <th className="px-4 py-2 text-right font-medium">Off-nominal</th>
            </tr>
          </thead>
          <tbody className="font-mono text-sm">
            {r.trends.map((t) => (
              <tr key={t.indicator} className="border-b border-line last:border-0">
                <td className="px-4 py-2 font-sans text-base">
                  {t.name} <span className="font-mono text-sm text-ink-muted">{t.unit}</span>
                </td>
                <td className="px-4 py-2 text-ink-muted">{t.nominal}</td>
                <td className="px-4 py-2 text-right">{t.readings}</td>
                <td className="px-4 py-2 text-right">{n(t.min)}</td>
                <td className="px-4 py-2 text-right">{n(t.max)}</td>
                <td className="px-4 py-2 text-right">{n(t.mean)}</td>
                <td className="px-4 py-2 text-right">{n(t.last)}</td>
                <td className="px-4 py-2 text-right">{t.change === null ? '—' : `${t.change > 0 ? '+' : ''}${formatNumber(t.change)}`}</td>
                <td className={clsx('px-4 py-2 text-right', t.offNominal > 0 && 'font-semibold')}>{t.offNominal}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </Section>
  )
}
