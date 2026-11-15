import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { LoginPage } from './LoginPage.tsx'

const session = { user: { id: 4, missionId: 1, name: 'Amina Rahimova', role: 'crew' }, username: 'rahimova' }
const demo = { password: 'aura-mx1', accounts: [{ username: 'rahimova', name: 'Amina Rahimova', role: 'crew' }] }

function json(status: number, body: unknown) {
  return Promise.resolve(new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } }))
}

function renderLogin(onSignedIn = vi.fn()) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  render(
    <QueryClientProvider client={qc}>
      <LoginPage onSignedIn={onSignedIn} />
    </QueryClientProvider>,
  )
  return onSignedIn
}

describe('LoginPage', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('signs in with a demo account in one click', async () => {
    const fetchMock = vi.fn((url: string, init?: RequestInit) => {
      if (url.endsWith('/auth/demo-accounts')) return json(200, demo)
      if (url.endsWith('/auth/login')) return json(200, session)
      if (url.endsWith('/missions/current')) return json(404, { error: 'no mission yet' })
      throw new Error(`unexpected ${init?.method ?? 'GET'} ${url}`)
    })
    vi.stubGlobal('fetch', fetchMock)
    const onSignedIn = renderLogin()

    fireEvent.click(await screen.findByRole('button', { name: 'Sign in as Amina Rahimova' }))
    await waitFor(() => expect(onSignedIn).toHaveBeenCalled())
    expect(onSignedIn.mock.calls[0][0]).toEqual(session)
    const login = fetchMock.mock.calls.find(([url]) => url.endsWith('/auth/login'))!
    expect(JSON.parse(login[1]!.body as string)).toEqual({ username: 'rahimova', password: 'aura-mx1' })
  })

  it('says so when the password is wrong', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn((url: string) => (url.endsWith('/auth/login') ? json(401, { error: 'wrong username or password' }) : json(404, { error: 'none' }))),
    )
    const onSignedIn = renderLogin()

    fireEvent.change(screen.getByLabelText('Username'), { target: { value: 'rahimova' } })
    fireEvent.change(screen.getByLabelText('Password'), { target: { value: 'nope' } })
    fireEvent.click(screen.getByRole('button', { name: 'Sign in' }))
    expect(await screen.findByText('Wrong username or password.')).toBeInTheDocument()
    expect(onSignedIn).not.toHaveBeenCalled()
  })
})
