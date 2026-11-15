import { useState } from 'react'
import { useDeletePasskey, usePasskeys, useRegisterPasskey } from '../features/auth/api.ts'
import { passkeyError, passkeysSupported } from '../features/auth/passkeys.ts'
import { roleLabel } from '../features/crew/api.ts'
import { useCurrentAstronaut } from '../features/crew/context.ts'
import { ApiError } from '../shared/api/client.ts'
import { Button } from '../shared/ui/Button.tsx'
import { Notice } from '../shared/ui/Notice.tsx'

/** A default name for the passkey from the device the member is on. */
function deviceLabel(): string {
  const ua = navigator.userAgent
  if (/Windows/.test(ua)) return 'Windows Hello'
  if (/iPhone|iPad/.test(ua)) return 'Face ID / Touch ID'
  if (/Mac/.test(ua)) return 'Touch ID'
  if (/Android/.test(ua)) return 'Android'
  return 'This device'
}

/** Account display: who you are and your passkeys (FR-74). */
export function AccountPage() {
  const { me } = useCurrentAstronaut()
  const passkeys = usePasskeys()
  const register = useRegisterPasskey()
  const remove = useDeletePasskey()
  const [label, setLabel] = useState(deviceLabel)
  const [confirming, setConfirming] = useState<string | null>(null)
  const supported = passkeysSupported()
  const error = register.error instanceof ApiError ? register.error.message : register.error ? passkeyError(register.error) : null

  return (
    <div className="flex max-w-3xl flex-col gap-8">
      <header>
        <h1 className="text-[40px] leading-[48px] font-semibold tracking-[-0.01em]">Account</h1>
        <p className="text-ink-muted">
          {me.name} · {roleLabel[me.role]}
        </p>
      </header>

      <section aria-labelledby="pk-title" className="flex flex-col gap-4 rounded-lg border-[1.5px] border-line bg-surface-raised p-6">
        <h2 id="pk-title" className="text-2xl leading-8 font-semibold">
          Passkeys
        </h2>
        <p>
          Sign in with Windows Hello, Touch ID or Face ID instead of a password. The private key never leaves your device; AURA keeps only the public key.
        </p>

        {!supported && <Notice alert="advisory">This browser or connection does not support passkeys (they need HTTPS).</Notice>}

        {passkeys.data && passkeys.data.length > 0 ? (
          <ul className="flex flex-col gap-2">
            {passkeys.data.map((p) => (
              <li key={p.id} className="flex flex-wrap items-center gap-3 rounded-md border-[1.5px] border-line px-4 py-3">
                <span className="font-semibold">{p.label}</span>
                <span className="font-mono text-sm text-ink-muted">
                  added {p.createdAt.slice(0, 10)}
                  {p.lastUsedAt ? ` · last used ${p.lastUsedAt.slice(0, 16).replace('T', ' ')} UTC` : ' · not used yet'}
                </span>
                <span className="ml-auto flex gap-2">
                  {confirming === p.id ? (
                    <>
                      <span className="self-center text-sm">Remove this passkey?</span>
                      <Button variant="command" className="h-8 px-3 text-sm" disabled={remove.isPending} onClick={() => remove.mutate(p.id, { onSuccess: () => setConfirming(null) })}>
                        Confirm
                      </Button>
                      <Button className="h-8 px-3 text-sm" onClick={() => setConfirming(null)}>
                        Cancel
                      </Button>
                    </>
                  ) : (
                    <Button className="h-8 px-3 text-sm" onClick={() => setConfirming(p.id)}>
                      Remove
                    </Button>
                  )}
                </span>
              </li>
            ))}
          </ul>
        ) : (
          passkeys.data && <p className="text-ink-muted">No passkeys yet.</p>
        )}

        {supported && (
          <form
            className="flex flex-wrap items-end gap-3"
            onSubmit={(e) => {
              e.preventDefault()
              register.mutate(label)
            }}
          >
            <label htmlFor="pk-label" className="flex flex-col gap-1">
              <span className="text-sm leading-5 font-medium text-ink-muted">Name</span>
              <input
                id="pk-label"
                value={label}
                onChange={(e) => setLabel(e.target.value)}
                maxLength={60}
                className="h-10 w-64 rounded-md border-[1.5px] border-line-strong bg-surface px-3 text-ink"
              />
            </label>
            <Button type="submit" variant="command" disabled={register.isPending}>
              {register.isPending ? 'Waiting for your device…' : 'Add a passkey'}
            </Button>
          </form>
        )}
        {register.isSuccess && <Notice alert="nominal">Passkey added. Next time choose «Sign in with a passkey».</Notice>}
        {error && <Notice alert="warning">{error}</Notice>}
        <p className="text-sm leading-5 text-ink-muted">
          Demo data is recreated when the free server restarts, so passkeys registered here may need to be added again.
        </p>
      </section>
    </div>
  )
}
