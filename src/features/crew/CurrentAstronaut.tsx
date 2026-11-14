import { useMemo, useState, type ReactNode } from 'react'
import { useAstronauts } from './api.ts'
import { CurrentAstronautContext, type CurrentAstronautValue } from './context.ts'

const STORAGE_KEY = 'aura.astronautId'

/**
 * Who is using AURA. P0 has no login: the user picks a crew member and the
 * choice is remembered in localStorage. P1 replaces this with sign-in.
 */
export function CurrentAstronautProvider({ children }: { children: ReactNode }) {
  const { data, isLoading, isError } = useAstronauts()
  const [selectedId, setSelectedId] = useState<number | null>(() => {
    const raw = localStorage.getItem(STORAGE_KEY)
    return raw ? Number(raw) : null
  })

  const value = useMemo<CurrentAstronautValue>(() => {
    const crew = (data ?? []).filter((a) => a.role !== 'flight_surgeon')
    const current = crew.find((a) => a.id === selectedId) ?? crew[0]
    return {
      crew,
      current,
      isLoading,
      isError,
      select: (id) => {
        localStorage.setItem(STORAGE_KEY, String(id))
        setSelectedId(id)
      },
    }
  }, [data, selectedId, isLoading, isError])

  return <CurrentAstronautContext.Provider value={value}>{children}</CurrentAstronautContext.Provider>
}
