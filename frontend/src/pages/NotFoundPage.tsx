import { Link } from 'react-router'
import { Notice } from '../shared/ui/Notice.tsx'

function BackLink() {
  return (
    <p>
      <Link to="/" className="font-semibold underline decoration-line-strong underline-offset-4 hover:decoration-ink">
        Go to Readiness
      </Link>
    </p>
  )
}

/** Unknown address, shown inside the app shell. */
export function NotFoundPage() {
  return (
    <section className="flex max-w-2xl flex-col gap-4">
      <h1 className="text-[40px] leading-[48px] font-semibold tracking-[-0.01em]">Page not found</h1>
      <Notice alert="advisory">There is no AURA display at this address.</Notice>
      <BackLink />
    </section>
  )
}

/** A display that failed to render, shown inside the app shell (route errorElement). */
export function RouteErrorPage() {
  return (
    <section className="flex max-w-2xl flex-col gap-4">
      <h1 className="text-[40px] leading-[48px] font-semibold tracking-[-0.01em]">Something went wrong</h1>
      <Notice alert="warning">This display failed to load. Your readings are safe on the AURA server.</Notice>
      <BackLink />
    </section>
  )
}
