import { createBrowserRouter } from 'react-router'
import { BodyMapPage } from '../pages/BodyMapPage.tsx'
import { CrewPage } from '../pages/CrewPage.tsx'
import { DashboardPage } from '../pages/DashboardPage.tsx'
import { LogPage } from '../pages/LogPage.tsx'
import { NotFoundPage, RouteErrorPage } from '../pages/NotFoundPage.tsx'
import { ReportPage } from '../pages/ReportPage.tsx'
import { Layout } from './Layout.tsx'

export const router = createBrowserRouter([
  {
    element: <Layout />,
    children: [
      {
        // Errors in any display render inside the app shell, keeping the status bar visible.
        errorElement: <RouteErrorPage />,
        children: [
          { index: true, element: <DashboardPage /> },
          { path: 'body', element: <BodyMapPage /> },
          { path: 'crew', element: <CrewPage /> },
          { path: 'log', element: <LogPage /> },
          { path: 'report', element: <ReportPage /> },
          // Charts (Recharts) load only when Trends is opened.
          { path: 'trends', lazy: async () => ({ Component: (await import('../pages/TrendsPage.tsx')).TrendsPage }) },
          { path: '*', element: <NotFoundPage /> },
        ],
      },
    ],
  },
])
