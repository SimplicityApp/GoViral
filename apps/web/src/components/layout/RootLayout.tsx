import { Link, Outlet } from 'react-router-dom'
import { Sidebar } from './Sidebar'
import { TopBar } from './TopBar'

export function RootLayout() {
  return (
    <div className="flex h-screen">
      <Sidebar />
      <div className="flex flex-1 flex-col overflow-hidden">
        <TopBar />
        <main className="flex-1 overflow-y-auto">
          <Outlet />
          <footer className="mt-8 border-t border-[var(--color-border)] px-6 py-4 text-center text-xs text-[var(--color-text-secondary)]">
            <Link to="/terms" className="hover:text-[var(--color-text)] transition-colors">
              Terms of Service
            </Link>
            <span className="mx-2">·</span>
            <Link to="/privacy" className="hover:text-[var(--color-text)] transition-colors">
              Privacy Policy
            </Link>
          </footer>
        </main>
      </div>
    </div>
  )
}
