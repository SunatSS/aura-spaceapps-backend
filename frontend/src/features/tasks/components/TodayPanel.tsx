import clsx from 'clsx'
import { Link } from 'react-router'
import type { Catalog } from '../../../shared/api/catalog.ts'
import { formatMET } from '../../../shared/lib/time.ts'
import { Button } from '../../../shared/ui/Button.tsx'
import { StatusBadge } from '../../../shared/ui/StatusBadge.tsx'
import { useTasks, useToggleTask, type Task } from '../api.ts'

type Props = { astronautId: number; canMark: boolean; catalog: Catalog; missionStart?: Date }

/**
 * Today's schedule (FR-40…FR-43): reading tasks tick themselves when the
 * readings are logged; manual tasks are marked here. Overdue items are
 * reminders (Advisory), not health alerts. Streak and badges are plain text.
 */
export function TodayPanel({ astronautId, canMark, catalog, missionStart }: Props) {
  const tasks = useTasks(astronautId)
  const toggle = useToggleTask(astronautId)
  if (!tasks.data) return null
  const t = tasks.data
  const name = (code: string) => catalog.indicators.find((i) => i.code === code)?.shortName ?? code
  const time = (iso: string) => (missionStart ? `MET ${formatMET(missionStart, new Date(iso)).slice(0, -3)}` : iso.slice(11, 16))

  return (
    <section aria-labelledby="today-title" className="flex flex-col gap-4 rounded-lg border-[1.5px] border-line bg-surface-raised p-6">
      <div className="flex flex-wrap items-baseline justify-between gap-3">
        <h2 id="today-title" className="text-2xl leading-8 font-semibold">
          Today
        </h2>
        <p className="font-mono text-sm">
          {t.done}/{t.total} done
          {t.overdue > 0 && <span className="text-advisory-ink"> · {t.overdue} overdue</span>}
          <span className="text-ink-muted"> · streak </span>
          {t.streak} {t.streak === 1 ? 'day' : 'days'}
          <span className="text-ink-muted"> (best {t.bestStreak})</span>
        </p>
      </div>
      <ul className="flex flex-col">
        {t.tasks.map((task) => (
          <TaskRow key={task.code} task={task} name={name} time={time} canMark={canMark} busy={toggle.isPending} onToggle={(done) => toggle.mutate({ code: task.code, done })} />
        ))}
      </ul>
      {t.badges.length > 0 && (
        <div className="flex flex-wrap items-center gap-2 border-t border-line pt-3">
          <span className="text-sm font-medium text-ink-muted">Badges</span>
          {t.badges.map((b) => (
            <span key={b.code} title={b.description} className="inline-flex h-6 items-center rounded-sm border-[1.5px] border-line-strong px-2 text-sm font-semibold">
              {b.title}
            </span>
          ))}
        </div>
      )}
    </section>
  )
}

function TaskRow({
  task,
  name,
  time,
  canMark,
  busy,
  onToggle,
}: {
  task: Task
  name: (code: string) => string
  time: (iso: string) => string
  canMark: boolean
  busy: boolean
  onToggle: (done: boolean) => void
}) {
  return (
    <li className="flex flex-wrap items-center gap-x-3 gap-y-1 border-b border-line py-2 last:border-0">
      <span aria-hidden="true" className={clsx('font-mono text-lg leading-none', task.done ? 'text-ink' : 'text-ink-muted')}>
        {task.done ? '☑' : '☐'}
      </span>
      <span className={clsx('font-semibold', task.done && 'text-ink-muted')}>{task.title}</span>
      {task.every === 'weekly' && <span className="text-sm text-ink-muted">weekly</span>}
      <span className="font-mono text-sm text-ink-muted">
        {task.done && task.doneAt ? `done ${time(task.doneAt)}` : `due ${time(task.due)}`}
      </span>
      {task.overdue && <StatusBadge alert="advisory" quiet />}
      {!task.done && task.missing.length > 0 && task.missing.length < task.indicators.length && (
        <span className="text-sm text-ink-muted">missing: {task.missing.map(name).join(', ')}</span>
      )}
      <span className="ml-auto">
        {task.manual && canMark ? (
          <Button className="h-8 px-3 text-sm" disabled={busy} onClick={() => onToggle(!task.done)}>
            {task.done ? 'Undo' : 'Mark done'}
          </Button>
        ) : (
          !task.done &&
          canMark && (
            <Link to="/log" className="text-sm font-semibold underline underline-offset-2">
              Log
            </Link>
          )
        )}
      </span>
    </li>
  )
}
