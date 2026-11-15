import { api, type Schemas } from '../../shared/api/client.ts'

/**
 * Passkey ceremonies (FR-74): the browser talks to Windows Hello, Touch ID,
 * Face ID or a security key; the server verifies the result. Binary fields
 * travel as base64url.
 */

export type Passkey = Schemas['Passkey']
type Session = Schemas['Session']

const toB64url = (buf: ArrayBuffer) =>
  btoa(String.fromCharCode(...new Uint8Array(buf)))
    .replace(/\+/g, '-')
    .replace(/\//g, '_')
    .replace(/=+$/, '')

const fromB64url = (s: string) => {
  const b64 = s.replace(/-/g, '+').replace(/_/g, '/') + '==='.slice((s.length + 3) % 4)
  return Uint8Array.from(atob(b64), (c) => c.charCodeAt(0))
}

/** Whether this browser can use passkeys at all (needs HTTPS or localhost). */
export function passkeysSupported(): boolean {
  return typeof window !== 'undefined' && window.isSecureContext && 'PublicKeyCredential' in window && !!navigator.credentials
}

/** Creates a passkey for the signed-in member and stores it on the server. */
export async function registerPasskey(label: string): Promise<Passkey> {
  const o = await api<Schemas['PasskeyRegisterOptions']>('/auth/passkeys/register/begin', { method: 'POST' })
  const cred = (await navigator.credentials.create({
    publicKey: {
      challenge: fromB64url(o.challenge),
      rp: o.rp,
      user: { id: fromB64url(o.user.id), name: o.user.name, displayName: o.user.displayName },
      pubKeyCredParams: o.pubKeyCredParams.map((p) => ({ type: 'public-key' as const, alg: p.alg })),
      excludeCredentials: o.excludeCredentialIds.map((id) => ({ type: 'public-key' as const, id: fromB64url(id) })),
      authenticatorSelection: { residentKey: 'required', requireResidentKey: true, userVerification: 'preferred' },
      attestation: 'none',
      timeout: o.timeout,
    },
  })) as PublicKeyCredential | null
  if (!cred) throw new Error('No passkey was created.')
  const res = cred.response as AuthenticatorAttestationResponse
  return api<Passkey>('/auth/passkeys/register/finish', {
    method: 'POST',
    body: JSON.stringify({
      challenge: o.challenge,
      clientDataJSON: toB64url(res.clientDataJSON),
      attestationObject: toB64url(res.attestationObject),
      label,
    }),
  })
}

/** Signs in with any passkey this device offers for AURA. */
export async function signInWithPasskey(): Promise<Session> {
  const o = await api<{ challenge: string; rpId: string; timeout: number }>('/auth/passkeys/login/begin', { method: 'POST' })
  const cred = (await navigator.credentials.get({
    publicKey: { challenge: fromB64url(o.challenge), rpId: o.rpId, userVerification: 'preferred', timeout: o.timeout },
  })) as PublicKeyCredential | null
  if (!cred) throw new Error('No passkey was chosen.')
  const res = cred.response as AuthenticatorAssertionResponse
  return api<Session>('/auth/passkeys/login/finish', {
    method: 'POST',
    body: JSON.stringify({
      challenge: o.challenge,
      id: cred.id,
      clientDataJSON: toB64url(res.clientDataJSON),
      authenticatorData: toB64url(res.authenticatorData),
      signature: toB64url(res.signature),
      userHandle: res.userHandle ? toB64url(res.userHandle) : '',
    }),
  })
}

/** A readable reason for a failed ceremony. */
export function passkeyError(e: unknown): string {
  if (e instanceof DOMException) {
    if (e.name === 'NotAllowedError') return 'The passkey request was cancelled or timed out.'
    if (e.name === 'InvalidStateError') return 'This device already has a passkey for your account.'
    if (e.name === 'SecurityError') return 'Passkeys need a secure connection (HTTPS).'
    return e.message
  }
  return e instanceof Error ? e.message : 'Passkey failed.'
}
