import { useState } from 'react'
import { useUpdateConfigMutation } from '@/hooks/useConfig'
import type { AppConfig } from '@/hooks/useConfig'
import { apiClient, BASE_URL } from '@/lib/api'
import { useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { Download, Puzzle, Check } from 'lucide-react'

export function ConnectXStep({
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
  const [username, setUsername] = useState(config?.x.username || '')
  const [saved, setSaved] = useState(false)
  const [cookiesSynced, setCookiesSynced] = useState(!!config?.x.has_twikit_auth)
  const updateConfig = useUpdateConfigMutation()
  const queryClient = useQueryClient()

  const handleSaveUsername = () => {
    updateConfig.mutate(
      { x: { username } },
      {
        onSuccess: () => {
          toast.success('X username saved')
          setSaved(true)
        },
        onError: () => toast.error('Failed to save username'),
      },
    )
  }

  const handleExtractCookies = async () => {
    try {
      const cookies = await extractCookies()
      if (cookies.x) {
        await apiClient.post('/x/login-cookies', cookies.x)
        void queryClient.invalidateQueries({ queryKey: ['config'] })
        toast.success('X cookies synced')
        setCookiesSynced(true)
      } else {
        toast.error('No X cookies found — log into X first')
      }
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Cookie extraction failed')
    }
  }

  return (
    <div className="flex flex-col items-center">
      <h2 className="mb-2 text-xl font-bold text-[var(--color-text)]">
        Connect X (Twitter)
      </h2>
      <p className="mb-6 text-sm text-[var(--color-text-secondary)]">
        Enter your username and sync cookies so GoViral can fetch your posts and
        publish on your behalf.
      </p>
      <div className="flex w-full max-w-md flex-col gap-6">
        {/* Username */}
        <div>
          <label className="mb-1 block text-sm font-medium text-[var(--color-text)]">
            Username
          </label>
          <div className="flex items-center gap-2">
            <span className="text-sm text-[var(--color-text-secondary)]">@</span>
            <input
              type="text"
              value={username}
              onChange={(e) => { setUsername(e.target.value); setSaved(false) }}
              placeholder="your_handle"
              className="flex-1 rounded-[var(--radius-input)] border border-[var(--color-border)] bg-[var(--color-card)] px-3 py-2 text-sm text-[var(--color-text)] placeholder:text-[var(--color-text-secondary)]"
            />
          </div>
          <button
            onClick={handleSaveUsername}
            disabled={updateConfig.isPending || !username.trim()}
            className="mt-2 flex items-center gap-2 rounded-[var(--radius-button)] bg-[var(--color-accent)] px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-[var(--color-accent-hover)] disabled:opacity-50"
          >
            {saved ? <><Check size={16} /> Saved</> : updateConfig.isPending ? 'Saving...' : 'Save Username'}
          </button>
        </div>

        {/* Cookie Sync */}
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
                One-click extraction of X cookies from your browser.
              </p>
              <button
                onClick={handleExtractCookies}
                disabled={extensionExtracting}
                className="flex items-center gap-2 rounded-[var(--radius-button)] bg-[var(--color-accent)] px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-[var(--color-accent-hover)] disabled:opacity-50"
              >
                <Download size={16} />
                {extensionExtracting ? 'Extracting...' : 'Extract X Cookies'}
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
