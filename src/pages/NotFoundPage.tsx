import { isRouteErrorResponse, Link, useRouteError } from 'react-router'
import { Notice } from '../shared/ui/Notice.tsx'

/** Unknown address or a route error, shown inside the app shell. */
export function NotFoundPage() {
  const error = useRouteError()
  const isError = error !== undefined && !(isRouteErrorResponse(error) && error.status === 404)
  return (
    <section className="flex max-w-2xl flex-col gap-4">
      <h1 className="text-[40px] leading-[48px] font-semibold tracking-[-0.01em]">{isError ? 'Something went wrong' : 'Page not found'}</h1>
      <Notice alert={isError ? 'warning' : 'advisory'}>
        {isError ? 'This display failed to load. Your readings are safe on the AURA server.' : 'There is no AURA display at this address.'}
      </Notice>
      <p>
        <Link to="/" className="font-semibold underline decoration-line-strong underline-offset-4 hover:decoration-ink">
          Go to Readiness
        </Link>
      </p>
    </section>
  )
}
