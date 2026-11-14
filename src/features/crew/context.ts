import { createContext, useContext } from 'react'
import type { Astronaut } from './api.ts'

export type CurrentAstronautValue = {
  /** Crew members with health data (the flight surgeon is on Earth and has none). */
  crew: Astronaut[]
  current: Astronaut | undefined
  select: (id: number) => void
  isLoading: boolean
  isError: boolean
}

export const CurrentAstronautContext = createContext<CurrentAstronautValue | null>(null)

export function useCurrentAstronaut(): CurrentAstronautValue {
  const v = useContext(CurrentAstronautContext)
  if (!v) throw new Error('useCurrentAstronaut must be used inside CurrentAstronautProvider')
  return v
}
