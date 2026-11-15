import clsx from 'clsx'
import { useState, type ReactNode } from 'react'
import type { AlertClass } from '../../../shared/lib/status.ts'
import { StatusBadge } from '../../../shared/ui/StatusBadge.tsx'

export type LegendSection = 'classes' | 'values' | 'tiles' | 'zones' | 'crew' | 'vehicle' | 'supplies' | 'tasks' | 'profile' | 'score' | 'recommendations' | 'statusbar'

type Props = {
  /** Which parts are relevant on this display, in display order. */
  sections: LegendSection[]
}

const STORAGE_KEY = 'aura.legend.open'

function readOpen(): boolean {
  try {
    return localStorage.getItem(STORAGE_KEY) === '1'
  } catch {
    return false
  }
}

/**
 * Legend of every symbol, colour and field on the displays, so the crew does not
 * have to remember the coding (challenge: "do not assume astronauts know
 * everything"). Collapsible, in the same place under each display title, never
 * covering data (NASA-STD-3001 Vol. 2 App. F F.4.2, F.5.3). Colour meanings follow
 * App. F Table F.5-1.
 */
export function Legend({ sections }: Props) {
  const [open, setOpen] = useState(readOpen)
  const toggle = () => {
    setOpen((v) => {
      try {
        localStorage.setItem(STORAGE_KEY, v ? '0' : '1')
      } catch {
        // storage unavailable: the legend still works for this view
      }
      return !v
    })
  }

  return (
    <section aria-labelledby="legend-title" className="rounded-lg border-[1.5px] border-line bg-surface">
      <h2 id="legend-title" className="m-0">
        <button
          type="button"
          aria-expanded={open}
          aria-controls="legend-body"
          onClick={toggle}
          className="flex w-full items-center justify-between gap-4 px-4 py-3 text-left text-base font-semibold"
        >
          <span>
            Legend <span className="font-normal text-ink-muted">— what the colours, symbols and numbers mean</span>
          </span>
          <span className="font-mono text-sm text-ink-muted">{open ? 'Hide' : 'Show'}</span>
        </button>
      </h2>
      {open && (
        <div id="legend-body" className="grid gap-6 border-t border-line px-4 py-4 md:grid-cols-2 xl:grid-cols-3">
          {sections.map((s) => (
            <div key={s}>{SECTIONS[s]}</div>
          ))}
          <p className="text-sm leading-5 text-ink-muted md:col-span-2 xl:col-span-3">
            Colour meanings follow NASA-STD-3001 Vol. 2 Appendix F. Colour is never the only cue: every class also has its own shape and word.
          </p>
        </div>
      )}
    </section>
  )
}

function Block({ title, children }: { title: string; children: ReactNode }) {
  return (
    <div className="flex flex-col gap-2">
      <h3 className="text-sm leading-5 font-semibold text-ink-muted">{title}</h3>
      <dl className="grid grid-cols-[max-content_1fr] items-center gap-x-3 gap-y-2 text-sm leading-5">{children}</dl>
    </div>
  )
}

function Row({ sample, children }: { sample: ReactNode; children: ReactNode }) {
  return (
    <>
      <dt className="flex items-center">{sample}</dt>
      <dd className="m-0">{children}</dd>
    </>
  )
}

const badge = (a: AlertClass) => <StatusBadge alert={a} />
const mono = (text: ReactNode, cls?: string) => <span className={clsx('font-mono font-medium', cls)}>{text}</span>

/** A small tile outline sample. */
function Swatch({ className }: { className: string }) {
  return <span aria-hidden="true" className={clsx('inline-block h-6 w-10 rounded-sm border-[1.5px]', className)} />
}

const SECTIONS: Record<LegendSection, ReactNode> = {
  classes: (
    <Block title="Alert classes">
      <Row sample={badge('warning')}>Emergency or warning: act now; may need the flight surgeon.</Row>
      <Row sample={badge('caution')}>Off-nominal: needs attention today.</Row>
      <Row sample={badge('advisory')}>Information or a tip; no action required.</Row>
      <Row sample={badge('stale')}>No fresh reading — the last one is older than its freshness window (usually 24 h). Measure again.</Row>
      <Row sample={badge('nominal')}>Within the nominal range. Nominal values are not coloured.</Row>
      <Row sample={<span className="text-sm font-semibold text-unavailable">Unavailable</span>}>Not monitored in this prototype, or a control you cannot use now.</Row>
    </Block>
  ),
  values: (
    <Block title="Values">
      <Row sample={mono(<>118 <span className="font-normal text-ink-muted">mmHg</span></>)}>Plain value: nominal.</Row>
      <Row sample={mono(<>165 <span className="font-normal text-ink-muted">mmHg</span> <span className="text-warning-ink">↑</span></>)}>Above the nominal range; arrow colour = class.</Row>
      <Row sample={mono(<>6.5 <span className="font-normal text-ink-muted">h</span> <span className="text-caution-ink">↓</span></>)}>Below the nominal range.</Row>
      <Row sample={mono('36.7 °C', 'text-stale-ink')}>Cyan value: stale; the time of that reading is shown next to it.</Row>
      <Row sample={mono('3 /5')}>Self-report scale: score out of 5.</Row>
      <Row sample={mono('90–129 mmHg', 'text-ink-muted')}>Nominal range. Ranges are illustrative, not medical limits.</Row>
    </Block>
  ),
  tiles: (
    <Block title="Body system tiles">
      <Row sample={<Swatch className="border-line-strong bg-surface-raised" />}>Nominal.</Row>
      <Row sample={<Swatch className="border-caution-ink bg-surface-raised" />}>Caution: yellow border.</Row>
      <Row sample={<Swatch className="border-warning bg-warning" />}>Warning: the whole tile turns red — only for what needs immediate attention.</Row>
      <Row sample={<Swatch className="border-dashed border-stale-ink bg-surface-raised" />}>Stale data: dashed cyan border.</Row>
      <Row sample={<Swatch className="border-dashed border-unavailable" />}>Unavailable: dashed gray border.</Row>
      <Row sample={mono('SYS-01', 'text-ink-muted')}>System code; select a tile to open it on the body map.</Row>
    </Block>
  ),
  zones: (
    <Block title="Body map">
      <Row sample={<Swatch className="border-ink bg-surface-raised" />}>Nominal zone: outline only.</Row>
      <Row sample={<Swatch className="border-caution-ink bg-caution/40" />}>Caution zone.</Row>
      <Row sample={<Swatch className="border-warning bg-warning" />}>Warning zone.</Row>
      <Row sample={<Swatch className="border-dashed border-unavailable" />}>Not monitored in this prototype.</Row>
      <Row sample={<Swatch className="border-[3px] border-navigation bg-surface-raised" />}>Green outline: the selected system.</Row>
      <Row sample={<span className="font-mono text-sm">Tab · Enter</span>}>Keyboard: move between zones and open one.</Row>
    </Block>
  ),
  crew: (
    <Block title="Crew display">
      <Row sample={<Swatch className="border-warning-ink bg-surface-raised" />}>Member card: red border if their readiness is Warning, yellow if Caution.</Row>
      <Row sample={mono(<><span className="text-warning-ink">W 1</span> · <span className="text-caution-ink">C 2</span></>)}>That member's active warnings and cautions.</Row>
      <Row
        sample={
          <svg viewBox="0 0 56 20" className="h-5 w-14" aria-hidden="true">
            <line x1="0" y1="5" x2="56" y2="5" className="stroke-line-strong" strokeWidth="1" strokeDasharray="3 3" />
            <line x1="0" y1="15" x2="56" y2="15" className="stroke-line-strong" strokeWidth="1" strokeDasharray="3 3" />
            <polyline points="4,12 16,10 28,9 40,4 52,2" fill="none" className="stroke-ink-muted" strokeWidth="1.5" />
            <path d="M36 7 44 7 40 0Z" className="fill-caution" />
            <circle cx="52" cy="2" r="2.5" className="fill-ink" />
          </svg>
        }
      >
        7-day history: dashed lines are the nominal limits, the dot is the latest reading; triangles and diamonds mark Caution and Warning readings.
      </Row>
      <Row sample={mono('9/9')}>Fresh data: indicators with a reading inside their freshness window.</Row>
    </Block>
  ),
  vehicle: (
    <Block title="Vehicle">
      <Row sample={mono(<>3.8 <span className="font-normal text-ink-muted">mmHg</span> <span className="text-caution-ink">↑</span></>)}>Life-support reading above its nominal limit; same classes and arrows as health data.</Row>
      <Row sample={<Swatch className="border-caution-ink bg-surface-raised" />}>Module border: its worst parameter. Everyone in that module gets the advice.</Row>
      <Row sample={mono('22.5 °C', 'text-stale-ink')}>No telemetry for over 4 hours: Stale.</Row>
      <Row sample={mono('ppCO₂')}>Carbon dioxide partial pressure; above 3 mmHg it causes headaches and poor sleep.</Row>
      <Row sample={mono('µGy/h')}>Radiation dose rate from the module dosimeter.</Row>
    </Block>
  ),
  supplies: (
    <Block title="Med supplies">
      <Row sample={badge('warning')}>Out of stock or expired.</Row>
      <Row sample={badge('caution')}>Below minimum, expires within 30 days, count differs from the record, or runs out before the mission ends.</Row>
      <Row sample={badge('advisory')}>Inventory count due (not counted for 30 days).</Row>
      <Row sample={mono('4 strip/day')}>Average use over the last 30 days, the basis of the forecast.</Row>
      <Row sample={<span className="text-sm font-semibold">Dispose</span>}>Write off expired stock; asks for confirmation and cannot be undone.</Row>
    </Block>
  ),
  tasks: (
    <Block title="Today">
      <Row sample={mono('☑')}>Done: reading tasks tick themselves when the readings are logged.</Row>
      <Row sample={mono('☐')}>Still to do; manual tasks such as exercise have a «Mark done» button.</Row>
      <Row sample={badge('advisory')}>Overdue: a reminder, not a health alert.</Row>
      <Row sample={mono('streak 12 days')}>Days in a row with every daily task done; today does not break it until the day is over.</Row>
    </Block>
  ),
  profile: (
    <Block title="Profile">
      <Row sample={mono('42–95 bpm')}>Personal range set by the flight surgeon; it replaces the standard range for this member.</Row>
      <Row sample={<span className="text-xs font-semibold">personal</span>}>Marks a reading judged against a personal range.</Row>
      <Row sample={mono('baseline 56')}>Preflight value, for comparison.</Row>
      <Row sample={mono('1.5')}>Weight of a body system in this member's Readiness Score (standard 1).</Row>
    </Block>
  ),
  score: (
    <Block title="Readiness score">
      <Row sample={mono('80–100')}>Nominal — fit for planned duties.</Row>
      <Row sample={mono('50–79')}>Caution — some systems need attention.</Row>
      <Row sample={mono('0–49')}>Warning — at least one critical reading.</Row>
      <Row sample={mono('49', 'text-warning-ink')}>Any warning caps the score at 49, so one critical value is never hidden by good averages.</Row>
      <Row sample={mono('9/9')}>Fresh data: indicators with a reading inside their freshness window.</Row>
    </Block>
  ),
  recommendations: (
    <Block title="Recommendations">
      <Row sample={<span className="text-sm font-semibold">Basis</span>}>The readings and limits that triggered the advice.</Row>
      <Row sample={<span className="text-sm font-semibold">Re-check</span>}>When to measure again.</Row>
      <Row
        sample={
          <span className="inline-flex h-6 items-center rounded-sm border-[1.5px] border-warning-ink px-2 text-sm font-semibold text-warning-ink">Report</span>
        }
      >
        Report to the flight surgeon. «Earth reply» is the one-way signal delay — act on the advice meanwhile.
      </Row>
      <Row sample={<span className="inline-flex h-6 items-center rounded-md border-[1.5px] border-command-edge bg-command px-2 text-sm font-semibold text-on-command">Acknowledge</span>}>
        Mark as seen; it moves to the end of the list.
      </Row>
    </Block>
  ),
  statusbar: (
    <Block title="Top bar">
      <Row sample={mono('MET 209/13:05:12')}>Mission elapsed time: days/hours:minutes:seconds since launch.</Row>
      <Row sample={mono('UTC 13:05:12')}>Coordinated Universal Time, 24-hour clock.</Row>
      <Row sample={mono(<><span className="text-warning-ink">W 1</span> · <span className="text-caution-ink">C 2</span></>)}>Active warnings and cautions for the selected crew member.</Row>
      <Row sample={mono('AOS')}>Link to the AURA server is up (acquisition of signal).</Row>
      <Row sample={mono('LOS', 'text-warning-ink')}>Link lost (loss of signal): data may be out of date.</Row>
      <Row sample={mono('3–22 min')}>«Earth»: one-way signal delay to Earth in the current mission phase.</Row>
    </Block>
  ),
}
