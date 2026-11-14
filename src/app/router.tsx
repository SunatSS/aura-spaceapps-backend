import { createBrowserRouter } from 'react-router'
import { BodyMapPage } from '../pages/BodyMapPage.tsx'
import { DashboardPage } from '../pages/DashboardPage.tsx'
import { LogPage } from '../pages/LogPage.tsx'
import { TrendsPage } from '../pages/TrendsPage.tsx'
import { Layout } from './Layout.tsx'

export const router = createBrowserRouter([
  {
    element: <Layout />,
    children: [
      { index: true, element: <DashboardPage /> },
      { path: 'body', element: <BodyMapPage /> },
      { path: 'log', element: <LogPage /> },
      { path: 'trends', element: <TrendsPage /> },
    ],
  },
])
