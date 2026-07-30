import { lazy, Suspense } from 'react'
import { Route, Routes } from 'react-router'
import { Layout } from './components/Layout'

const App = lazy(() => import('./App.tsx'))
const ContributorPage = lazy(() =>
  import('./components/ContributorPage').then(module => ({ default: module.ContributorPage })),
)
const PRListPage = lazy(() =>
  import('./components/PRListPage').then(module => ({ default: module.PRListPage })),
)

export function AppRoutes() {
  return (
    <Suspense fallback={null}>
      <Routes>
        <Route element={<Layout />}>
          <Route path="/" element={<App />} />
          <Route path="/contributors" element={<ContributorPage />} />
          <Route path="/lines" element={<App metric="lines" />} />
          <Route path="/prs" element={<PRListPage />} />
        </Route>
      </Routes>
    </Suspense>
  )
}
