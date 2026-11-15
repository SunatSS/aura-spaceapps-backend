import { roleLabel } from '../api.ts'
import { useCurrentAstronaut } from '../context.ts'

/** Crew member picker in the top bar; only for those who may open other members (FR-72). */
export function AstronautSelect() {
  const { crew, current, select } = useCurrentAstronaut()
  if (crew.length < 2) return null
  return (
    <label className="flex items-center">
      <span className="sr-only">Crew member</span>
      <select
        className="h-8 rounded-md border-[1.5px] border-line-strong bg-surface-raised px-2 text-sm leading-5 font-semibold text-ink"
        value={current?.id}
        onChange={(e) => select(Number(e.target.value))}
        title={current ? `${current.name} — ${roleLabel[current.role]}` : undefined}
      >
        {crew.map((a) => (
          <option key={a.id} value={a.id}>
            {a.name}
          </option>
        ))}
      </select>
    </label>
  )
}
