import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api, type Schemas } from '../../shared/api/client.ts'

export type Supplies = Schemas['Supplies']
export type SupplyItem = Schemas['SupplyItem']
export type JournalEntry = Schemas['JournalEntry']

/** Medical kit with statuses and forecast (FR-90, FR-91, FR-94). */
export function useSupplies() {
  return useQuery({ queryKey: ['supplies'], queryFn: () => api<Supplies>('/supplies') })
}

/** Who took what, when and why (FR-92). */
export function useSupplyJournal() {
  return useQuery({
    queryKey: ['supply-journal'],
    queryFn: () => api<{ entries: JournalEntry[] }>('/supplies/journal').then((r) => r.entries),
  })
}

type Change =
  | { action: 'use'; id: number; quantity: number; reason: string; recommendationId?: string }
  | { action: 'count'; id: number; quantity: number }
  | { action: 'dispose'; id: number }

/** Use, count or dispose of an item; stock shown in recommendations refreshes too. */
export function useChangeSupply() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ action, id, ...body }: Change) =>
      api<SupplyItem>(`/supplies/${id}/${action}`, { method: 'POST', body: action === 'dispose' ? undefined : JSON.stringify(body) }),
    onSuccess: () => {
      for (const key of ['supplies', 'supply-journal', 'guidance', 'crew-overview', 'report']) void qc.invalidateQueries({ queryKey: [key] })
    },
  })
}

export const categoryLabel: Record<SupplyItem['category'], string> = {
  medication: 'Medications',
  consumable: 'Consumables',
  equipment: 'Equipment',
  oxygen: 'Oxygen',
}
