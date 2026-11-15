import { useMemo } from 'react'

const NAMES = ['ink', 'ink-muted', 'line', 'line-strong', 'caution-ink', 'warning-ink', 'stale-ink', 'caution', 'warning', 'surface-raised'] as const
export type TokenName = (typeof NAMES)[number]

/**
 * Resolved colour tokens for libraries that take colours as SVG attributes
 * (Recharts), where CSS `var()` does not work.
 */
export function useTokens(): Record<TokenName, string> {
  return useMemo(() => {
    const style = getComputedStyle(document.documentElement)
    return Object.fromEntries(NAMES.map((n) => [n, style.getPropertyValue(`--color-${n}`).trim() || '#888888'])) as Record<TokenName, string>
  }, [])
}
