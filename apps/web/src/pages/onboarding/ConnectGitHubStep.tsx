import { useState, useEffect, useRef } from 'react'
import type { AppConfig } from '@/hooks/useConfig'
import { apiClient } from '@/lib/api'
import { useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { Check, X, Github } from 'lucide-react'

export function ConnectGitHubStep({ config }: { config: AppConfig | undefined }) {
  const hasAuth = config?.github?.has_auth ?? false
  const hasOAuth = config?.github?.has_oauth ?? false
  const hasPat = config?.github?.has_pat ?? false
  const [connecting, setConnecting] = useState(false)
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null)
  const queryClient = useQueryClient()

  useEffect(() => {
    return () => {
      if (pollRef.current) clearInterval(pollRef.current)
    }
  }, [])

  const handleConnect = async () => {
    setConnecting(true)
    try {
      const resp = await apiClient.post<{ auth_url: string; key: string; status: string }>('/auth/github/start', {})
      if (resp.auth_url) {
        window.open(resp.auth_url, '_blank', 'noopener')
      }

      const authKey = resp.key
      // Poll for completion
      pollRef.current = setInterval(async () => {
        try {
          const status = await apiClient.get<{ status: string; error?: string }>(`/auth/github/status?key=${authKey}`)
          if (status.status === 'completed') {
            if (pollRef.current) clearInterval(pollRef.current)
            pollRef.current = null
            setConnecting(false)
            void queryClient.invalidateQueries({ queryKey: ['config'] })
            toast.success('GitHub connected')
          } else if (status.status === 'failed') {
            if (pollRef.current) clearInterval(pollRef.current)
            pollRef.current = null
            setConnecting(false)
            toast.error(status.error || 'GitHub authorization failed')
          }
        } catch {
          // ignore polling errors
        }
      }, 2000)
    } catch (err) {
      setConnecting(false)
      toast.error(err instanceof Error ? err.message : 'Failed to start GitHub OAuth')
    }
  }

  return (
    <div className="flex flex-col items-center">
      <h2 className="mb-2 text-xl font-bold text-[var(--color-text)]">
        Connect GitHub
      </h2>
      <p className="mb-6 text-sm text-[var(--color-text-secondary)]">
        Connect your GitHub account so GoViral can read your repository commits for the Code to Post feature.
      </p>
      <div className="flex w-full max-w-md flex-col gap-4 items-center">
        {hasAuth ? (
          <div className="flex items-center gap-2 rounded-lg bg-green-100 px-4 py-3 text-sm font-medium text-green-800 dark:bg-green-500/15 dark:text-green-400">
            <Check size={16} />
            GitHub connected
          </div>
        ) : hasOAuth ? (
          <button
            onClick={handleConnect}
            disabled={connecting}
            className="flex items-center gap-2 rounded-[var(--radius-button)] bg-[var(--color-accent)] px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-[var(--color-accent-hover)] disabled:opacity-50"
          >
            <Github size={16} />
            {connecting ? 'Waiting for authorization...' : 'Connect GitHub'}
          </button>
        ) : hasPat ? (
          <div className="flex items-center gap-2 rounded-lg bg-green-100 px-4 py-3 text-sm font-medium text-green-800 dark:bg-green-500/15 dark:text-green-400">
            <Check size={16} />
            GitHub token configured
          </div>
        ) : (
          <div className="flex items-center gap-2 rounded-lg bg-amber-100 px-4 py-3 text-sm font-medium text-amber-800 dark:bg-amber-500/15 dark:text-amber-400">
            <X size={16} />
            GitHub not configured — contact your server administrator
          </div>
        )}
      </div>
    </div>
  )
}
