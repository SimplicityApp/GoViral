import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { ApiKeyGate } from '@/components/auth/ApiKeyGate'
import { RootLayout } from '@/components/layout/RootLayout'
import { PlatformLayout } from '@/components/layout/PlatformLayout'
import { defaultPlatform } from '@/lib/platforms'
import { Dashboard } from '@/pages/Dashboard'
import { Posts } from '@/pages/Posts'
import { Trending } from '@/pages/Trending'
import { Generate } from '@/pages/Generate'
import { History } from '@/pages/History'
import { Publish } from '@/pages/Publish'
import { Settings } from '@/pages/Settings'
import { Autopilot } from '@/pages/Autopilot'

function App() {
  return (
    <BrowserRouter>
      <ApiKeyGate>
        <Routes>
          <Route path="/" element={<Navigate to={`/${defaultPlatform}/dashboard`} replace />} />
          <Route element={<RootLayout />}>
            <Route path="/settings" element={<Settings />} />
            <Route path="/:platform" element={<PlatformLayout />}>
              <Route index element={<Navigate to="dashboard" replace />} />
              <Route path="dashboard" element={<Dashboard />} />
              <Route path="posts" element={<Posts />} />
              <Route path="trending" element={<Trending />} />
              <Route path="generate" element={<Generate />} />
              <Route path="history" element={<History />} />
              <Route path="publish" element={<Publish />} />
              <Route path="autopilot" element={<Autopilot />} />
            </Route>
          </Route>
        </Routes>
      </ApiKeyGate>
    </BrowserRouter>
  )
}

export default App
