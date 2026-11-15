import { useMemo, useState, type ReactNode } from 'react'
import { useAstronauts, type Astronaut } from './api.ts'
import { CurrentAstronautContext, type CurrentAstronautValue } from './context.ts'

/**
 * Whose data is on screen. The server returns only the members the signed-in
 * user may open (FR-72): a crew member gets themself, the commander, the
 * deputy and the flight surgeon get the whole crew. The choice is remembered
 * per signed-in user.
 */
export function CurrentAstronautProvider({ me, children }: { me: Astronaut; children: ReactNode }) {
  const { data, isLoading, isError } = useAstronauts()
  const storageKey = `aura.astronautId.${me.id}`
  const [selectedId, setSelectedId] = useState<number | null>(() => {
    const raw = localStorage.getItem(storageKey)
    return raw ? Number(raw) : null
  })

  const value = useMemo<CurrentAstronautValue>(() => {
    const crew = (data ?? []).filter((a) => a.role !== 'flight_surgeon')
    const current = crew.find((a) => a.id === selectedId) ?? crew.find((a) => a.id === me.id) ?? crew[0]
    return {
      me,
      crew,
      current,
      isLoading,
      isError,
      select: (id) => {
        localStorage.setItem(storageKey, String(id))
        setSelectedId(id)
      },
    }
  }, [data, selectedId, isLoading, isError, me, storageKey])

  return <CurrentAstronautContext.Provider value={value}>{children}</CurrentAstronautContext.Provider>
}
