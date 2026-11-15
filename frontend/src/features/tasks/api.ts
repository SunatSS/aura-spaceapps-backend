import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api, type Schemas } from '../../shared/api/client.ts'

export type Tasks = Schemas['Tasks']
export type Task = Schemas['Task']

/** Today's tasks, overdue reminders, streak and badges (FR-40…FR-43). */
export function useTasks(astronautId: number | undefined) {
  return useQuery({
    queryKey: ['tasks', astronautId],
    queryFn: () => api<Tasks>(`/astronauts/${astronautId}/tasks`),
    enabled: astronautId !== undefined,
    refetchInterval: 60_000,
  })
}

/** Mark or unmark a manual task (exercise) for today. */
export function useToggleTask(astronautId: number | undefined) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ code, done }: { code: string; done: boolean }) =>
      api<Tasks>(`/astronauts/${astronautId}/tasks/${code}/complete`, { method: done ? 'POST' : 'DELETE' }),
    onSuccess: (data) => {
      qc.setQueryData(['tasks', astronautId], data)
      void qc.invalidateQueries({ queryKey: ['crew-overview'] })
    },
  })
}
