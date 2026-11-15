import { useCallback, useState } from 'react'

/**
 * Acknowledged recommendation IDs per astronaut, kept in localStorage (P0;
 * server-side acknowledgement comes with the alerts display in P1).
 */
export function useAcknowledged(astronautId: number | undefined) {
  const key = `aura.ack.${astronautId}`
  const [ids, setIds] = useState<string[]>(() => read(key))
  const [forKey, setForKey] = useState(key)
  if (forKey !== key) {
    setForKey(key)
    setIds(read(key))
  }

  const acknowledge = useCallback(
    (id: string) => {
      setIds((prev) => {
        const next = prev.includes(id) ? prev : [...prev, id]
        localStorage.setItem(key, JSON.stringify(next))
        return next
      })
    },
    [key],
  )
  return { isAcknowledged: (id: string) => ids.includes(id), acknowledge }
}

function read(key: string): string[] {
  try {
    const v: unknown = JSON.parse(localStorage.getItem(key) ?? '[]')
    return Array.isArray(v) ? v.filter((x): x is string => typeof x === 'string') : []
  } catch {
    return []
  }
}
