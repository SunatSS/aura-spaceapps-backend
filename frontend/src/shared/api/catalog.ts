import { useQuery } from '@tanstack/react-query'
import { api, type Schemas } from './client.ts'

export type Catalog = Schemas['Catalog']
export type Indicator = Schemas['Indicator']

/** Health content catalog: indicators, body systems, NASA HRP risks. Static per deploy. */
export function useCatalog() {
  return useQuery({
    queryKey: ['catalog'],
    queryFn: () => api<Catalog>('/indicators'),
    staleTime: Infinity,
  })
}
