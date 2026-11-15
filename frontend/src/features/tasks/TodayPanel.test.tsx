import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { Catalog } from '../../shared/api/catalog.ts'
import { TodayPanel } from './components/TodayPanel.tsx'

const task = (code: string, title: string, extra: object) => ({
  code,
  title,
  every: 'daily',
  manual: false,
  indicators: [],
  missing: [],
  due: '2026-11-15T09:00:00Z',
  done: false,
  doneAt: null,
  overdue: false,
  ...extra,
})

const tasks = {
  day: '2026-11-15',
  tasks: [
    task('morning_vitals', 'Morning vitals', { done: true, doneAt: '2026-11-15T07:00:00Z' }),
    task('morning_checkin', 'Morning check-in', { overdue: true, indicators: ['mood', 'stress'], missing: ['stress'] }),
    task('exercise', 'Exercise session (2 h)', { manual: true, due: '2026-11-15T20:00:00Z' }),
  ],
  done: 1,
  total: 3,
  overdue: 1,
  streak: 12,
  bestStreak: 20,
  badges: [{ code: 'streak_7', title: 'Week of readiness', description: '7 days', earnedAt: '2026-11-01T00:00:00Z' }],
}

const catalog = { indicators: [{ code: 'stress', shortName: 'Stress' }] } as unknown as Catalog

describe('TodayPanel', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('shows progress, the overdue reminder and marks a manual task', async () => {
    const fetchMock = vi.fn((_url: string, init?: RequestInit) =>
      Promise.resolve(
        new Response(JSON.stringify(init?.method === 'POST' ? { ...tasks, done: 2 } : tasks), { status: 200, headers: { 'Content-Type': 'application/json' } }),
      ),
    )
    vi.stubGlobal('fetch', fetchMock)
    render(
      <QueryClientProvider client={new QueryClient()}>
        <MemoryRouter>
          <TodayPanel astronautId={4} canMark catalog={catalog} />
        </MemoryRouter>
      </QueryClientProvider>,
    )

    expect(await screen.findByText(/1\/3 done/)).toBeInTheDocument()
    expect(screen.getByText(/1 overdue/)).toBeInTheDocument()
    expect(screen.getByText('missing: Stress')).toBeInTheDocument()
    expect(screen.getByText('Week of readiness')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Mark done' }))
    await waitFor(() => expect(fetchMock).toHaveBeenCalledWith('/api/v1/astronauts/4/tasks/exercise/complete', expect.objectContaining({ method: 'POST' })))
    expect(await screen.findByText(/2\/3 done/)).toBeInTheDocument()
  })
})
