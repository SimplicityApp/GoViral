import { useEffect } from 'react'
import { BrowserRouter, Routes, Route } from 'react-router-dom'
import { Sidebar } from '@/components/layout/Sidebar'
import { TopBar } from '@/components/layout/TopBar'
import { ApiKeyGate } from '@/components/auth/ApiKeyGate'
import { defaultPlatform } from '@/lib/platforms'
import { Dashboard } from '@/pages/Dashboard'
import { Posts } from '@/pages/Posts'
import { Trending } from '@/pages/Trending'
import { Generate } from '@/pages/Generate'
import { History } from '@/pages/History'
import { Publish } from '@/pages/Publish'
import { Settings } from '@/pages/Settings'

function App() {
  useEffect(() => {
    document.documentElement.dataset.platform = defaultPlatform
  }, [])

  return (
    <BrowserRouter>
      <ApiKeyGate>
        <div className="flex h-screen">
          <Sidebar />
          <div className="flex flex-1 flex-col overflow-hidden">
            <TopBar />
            <main className="flex-1 overflow-y-auto">
              <Routes>
                <Route path="/" element={<Dashboard />} />
                <Route path="/posts" element={<Posts />} />
                <Route path="/trending" element={<Trending />} />
                <Route path="/generate" element={<Generate />} />
                <Route path="/history" element={<History />} />
                <Route path="/publish" element={<Publish />} />
                <Route path="/settings" element={<Settings />} />
              </Routes>
            </main>
          </div>
        </div>
      </ApiKeyGate>
    </BrowserRouter>
  )
}

export default App
