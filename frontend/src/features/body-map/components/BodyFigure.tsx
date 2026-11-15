import clsx from 'clsx'
import type { KeyboardEvent, ReactNode } from 'react'
import { alertLabel, type AlertClass } from '../../../shared/lib/status.ts'

export type ZoneState = AlertClass | 'unavailable'

export type Zone = {
  code: string // body system code from the API
  sysCode: string // SYS-01 …
  name: string
  state: ZoneState
}

type Props = {
  zones: Zone[]
  selected: string | undefined
  onSelect: (code: string) => void
}

/** Where each system's label sits and which point of the figure its leader line touches. */
const LAYOUT: Record<string, { side: 'left' | 'right'; y: number; target: [number, number] }> = {
  behavioral: { side: 'left', y: 52, target: [278, 70] },
  respiratory: { side: 'left', y: 168, target: [274, 186] },
  metabolic: { side: 'left', y: 268, target: [284, 282] },
  musculoskeletal: { side: 'left', y: 436, target: [274, 452] },
  eyes: { side: 'right', y: 36, target: [342, 66] },
  immune: { side: 'right', y: 112, target: [372, 134] },
  cardiovascular: { side: 'right', y: 204, target: [338, 214] },
}

const GLYPH: Record<ZoneState, string> = {
  nominal: '○',
  caution: '▲',
  warning: '◆',
  advisory: '■',
  stale: '◌',
  unavailable: '',
}

const stateLabel = (s: ZoneState) => (s === 'unavailable' ? 'Unavailable' : alertLabel[s])

/** Fill and stroke per state: colour only where it means something (NASA App. F F.5.10). */
function zoneClass(state: ZoneState, selected: boolean): string {
  const fill: Record<ZoneState, string> = {
    nominal: 'fill-surface-raised',
    caution: 'fill-caution/40',
    warning: 'fill-warning',
    advisory: 'fill-advisory/40',
    stale: 'fill-surface-raised',
    unavailable: 'fill-transparent',
  }
  const stroke: Record<ZoneState, string> = {
    nominal: 'stroke-ink',
    caution: 'stroke-caution-ink',
    warning: 'stroke-warning',
    advisory: 'stroke-advisory-ink',
    stale: 'stroke-stale-ink',
    unavailable: 'stroke-unavailable',
  }
  return clsx(
    fill[state],
    selected ? 'stroke-navigation' : stroke[state],
    (state === 'stale' || state === 'unavailable') && !selected && '[stroke-dasharray:6_4]',
    'cursor-pointer transition-colors',
  )
}

const labelInk: Record<ZoneState, string> = {
  nominal: 'fill-ink',
  caution: 'fill-caution-ink',
  warning: 'fill-warning-ink',
  advisory: 'fill-advisory-ink',
  stale: 'fill-stale-ink',
  unavailable: 'fill-unavailable',
}

/**
 * Schematic human figure: each body system is a zone coloured by its alert class
 * and labelled with code, name and status word (schematics must label every
 * object, App. F F.5.9). Zones and labels are both buttons.
 */
export function BodyFigure({ zones, selected, onSelect }: Props) {
  const byCode = Object.fromEntries(zones.map((z) => [z.code, z]))

  const zone = (code: string, children: ReactNode) => {
    const z = byCode[code]
    if (!z) return null
    const isSel = selected === code
    return (
      <g
        role="button"
        tabIndex={0}
        aria-pressed={isSel}
        aria-label={`${z.sysCode} ${z.name}: ${stateLabel(z.state)}`}
        className={zoneClass(z.state, isSel)}
        strokeWidth={isSel ? 3 : 1.5}
        onClick={() => onSelect(code)}
        onKeyDown={(e) => activate(e, () => onSelect(code))}
      >
        {children}
      </g>
    )
  }

  return (
    <svg viewBox="0 0 640 580" className="w-full max-w-[640px]" role="group" aria-label="Body map">
      {/* Draw order: trunk, limbs, head, then organs on top. */}
      {zone('immune', <path d="M308 110h24v18h18a28 28 0 0 1 28 28v144a28 28 0 0 1-28 28h-60a28 28 0 0 1-28-28V156a28 28 0 0 1 28-28h18z" />)}
      {zone(
        'musculoskeletal',
        <>
          <rect x="222" y="140" width="30" height="172" rx="15" />
          <rect x="388" y="140" width="30" height="172" rx="15" />
          <rect x="272" y="332" width="40" height="224" rx="18" />
          <rect x="328" y="332" width="40" height="224" rx="18" />
        </>,
      )}
      {zone('behavioral', <circle cx="320" cy="70" r="42" />)}
      {zone(
        'eyes',
        <>
          <ellipse cx="305" cy="66" rx="9" ry="5.5" />
          <ellipse cx="335" cy="66" rx="9" ry="5.5" />
        </>,
      )}
      {zone(
        'respiratory',
        <>
          <ellipse cx="294" cy="186" rx="22" ry="44" />
          <ellipse cx="346" cy="186" rx="22" ry="44" />
        </>,
      )}
      {zone('metabolic', <ellipse cx="320" cy="282" rx="38" ry="26" />)}
      {zone('cardiovascular', <path d="M325 240c-25-16-27-40-12-44 7-2 12 4 12 8 0-4 5-10 12-8 15 4 13 28-12 44z" />)}

      {zones.map((z) => {
        const l = LAYOUT[z.code]
        if (!l) return null
        const left = l.side === 'left'
        const x = left ? 182 : 458
        const isSel = selected === z.code
        return (
          <g
            key={z.code}
            role="button"
            tabIndex={-1}
            aria-hidden="true"
            className="cursor-pointer"
            onClick={() => onSelect(z.code)}
          >
            <line
              x1={left ? 188 : 452}
              y1={l.y + 14}
              x2={l.target[0]}
              y2={l.target[1]}
              className={isSel ? 'stroke-navigation' : 'stroke-line-strong'}
              strokeWidth={isSel ? 2 : 1.5}
            />
            <rect x={left ? 8 : 452} y={l.y - 14} width={180} height={58} className="fill-transparent" />
            <text x={x} y={l.y} textAnchor={left ? 'end' : 'start'} className="fill-ink-muted font-mono text-sm">
              {z.sysCode}
            </text>
            <text x={x} y={l.y + 18} textAnchor={left ? 'end' : 'start'} className="fill-ink text-sm font-semibold">
              {z.name}
            </text>
            <text x={x} y={l.y + 36} textAnchor={left ? 'end' : 'start'} className={clsx('text-sm font-semibold', labelInk[z.state])}>
              {GLYPH[z.state] && `${GLYPH[z.state]} `}
              {stateLabel(z.state)}
            </text>
          </g>
        )
      })}
    </svg>
  )
}

function activate(e: KeyboardEvent, fn: () => void) {
  if (e.key === 'Enter' || e.key === ' ') {
    e.preventDefault()
    fn()
  }
}
