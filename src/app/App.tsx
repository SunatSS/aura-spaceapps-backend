import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { RouterProvider } from 'react-router/dom'
import { CurrentAstronautProvider } from '../features/crew/CurrentAstronaut.tsx'
import { router } from './router.tsx'

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      retry: 1,
    },
  },
})

export function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <CurrentAstronautProvider>
        <RouterProvider router={router} />
      </CurrentAstronautProvider>
    </QueryClientProvider>
  )
}
