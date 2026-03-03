import { useState, useEffect, type ReactNode } from 'react'
import { useLocation } from 'react-router-dom'
import { apiClient } from '../../lib/api'

const STORAGE_KEY = 'goviral_api_key'

async function checkAuth(): Promise<boolean> {
  try {
    await apiClient.get('/config')
    return true
  } catch {
    return false
  }
}

export function ApiKeyGate({ children, publicPaths = [] }: { children: ReactNode; publicPaths?: string[] }) {
  const { pathname } = useLocation()
  const isPublic = publicPaths.includes(pathname)
  const [status, setStatus] = useState<'checking' | 'ok' | 'needs_key'>('checking')
  const [input, setInput] = useState('')
  const [error, setError] = useState('')

  useEffect(() => {
    if (isPublic) return
    checkAuth().then((ok) => setStatus(ok ? 'ok' : 'needs_key'))
  }, [isPublic])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    const key = input.trim()
    if (!key) return

    localStorage.setItem(STORAGE_KEY, key)
    const ok = await checkAuth()
    if (ok) {
      setStatus('ok')
    } else {
      localStorage.removeItem(STORAGE_KEY)
      setError('Invalid API key')
    }
  }

  if (isPublic) {
    return <>{children}</>
  }

  if (status === 'checking') {
    return (
      <div className="flex h-screen items-center justify-center bg-[var(--color-bg)]">
        <div className="h-6 w-6 animate-spin rounded-full border-2 border-[var(--color-accent)] border-t-transparent" />
      </div>
    )
  }

  if (status === 'needs_key') {
    return (
      <div className="flex h-screen items-center justify-center bg-[var(--color-bg)]">
        <form
          onSubmit={handleSubmit}
          className="w-full max-w-sm rounded-[var(--radius-card)] border border-[var(--color-border)] bg-[var(--color-card)] p-8"
        >
          <h1 className="mb-1 text-xl font-bold text-[var(--color-text)]">GoViral</h1>
          <p className="mb-6 text-sm text-[var(--color-text-secondary)]">
            Enter your server API key to continue.
          </p>
          <input
            type="password"
            value={input}
            onChange={(e) => setInput(e.target.value)}
            placeholder="Server API key"
            autoFocus
            className="mb-3 w-full rounded-[var(--radius-input)] border border-[var(--color-border)] bg-[var(--color-bg)] px-3 py-2 text-sm text-[var(--color-text)] placeholder:text-[var(--color-text-secondary)] focus:border-[var(--color-accent)] focus:outline-none"
          />
          {error && <p className="mb-3 text-sm text-red-500">{error}</p>}
          <button
            type="submit"
            className="w-full rounded-[var(--radius-button)] bg-[var(--color-accent)] px-4 py-2.5 text-sm font-medium text-white transition-colors hover:bg-[var(--color-accent-hover)]"
          >
            Connect
          </button>
        </form>
      </div>
    )
  }

  return <>{children}</>
}
