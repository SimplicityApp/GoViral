import { useState, useEffect, useMemo } from 'react'
import { useConfigQuery, useUpdateConfigMutation } from '@/hooks/useConfig'
import type { UpdateConfigPayload } from '@/hooks/useConfig'
import { useDaemonConfigQuery, useUpdateDaemonConfigMutation } from '@/hooks/useDaemon'
import { usePersonaQuery } from '@/hooks/usePersona'
import { useExtensionCookies } from '@/hooks/useExtensionCookies'
import { usePlatformStore } from '@/stores/platform-store'
import { LoadingSpinner } from '@/components/shared/LoadingSpinner'
import { MaskedInput } from '@/components/shared/MaskedInput'
import { NicheSelector } from '@/components/shared/NicheSelector'
import { Puzzle, Download, ChevronDown } from 'lucide-react'
import { toast } from 'sonner'
import { useQueryClient } from '@tanstack/react-query'
import { apiClient, BASE_URL } from '@/lib/api'

export function Settings() {
  const { activePlatform } = usePlatformStore()
  const { data: config, isLoading } = useConfigQuery()
  const { data: persona } = usePersonaQuery(activePlatform)
  const updateConfig = useUpdateConfigMutation()
  const queryClient = useQueryClient()
  const { extension, extracting: extensionExtracting, extractCookies } = useExtensionCookies()
  const [xCookieForm, setXCookieForm] = useState({ auth_token: '', ct0: '' })
  const [liCookieForm, setLiCookieForm] = useState({ li_at: '', jsessionid: '' })

  const [activeTab, setActiveTab] = useState<'x' | 'linkedin' | 'github' | 'telegram'>('x')
  const [xAdvancedOpen, setXAdvancedOpen] = useState(false)
  const [liAdvancedOpen, setLiAdvancedOpen] = useState(false)

  const { data: daemonConfig } = useDaemonConfigQuery()
  const updateDaemonConfig = useUpdateDaemonConfigMutation()
  const [telegramForm, setTelegramForm] = useState({
    bot_token: '',
    chat_id: 0,
    webhook_url: '',
  })

  useEffect(() => {
    if (daemonConfig) {
      setTelegramForm({
        bot_token: daemonConfig.telegram.bot_token,
        chat_id: daemonConfig.telegram.chat_id,
        webhook_url: daemonConfig.telegram.webhook_url,
      })
    }
  }, [daemonConfig])

  const handleSaveTelegram = () => {
    if (!daemonConfig) return
    updateDaemonConfig.mutate(
      {
        daemon: daemonConfig.daemon,
        telegram: {
          bot_token: telegramForm.bot_token,
          chat_id: telegramForm.chat_id,
          webhook_url: telegramForm.webhook_url,
          connected: daemonConfig.telegram.connected,
        },
      },
      {
        onSuccess: () => toast.success('Telegram settings saved'),
        onError: () => toast.error('Failed to save Telegram settings'),
      },
    )
  }

  const [form, setForm] = useState({
    claude_api_key: '',
    gemini_api_key: '',
    x_username: '',
    x_api_key: '',
    x_api_secret: '',
    x_client_id: '',
    x_client_secret: '',
    linkedin_client_id: '',
    linkedin_client_secret: '',
    github_pat: '',
    niches: [] as string[],
    linkedin_niches: [] as string[],
  })

  useEffect(() => {
    if (config) {
      setForm({
        claude_api_key: config.claude.api_key || '',
        gemini_api_key: config.gemini.api_key || '',
        x_username: config.x.username || '',
        x_api_key: config.x.api_key || '',
        x_api_secret: config.x.api_secret || '',
        x_client_id: config.x.client_id || '',
        x_client_secret: config.x.client_secret || '',
        linkedin_client_id: config.linkedin.client_id || '',
        linkedin_client_secret: config.linkedin.client_secret || '',
        github_pat: config.github?.personal_access_token || '',
        niches: config.niches || [],
        linkedin_niches: config.linkedin_niches || [],
      })
      setXCookieForm({
        auth_token: config.x.auth_token || '',
        ct0: config.x.ct0 || '',
      })
      setLiCookieForm({
        li_at: config.linkedin.li_at || '',
        jsessionid: config.linkedin.jsessionid || '',
      })
    }
  }, [config])

  const allNiches = useMemo(
    () => [...new Set([...form.niches, ...form.linkedin_niches])],
    [form.niches, form.linkedin_niches],
  )

  const handleSave = () => {
    // Build nested payload matching server's UpdateConfigRequest
    const payload: UpdateConfigPayload = {
      claude: { api_key: form.claude_api_key },
      gemini: { api_key: form.gemini_api_key },
      x: {
        username: form.x_username,
        api_key: form.x_api_key,
        api_secret: form.x_api_secret,
        client_id: form.x_client_id,
        client_secret: form.x_client_secret,
      },
      linkedin: {
        client_id: form.linkedin_client_id,
        client_secret: form.linkedin_client_secret,
      },
      github: { personal_access_token: form.github_pat },
      niches: form.niches,
      linkedin_niches: form.linkedin_niches,
    }

    updateConfig.mutate(payload, {
      onSuccess: () => toast.success('Settings saved'),
      onError: () => toast.error('Failed to save settings'),
    })
  }

  if (isLoading) return <LoadingSpinner />

  return (
    <div className="mx-auto max-w-2xl p-6">
      <h2 className="mb-6 text-lg font-semibold text-[var(--color-text)]">Settings</h2>

      <section className="mb-8">
        <h3 className="mb-4 text-sm font-semibold uppercase tracking-wide text-[var(--color-text-secondary)]">
          AI API Keys
        </h3>
        <div className="flex flex-col gap-4">
          <MaskedInput
            label="Claude API Key"
            value={form.claude_api_key}
            onChange={(v) => setForm((f) => ({ ...f, claude_api_key: v }))}
          />
          <MaskedInput
            label="Gemini API Key"
            value={form.gemini_api_key}
            onChange={(v) => setForm((f) => ({ ...f, gemini_api_key: v }))}
          />
        </div>
      </section>

      <section className="mb-8">
        <h3 className="mb-4 text-sm font-semibold uppercase tracking-wide text-[var(--color-text-secondary)]">
          <Puzzle size={14} className="mr-1.5 inline" />
          Browser Cookie Sync
        </h3>
        {extension.available ? (
          <>
            <div className="mb-3 flex items-center gap-2">
              <span className="rounded-full bg-green-500/15 px-3 py-1 text-xs font-medium text-green-400">
                Extension detected{extension.version ? ` v${extension.version}` : ''}
              </span>
            </div>
            <p className="mb-3 text-xs text-[var(--color-text-secondary)]">
              One-click extraction of X and LinkedIn cookies from your browser.
            </p>
            <button
              onClick={async () => {
                try {
                  const cookies = await extractCookies()
                  let xOk = false
                  let liOk = false

                  if (cookies.x) {
                    setXCookieForm(cookies.x)
                    try {
                      await apiClient.post('/x/login-cookies', cookies.x)
                      xOk = true
                    } catch {
                      toast.error('Failed to save X cookies')
                    }
                  }

                  if (cookies.linkedin) {
                    setLiCookieForm(cookies.linkedin)
                    try {
                      await apiClient.post('/linkedin/login-cookies', cookies.linkedin)
                      liOk = true
                    } catch {
                      toast.error('Failed to save LinkedIn cookies')
                    }
                  }

                  void queryClient.invalidateQueries({ queryKey: ['config'] })

                  if (xOk && liOk) {
                    toast.success('X and LinkedIn cookies synced')
                  } else if (xOk) {
                    toast.success('X cookies synced' + (!cookies.linkedin ? ' (not logged into LinkedIn)' : ''))
                  } else if (liOk) {
                    toast.success('LinkedIn cookies synced' + (!cookies.x ? ' (not logged into X)' : ''))
                  } else if (!cookies.x && !cookies.linkedin) {
                    toast.error('No cookies found — log into X or LinkedIn first')
                  }
                } catch (err) {
                  toast.error(err instanceof Error ? err.message : 'Cookie extraction failed')
                }
              }}
              disabled={extensionExtracting}
              className="flex items-center gap-2 rounded-[var(--radius-button)] bg-[var(--color-accent)] px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-[var(--color-accent-hover)] disabled:opacity-50"
            >
              <Download size={16} />
              {extensionExtracting ? 'Extracting...' : 'Extract & Sync All Cookies'}
            </button>
          </>
        ) : (
          <>
            <div className="mb-3 flex items-center gap-2">
              <span className="rounded-full bg-yellow-500/15 px-3 py-1 text-xs font-medium text-yellow-400">
                Extension not detected
              </span>
            </div>
            <div className="rounded-[var(--radius-card)] border border-[var(--color-border)] bg-[var(--color-card)] p-4 text-sm text-[var(--color-text-secondary)]">
              <p className="mb-2 font-medium text-[var(--color-text)]">Install the GoViral extension for one-click cookie sync:</p>
              <a
                href={`${BASE_URL}/extension/download`}
                className="mb-3 inline-flex items-center gap-2 rounded-[var(--radius-button)] bg-[var(--color-accent)] px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-[var(--color-accent-hover)]"
              >
                <Download size={16} />
                Download Extension
              </a>
              <ol className="mt-3 list-inside list-decimal space-y-1 text-xs">
                <li>Unzip the downloaded file</li>
                <li>Open <code className="rounded bg-[var(--color-border)] px-1">chrome://extensions</code></li>
                <li>Enable <strong>Developer mode</strong> (top right)</li>
                <li>Click <strong>Load unpacked</strong> and select the unzipped folder</li>
                <li>Refresh this page</li>
              </ol>
            </div>
          </>
        )}
      </section>

      {/* Tab bar */}
      <div className="mb-6 flex gap-1 border-b border-[var(--color-border)]">
        {(['x', 'linkedin', 'github', 'telegram'] as const).map((tab) => (
          <button
            key={tab}
            onClick={() => setActiveTab(tab)}
            className={`px-4 pb-2.5 text-sm font-medium transition-colors ${
              activeTab === tab
                ? 'border-b-2 border-[var(--color-accent)] text-[var(--color-accent)]'
                : 'text-[var(--color-text-secondary)] hover:text-[var(--color-text)]'
            }`}
          >
            {tab === 'x' ? 'X (Twitter)' : tab === 'linkedin' ? 'LinkedIn' : tab === 'github' ? 'GitHub' : 'Telegram'}
          </button>
        ))}
      </div>

      {/* X tab */}
      {activeTab === 'x' && (
        <>
          <section className="mb-8">
            <h3 className="mb-4 text-sm font-semibold uppercase tracking-wide text-[var(--color-text-secondary)]">
              X (Twitter) Username
            </h3>
            <div>
              <label className="mb-1 block text-sm font-medium text-[var(--color-text)]">
                Username <span className="text-red-500">*</span>
              </label>
              <div className="flex items-center gap-2">
                <span className="text-sm text-[var(--color-text-secondary)]">@</span>
                <input
                  type="text"
                  value={form.x_username}
                  onChange={(e) => setForm((f) => ({ ...f, x_username: e.target.value }))}
                  placeholder="your_handle"
                  className="flex-1 rounded-[var(--radius-input)] border border-[var(--color-border)] bg-[var(--color-card)] px-3 py-2 text-sm text-[var(--color-text)] placeholder:text-[var(--color-text-secondary)]"
                />
              </div>
            </div>
          </section>

          <section className="mb-8">
            <h3 className="mb-4 text-sm font-semibold uppercase tracking-wide text-[var(--color-text-secondary)]">
              X Cookie Auth (Twikit)
            </h3>
            <div className="mb-2 flex items-center gap-2">
              {config?.x.has_twikit_auth ? (
                <span className="rounded-full bg-green-500/15 px-3 py-1 text-xs font-medium text-green-400">
                  Connected
                </span>
              ) : (
                <span className="rounded-full bg-gray-500/15 px-3 py-1 text-xs font-medium text-gray-400">
                  Not Set Up
                </span>
              )}
            </div>
            <p className="mb-3 text-xs text-[var(--color-text-secondary)]">
              Synced via Browser Cookie Sync above. Required for posting via twikit fallback.
            </p>
            <div className="flex flex-col gap-3">
              <MaskedInput
                label="auth_token"
                value={xCookieForm.auth_token}
                onChange={() => {}}
              />
              <MaskedInput
                label="ct0"
                value={xCookieForm.ct0}
                onChange={() => {}}
              />
            </div>
          </section>

          <section className="mb-8">
            <h3 className="mb-4 text-sm font-semibold uppercase tracking-wide text-[var(--color-text-secondary)]">
              X Niche Tags
            </h3>
            <NicheSelector
              selected={form.niches}
              allNiches={allNiches}
              onChange={(niches) => setForm((f) => ({ ...f, niches }))}
              onAddNiche={(tag) =>
                setForm((f) => ({
                  ...f,
                  niches: f.niches.includes(tag) ? f.niches : [...f.niches, tag],
                }))
              }
            />
          </section>

          {persona && activePlatform === 'x' && (
            <section className="mb-8">
              <h3 className="mb-4 text-sm font-semibold uppercase tracking-wide text-[var(--color-text-secondary)]">
                Persona (x)
              </h3>
              <div className="rounded-[var(--radius-card)] border border-[var(--color-border)] bg-[var(--color-card)] p-4 text-sm text-[var(--color-text)]">
                <p className="mb-2">{persona.profile.voice_summary}</p>
                <div className="mt-3 grid grid-cols-2 gap-2 text-xs text-[var(--color-text-secondary)]">
                  <div>Tone: {persona.profile.writing_tone}</div>
                  <div>Length: {persona.profile.typical_length}</div>
                  <div>Vocab: {persona.profile.vocabulary_level}</div>
                  <div>Emoji: {persona.profile.emoji_usage}</div>
                </div>
              </div>
            </section>
          )}

          <section className="mb-8">
            <button
              type="button"
              onClick={() => setXAdvancedOpen((o) => !o)}
              className="flex w-full items-center justify-between rounded-[var(--radius-card)] border border-[var(--color-border)] bg-[var(--color-card)] px-4 py-3 text-sm font-semibold uppercase tracking-wide text-[var(--color-text-secondary)] transition-colors hover:bg-[var(--color-border)]"
            >
              Advanced
              <ChevronDown
                size={16}
                className={`transition-transform ${xAdvancedOpen ? 'rotate-180' : ''}`}
              />
            </button>
            {xAdvancedOpen && (
              <div className="mt-4 flex flex-col gap-4 pl-1">
                <MaskedInput
                  label="API Key"
                  value={form.x_api_key}
                  onChange={(v) => setForm((f) => ({ ...f, x_api_key: v }))}
                />
                <MaskedInput
                  label="API Secret"
                  value={form.x_api_secret}
                  onChange={(v) => setForm((f) => ({ ...f, x_api_secret: v }))}
                />
                <MaskedInput
                  label="Client ID"
                  value={form.x_client_id}
                  onChange={(v) => setForm((f) => ({ ...f, x_client_id: v }))}
                />
                <MaskedInput
                  label="Client Secret"
                  value={form.x_client_secret}
                  onChange={(v) => setForm((f) => ({ ...f, x_client_secret: v }))}
                />
                <a
                  href={`${BASE_URL}/oauth/x/login`}
                  className="w-fit rounded-[var(--radius-button)] border border-[var(--color-border)] px-4 py-2 text-sm font-medium text-[var(--color-text)] transition-colors hover:bg-[var(--color-card)]"
                >
                  Connect X (OAuth)
                </a>
              </div>
            )}
          </section>
        </>
      )}

      {/* LinkedIn tab */}
      {activeTab === 'linkedin' && (
        <>
          <section className="mb-8">
            <h3 className="mb-4 text-sm font-semibold uppercase tracking-wide text-[var(--color-text-secondary)]">
              LinkedIn Cookie Auth (Linkitin)
            </h3>
            <div className="mb-2 flex items-center gap-2">
              {config?.linkedin.has_linkitin_auth ? (
                <span className="rounded-full bg-green-500/15 px-3 py-1 text-xs font-medium text-green-400">
                  Connected
                </span>
              ) : (
                <span className="rounded-full bg-gray-500/15 px-3 py-1 text-xs font-medium text-gray-400">
                  Not Set Up
                </span>
              )}
            </div>
            <p className="mb-3 text-xs text-[var(--color-text-secondary)]">
              Synced via Browser Cookie Sync above. Required for posting via linkitin fallback.
            </p>
            <div className="flex flex-col gap-3">
              <MaskedInput
                label="li_at"
                value={liCookieForm.li_at}
                onChange={() => {}}
              />
              <MaskedInput
                label="JSESSIONID"
                value={liCookieForm.jsessionid}
                onChange={() => {}}
              />
            </div>
          </section>

          <section className="mb-8">
            <h3 className="mb-4 text-sm font-semibold uppercase tracking-wide text-[var(--color-text-secondary)]">
              LinkedIn Niche Tags
            </h3>
            <NicheSelector
              selected={form.linkedin_niches}
              allNiches={allNiches}
              onChange={(niches) => setForm((f) => ({ ...f, linkedin_niches: niches }))}
              onAddNiche={(tag) =>
                setForm((f) => ({
                  ...f,
                  linkedin_niches: f.linkedin_niches.includes(tag)
                    ? f.linkedin_niches
                    : [...f.linkedin_niches, tag],
                }))
              }
            />
          </section>

          {persona && activePlatform === 'linkedin' && (
            <section className="mb-8">
              <h3 className="mb-4 text-sm font-semibold uppercase tracking-wide text-[var(--color-text-secondary)]">
                Persona (linkedin)
              </h3>
              <div className="rounded-[var(--radius-card)] border border-[var(--color-border)] bg-[var(--color-card)] p-4 text-sm text-[var(--color-text)]">
                <p className="mb-2">{persona.profile.voice_summary}</p>
                <div className="mt-3 grid grid-cols-2 gap-2 text-xs text-[var(--color-text-secondary)]">
                  <div>Tone: {persona.profile.writing_tone}</div>
                  <div>Length: {persona.profile.typical_length}</div>
                  <div>Vocab: {persona.profile.vocabulary_level}</div>
                  <div>Emoji: {persona.profile.emoji_usage}</div>
                </div>
              </div>
            </section>
          )}

          <section className="mb-8">
            <button
              type="button"
              onClick={() => setLiAdvancedOpen((o) => !o)}
              className="flex w-full items-center justify-between rounded-[var(--radius-card)] border border-[var(--color-border)] bg-[var(--color-card)] px-4 py-3 text-sm font-semibold uppercase tracking-wide text-[var(--color-text-secondary)] transition-colors hover:bg-[var(--color-border)]"
            >
              Advanced
              <ChevronDown
                size={16}
                className={`transition-transform ${liAdvancedOpen ? 'rotate-180' : ''}`}
              />
            </button>
            {liAdvancedOpen && (
              <div className="mt-4 flex flex-col gap-4 pl-1">
                <MaskedInput
                  label="Client ID"
                  value={form.linkedin_client_id}
                  onChange={(v) => setForm((f) => ({ ...f, linkedin_client_id: v }))}
                />
                <MaskedInput
                  label="Client Secret"
                  value={form.linkedin_client_secret}
                  onChange={(v) => setForm((f) => ({ ...f, linkedin_client_secret: v }))}
                />
                <a
                  href={`${BASE_URL}/oauth/linkedin/login`}
                  className="w-fit rounded-[var(--radius-button)] border border-[var(--color-border)] px-4 py-2 text-sm font-medium text-[var(--color-text)] transition-colors hover:bg-[var(--color-card)]"
                >
                  Connect LinkedIn (OAuth)
                </a>
              </div>
            )}
          </section>
        </>
      )}

      {/* GitHub tab */}
      {activeTab === 'github' && (
        <section className="mb-8">
          <h3 className="mb-4 text-sm font-semibold uppercase tracking-wide text-[var(--color-text-secondary)]">
            GitHub
          </h3>
          <div className="flex flex-col gap-4">
            <MaskedInput
              label="Personal Access Token"
              value={form.github_pat}
              onChange={(v) => setForm((f) => ({ ...f, github_pat: v }))}
            />
            <p className="text-xs text-[var(--color-text-secondary)]">
              Used to read repository commits for the Code to Post feature. Requires repo read scope.
            </p>
          </div>
        </section>
      )}

      {/* Telegram tab */}
      {activeTab === 'telegram' && (
        <section className="mb-8">
          <div className="mb-4 flex items-center justify-between">
            <h3 className="text-sm font-semibold uppercase tracking-wide text-[var(--color-text-secondary)]">
              Telegram Bot
            </h3>
            {daemonConfig?.telegram.connected ? (
              <span className="rounded-full bg-green-500/15 px-3 py-1 text-xs font-medium text-green-400">
                Connected
              </span>
            ) : (
              <span className="rounded-full bg-gray-500/15 px-3 py-1 text-xs font-medium text-gray-400">
                Not connected
              </span>
            )}
          </div>
          <div className="flex flex-col gap-4">
            <MaskedInput
              label="Bot Token"
              value={telegramForm.bot_token}
              onChange={(v) => setTelegramForm((f) => ({ ...f, bot_token: v }))}
            />
            <div>
              <label className="mb-1 block text-sm font-medium text-[var(--color-text)]">
                Chat ID
              </label>
              <input
                type="number"
                value={telegramForm.chat_id || ''}
                onChange={(e) =>
                  setTelegramForm((f) => ({ ...f, chat_id: Number(e.target.value) }))
                }
                className="w-full rounded-[var(--radius-input)] border border-[var(--color-border)] bg-[var(--color-card)] px-3 py-2 text-sm text-[var(--color-text)]"
              />
            </div>
            <div>
              <label className="mb-1 block text-sm font-medium text-[var(--color-text)]">
                Webhook URL
              </label>
              <input
                type="url"
                value={telegramForm.webhook_url}
                onChange={(e) =>
                  setTelegramForm((f) => ({ ...f, webhook_url: e.target.value }))
                }
                placeholder="https://..."
                className="w-full rounded-[var(--radius-input)] border border-[var(--color-border)] bg-[var(--color-card)] px-3 py-2 text-sm text-[var(--color-text)] placeholder:text-[var(--color-text-secondary)]"
              />
            </div>
          </div>
          <div className="mt-4">
            <button
              onClick={handleSaveTelegram}
              disabled={updateDaemonConfig.isPending}
              className="rounded-[var(--radius-button)] border border-[var(--color-border)] px-4 py-2 text-sm font-medium text-[var(--color-text)] transition-colors hover:bg-[var(--color-card)] disabled:opacity-50"
            >
              {updateDaemonConfig.isPending ? 'Saving...' : 'Save Telegram'}
            </button>
          </div>
        </section>
      )}

      <button
        onClick={handleSave}
        disabled={updateConfig.isPending}
        className="rounded-[var(--radius-button)] bg-[var(--color-accent)] px-6 py-2.5 text-sm font-medium text-white transition-colors hover:bg-[var(--color-accent-hover)] disabled:opacity-50"
      >
        {updateConfig.isPending ? 'Saving...' : 'Save Settings'}
      </button>
    </div>
  )
}
