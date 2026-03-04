import { useState } from 'react'
import type { AppConfig } from '@/hooks/useConfig'
import { apiClient, BASE_URL } from '@/lib/api'
import { useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { Download, Puzzle } from 'lucide-react'

export function ConnectLinkedInStep({
  config,
  extension,
  extensionExtracting,
  extractCookies,
}: {
  config: AppConfig | undefined
  extension: { available: boolean; version: string | null }
  extensionExtracting: boolean
  extractCookies: () => Promise<{ x: { auth_token: string; ct0: string } | null; linkedin: { li_at: string; jsessionid: string } | null }>
}) {
  const [cookiesSynced, setCookiesSynced] = useState(!!config?.linkedin.has_linkitin_auth)
  const queryClient = useQueryClient()

  const handleExtractCookies = async () => {
    try {
      const cookies = await extractCookies()
      if (cookies.linkedin) {
        await apiClient.post('/linkedin/login-cookies', cookies.linkedin)
        void queryClient.invalidateQueries({ queryKey: ['config'] })
        toast.success('LinkedIn cookies synced')
        setCookiesSynced(true)
      } else {
        toast.error('No LinkedIn cookies found — log into LinkedIn first')
      }
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Cookie extraction failed')
    }
  }

  return (
    <div className="flex flex-col items-center">
      <h2 className="mb-2 text-xl font-bold text-[var(--color-text)]">
        Connect LinkedIn
      </h2>
      <p className="mb-6 text-sm text-[var(--color-text-secondary)]">
        Sync your LinkedIn cookies so GoViral can fetch your posts and publish content.
      </p>
      <div className="w-full max-w-md">
        <div className="rounded-[var(--radius-card)] border border-[var(--color-border)] bg-[var(--color-card)] p-4">
          <h3 className="mb-2 flex items-center gap-2 text-sm font-medium text-[var(--color-text)]">
            <Puzzle size={14} /> Cookie Sync
          </h3>
          {cookiesSynced && (
            <div className="mb-2">
              <span className="rounded-full bg-green-500/15 px-3 py-1 text-xs font-medium text-green-400">
                Connected
              </span>
            </div>
          )}
          {extension.available ? (
            <>
              <p className="mb-3 text-xs text-[var(--color-text-secondary)]">
                One-click extraction of LinkedIn cookies from your browser.
              </p>
              <button
                onClick={handleExtractCookies}
                disabled={extensionExtracting}
                className="flex items-center gap-2 rounded-[var(--radius-button)] bg-[var(--color-accent)] px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-[var(--color-accent-hover)] disabled:opacity-50"
              >
                <Download size={16} />
                {extensionExtracting ? 'Extracting...' : 'Extract LinkedIn Cookies'}
              </button>
            </>
          ) : (
            <>
              <p className="mb-2 text-xs text-[var(--color-text-secondary)]">
                Install the GoViral browser extension for one-click cookie sync.
              </p>
              <a
                href={`${BASE_URL}/extension/download`}
                className="inline-flex items-center gap-2 rounded-[var(--radius-button)] bg-[var(--color-accent)] px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-[var(--color-accent-hover)]"
              >
                <Download size={16} /> Download Extension
              </a>
            </>
          )}
        </div>
      </div>
    </div>
  )
}
