import { useState, type FormEvent } from 'react'
import { roleLabel } from '../features/crew/api.ts'
import { ApiError } from '../shared/api/client.ts'
import { useMission } from '../shared/api/mission.ts'
import { formatMET, formatUTC } from '../shared/lib/time.ts'
import { useNow } from '../shared/lib/useNow.ts'
import { Button } from '../shared/ui/Button.tsx'
import { Notice } from '../shared/ui/Notice.tsx'
import { useDemoAccounts, useLogin, type DemoAccounts, type Session } from '../features/auth/api.ts'

/** What each role can open, shown next to the demo accounts (FR-72). */
const ACCESS: Record<Session['user']['role'], string> = {
  crew: 'Own readings, recommendations and report',
  commander: 'Whole crew display; logs own readings',
  deputy_commander: 'Whole crew display; logs own readings',
  flight_surgeon: 'Crew dashboards and reports (on Earth, no own readings)',
}

type Props = { onSignedIn: (s: Session) => void }

/** Sign-in display (FR-71): username and password, plus one-click demo accounts. */
export function LoginPage({ onSignedIn }: Props) {
  const login = useLogin()
  const demo = useDemoAccounts()
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')

  const signIn = (u: string, p: string) => login.mutate({ username: u, password: p }, { onSuccess: onSignedIn })
  const submit = (e: FormEvent) => {
    e.preventDefault()
    signIn(username, password)
  }
  const error =
    login.error instanceof ApiError && login.error.status === 401
      ? 'Wrong username or password.'
      : login.error
        ? 'Cannot reach the AURA server. Check the connection and try again.'
        : null

  return (
    <div className="flex h-full flex-col">
      <LoginHeader />
      <main className="flex-1 overflow-auto p-8">
        <div className="mx-auto flex max-w-[1120px] flex-col gap-8">
          <header>
            <h1 className="text-[40px] leading-[48px] font-semibold tracking-[-0.01em]">Sign in</h1>
            <p className="text-ink-muted">Astronaut Universal Readiness Assessment — each crew member sees their own health data; the commander and the flight surgeon see the whole crew.</p>
          </header>

          <div className="grid items-start gap-8 lg:grid-cols-[360px_1fr]">
            <form onSubmit={submit} className="flex flex-col gap-4 rounded-lg border-[1.5px] border-line bg-surface-raised p-6" aria-labelledby="signin-title">
              <h2 id="signin-title" className="text-2xl leading-8 font-semibold">
                Account
              </h2>
              <label htmlFor="username" className="flex flex-col gap-1">
                <span className="text-sm leading-5 font-medium text-ink-muted">Username</span>
                <input
                  id="username"
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                  autoComplete="username"
                  autoCapitalize="none"
                  spellCheck={false}
                  required
                  className="h-10 rounded-md border-[1.5px] border-line-strong bg-surface px-3 font-mono text-ink"
                />
              </label>
              <label htmlFor="password" className="flex flex-col gap-1">
                <span className="text-sm leading-5 font-medium text-ink-muted">Password</span>
                <input
                  id="password"
                  type="password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  autoComplete="current-password"
                  required
                  className="h-10 rounded-md border-[1.5px] border-line-strong bg-surface px-3 font-mono text-ink"
                />
              </label>
              {error && <Notice alert="warning">{error}</Notice>}
              <Button type="submit" variant="command" disabled={login.isPending} className="justify-center">
                {login.isPending ? 'Signing in…' : 'Sign in'}
              </Button>
            </form>

            {demo.data && <DemoAccountList demo={demo.data} pending={login.isPending} onPick={(u) => signIn(u, demo.data!.password)} />}
          </div>
        </div>
      </main>
      <footer className="flex-none border-t border-line px-4 py-2 text-sm leading-5 text-ink-muted">
        Prototype for NASA Space Apps Challenge 2026 — not a medical device, not endorsed by NASA.
      </footer>
    </div>
  )
}

function DemoAccountList({ demo, pending, onPick }: { demo: DemoAccounts; pending: boolean; onPick: (username: string) => void }) {
  return (
    <section aria-labelledby="demo-title" className="flex flex-col gap-3">
      <h2 id="demo-title" className="text-2xl leading-8 font-semibold">
        Demo accounts
      </h2>
      <p className="text-ink-muted">
        This is a public demo: every account uses the password <span className="font-mono text-ink">{demo.password}</span>.
      </p>
      <div className="overflow-x-auto rounded-lg border-[1.5px] border-line bg-surface-raised">
        <table className="w-full text-left">
          <thead className="text-sm leading-5 text-ink-muted">
            <tr className="border-b border-line">
              <th className="px-4 py-2 font-medium">Name</th>
              <th className="px-4 py-2 font-medium">Role</th>
              <th className="px-4 py-2 font-medium">Username</th>
              <th className="px-4 py-2 font-medium">Can open</th>
              <th className="px-4 py-2 font-medium">
                <span className="sr-only">Action</span>
              </th>
            </tr>
          </thead>
          <tbody>
            {demo.accounts.map((a) => (
              <tr key={a.username} className="border-b border-line last:border-0">
                <td className="px-4 py-2 font-semibold whitespace-nowrap">{a.name}</td>
                <td className="px-4 py-2">{roleLabel[a.role]}</td>
                <td className="px-4 py-2 font-mono text-sm">{a.username}</td>
                <td className="px-4 py-2 text-sm text-ink-muted">{ACCESS[a.role]}</td>
                <td className="px-4 py-2 text-right">
                  <Button disabled={pending} onClick={() => onPick(a.username)} className="h-8 px-3 text-sm whitespace-nowrap" aria-label={`Sign in as ${a.name}`}>
                    Sign in
                  </Button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  )
}

/** Wordmark and mission clock: the status area stays visible before sign-in too (App. F F.4.2). */
function LoginHeader() {
  const now = useNow()
  const mission = useMission()
  return (
    <header className="flex min-h-12 flex-none flex-wrap items-center gap-x-4 gap-y-2 border-b border-line bg-surface px-4 py-2">
      <span className="text-lg font-bold tracking-[0.04em]">AURA</span>
      {mission.data && <span className="text-sm text-ink-muted">{mission.data.name}</span>}
      <div className="ml-auto flex gap-3 font-mono text-sm leading-5 whitespace-nowrap">
        <span>
          <span className="font-sans font-medium text-ink-muted">MET </span>
          {mission.data ? formatMET(new Date(mission.data.startedAt), now) : '—'}
        </span>
        <span>
          <span className="font-sans font-medium text-ink-muted">UTC </span>
          {formatUTC(now)}
        </span>
      </div>
    </header>
  )
}
