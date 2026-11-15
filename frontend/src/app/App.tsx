import { MutationCache, QueryCache, QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { RouterProvider } from 'react-router/dom'
import { SESSION_KEY, seesWholeCrew, useSession, type Session } from '../features/auth/api.ts'
import { LoginPage } from '../pages/LoginPage.tsx'
import { CurrentAstronautProvider } from '../features/crew/CurrentAstronaut.tsx'
import { ApiError } from '../shared/api/client.ts'
import { Button } from '../shared/ui/Button.tsx'
import { Notice } from '../shared/ui/Notice.tsx'
import { router } from './router.tsx'

/** Any 401 means the session is gone (sign-out elsewhere, server restart): show sign-in. */
function onUnauthorized(error: unknown) {
  if (error instanceof ApiError && error.status === 401) queryClient.setQueryData(SESSION_KEY, null)
}

const queryClient = new QueryClient({
  queryCache: new QueryCache({ onError: onUnauthorized }),
  mutationCache: new MutationCache({ onError: onUnauthorized }),
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      retry: (count, error) => !(error instanceof ApiError && error.status < 500) && count < 1,
    },
  },
})

/** Landing display after sign-in: the crew display for those who see the whole crew. */
function landOn(s: Session) {
  void router.navigate(seesWholeCrew(s.user.role) ? '/crew' : '/')
}

export function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <AuthGate />
    </QueryClientProvider>
  )
}

function AuthGate() {
  const session = useSession()
  if (session.isPending) return <Splash>Connecting to the AURA server…</Splash>
  if (session.isError)
    return (
      <Splash>
        <Notice alert="warning">Cannot reach the AURA server. The free host may be waking up — try again in a minute.</Notice>
        <Button onClick={() => void session.refetch()}>Try again</Button>
      </Splash>
    )
  if (!session.data) return <LoginPage onSignedIn={landOn} />
  return (
    <CurrentAstronautProvider me={session.data.user}>
      <RouterProvider router={router} />
    </CurrentAstronautProvider>
  )
}

function Splash({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex h-full flex-col items-center justify-center gap-4 p-8 text-ink-muted">
      <span className="text-lg font-bold tracking-[0.04em] text-ink">AURA</span>
      {children}
    </div>
  )
}
