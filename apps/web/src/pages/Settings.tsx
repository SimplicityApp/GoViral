import { useState, useEffect, useMemo } from 'react'
import { useConfigQuery, useUpdateConfigMutation } from '@/hooks/useConfig'
import type { UpdateConfigPayload } from '@/hooks/useConfig'
import { useDaemonConfigQuery, useUpdateDaemonConfigMutation } from '@/hooks/useDaemon'
import { usePersonaQuery } from '@/hooks/usePersona'
import { useExtensionCookies } from '@/hooks/useExtensionCookies'
import { usePlatformStore } from '@/stores/platform-store'
import { LoadingSpinner } from '@/components/shared/LoadingSpinner'
import { Eye, EyeOff, X, Plus, Puzzle, Download } from 'lucide-react'
import { toast } from 'sonner'
import { useQueryClient } from '@tanstack/react-query'
import { apiClient, BASE_URL } from '@/lib/api'

function MaskedInput({
  label,
  value,
  onChange,
}: {
  label: string
  value: string
  onChange: (v: string) => void
}) {
  const [visible, setVisible] = useState(false)
  return (
    <div>
      <label className="mb-1 block text-sm font-medium text-[var(--color-text)]">
        {label}
      </label>
      <div className="flex items-center gap-2">
        <input
          type={visible ? 'text' : 'password'}
          value={value}
          onChange={(e) => onChange(e.target.value)}
          className="flex-1 rounded-[var(--radius-input)] border border-[var(--color-border)] bg-[var(--color-card)] px-3 py-2 text-sm text-[var(--color-text)]"
        />
        <button
          type="button"
          onClick={() => setVisible(!visible)}
          className="text-[var(--color-text-secondary)] hover:text-[var(--color-text)]"
        >
          {visible ? <EyeOff size={16} /> : <Eye size={16} />}
        </button>
      </div>
    </div>
  )
}

function NicheSelector({
  selected,
  allNiches,
  onChange,
  onAddNiche,
}: {
  selected: string[]
  allNiches: string[]
  onChange: (niches: string[]) => void
  onAddNiche: (niche: string) => void
}) {
  const [open, setOpen] = useState(false)
  const [inputVal, setInputVal] = useState('')

  const toggle = (niche: string) => {
    onChange(
      selected.includes(niche)
        ? selected.filter((n) => n !== niche)
        : [...selected, niche],
    )
  }

  const handleAdd = () => {
    const tag = inputVal.trim()
    if (tag) {
      onAddNiche(tag)
      setInputVal('')
    }
  }

  return (
    <div className="relative">
      {/* Selected chips */}
      <div className="mb-3 flex flex-wrap gap-2">
        {selected.map((tag) => (
          <span
            key={tag}
            className="flex items-center gap-1 rounded-full border border-[var(--color-border)] bg-[var(--color-card)] px-3 py-1 text-sm text-[var(--color-text)]"
          >
            {tag}
            <button
              onClick={() => toggle(tag)}
              className="text-[var(--color-text-secondary)] hover:text-[var(--color-text)]"
            >
              <X size={14} />
            </button>
          </span>
        ))}
      </div>

      {/* Dropdown trigger */}
      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        className="flex items-center gap-2 rounded-[var(--radius-button)] border border-[var(--color-border)] px-3 py-2 text-sm text-[var(--color-text-secondary)] hover:text-[var(--color-text)] transition-colors"
      >
        <Plus size={16} /> Add niches
      </button>

      {/* Dropdown panel */}
      {open && (
        <div className="absolute z-10 mt-1 w-64 rounded-[var(--radius-card)] border border-[var(--color-border)] bg-[var(--color-card)] shadow-lg">
          <div className="max-h-48 overflow-y-auto p-2">
            {allNiches.length === 0 && (
              <p className="px-2 py-1.5 text-sm text-[var(--color-text-secondary)]">
                No niches yet — add one below
              </p>
            )}
            {allNiches.map((niche) => (
              <label
                key={niche}
                className="flex cursor-pointer items-center gap-2 rounded px-2 py-1.5 hover:bg-[var(--color-border)]"
              >
                <input
                  type="checkbox"
                  checked={selected.includes(niche)}
                  onChange={() => toggle(niche)}
                />
                <span className="text-sm text-[var(--color-text)]">{niche}</span>
              </label>
            ))}
          </div>
          {/* Add custom niche */}
          <div className="flex gap-2 border-t border-[var(--color-border)] p-2">
            <input
              type="text"
              placeholder="New niche..."
              value={inputVal}
              onChange={(e) => setInputVal(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && handleAdd()}
              className="flex-1 rounded-[var(--radius-input)] border border-[var(--color-border)] bg-[var(--color-card)] px-2 py-1 text-sm text-[var(--color-text)] placeholder:text-[var(--color-text-secondary)]"
            />
            <button
              type="button"
              onClick={handleAdd}
              className="rounded-[var(--radius-button)] border border-[var(--color-border)] px-2 py-1 text-[var(--color-text-secondary)] hover:text-[var(--color-text)]"
            >
              <Plus size={16} />
            </button>
          </div>
        </div>
      )}
    </div>
  )
}

export function Settings() {
  const { activePlatform } = usePlatformStore()
  const { data: config, isLoading } = useConfigQuery()
  const { data: persona } = usePersonaQuery(activePlatform)
  const updateConfig = useUpdateConfigMutation()
  const queryClient = useQueryClient()
  const { extension, extracting: extensionExtracting, extractCookies } = useExtensionCookies()
  const [xCookieForm, setXCookieForm] = useState({ auth_token: '', ct0: '' })
  const [savingXCookies, setSavingXCookies] = useState(false)
  const [liCookieForm, setLiCookieForm] = useState({ li_at: '', jsessionid: '' })
  const [savingLiCookies, setSavingLiCookies] = useState(false)

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

  const [serverApiKey, setServerApiKey] = useState(
    () => localStorage.getItem('goviral_api_key') || '',
  )

  const [form, setForm] = useState({
    claude_api_key: '',
    gemini_api_key: '',
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
    }
  }, [config])

  const allNiches = useMemo(
    () => [...new Set([...form.niches, ...form.linkedin_niches])],
    [form.niches, form.linkedin_niches],
  )

  const handleSave = () => {
    // Save server API key to localStorage (client-side only)
    if (serverApiKey) {
      localStorage.setItem('goviral_api_key', serverApiKey)
    } else {
      localStorage.removeItem('goviral_api_key')
    }

    // Build nested payload matching server's UpdateConfigRequest
    const payload: UpdateConfigPayload = {
      claude: { api_key: form.claude_api_key },
      gemini: { api_key: form.gemini_api_key },
      x: {
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
          Server
        </h3>
        <MaskedInput
          label="Server API Key"
          value={serverApiKey}
          onChange={setServerApiKey}
        />
        <p className="mt-1 text-xs text-[var(--color-text-secondary)]">
          Must match the api_key in your server config. Stored locally in your browser.
        </p>
      </section>

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
          X (Twitter) Credentials
        </h3>
        <div className="flex flex-col gap-4">
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
        </div>
      </section>

      <section className="mb-8">
        <h3 className="mb-4 text-sm font-semibold uppercase tracking-wide text-[var(--color-text-secondary)]">
          LinkedIn Credentials
        </h3>
        <div className="flex flex-col gap-4">
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
        </div>
        <div className="mt-3 flex items-center gap-2">
          <a
            href={`${BASE_URL}/oauth/x/login`}
            className="rounded-[var(--radius-button)] border border-[var(--color-border)] px-4 py-2 text-sm font-medium text-[var(--color-text)] transition-colors hover:bg-[var(--color-card)]"
          >
            Connect X (OAuth)
          </a>
          <a
            href={`${BASE_URL}/oauth/linkedin/login`}
            className="rounded-[var(--radius-button)] border border-[var(--color-border)] px-4 py-2 text-sm font-medium text-[var(--color-text)] transition-colors hover:bg-[var(--color-card)]"
          >
            Connect LinkedIn (OAuth)
          </a>
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
              <ol className="list-inside list-decimal space-y-1 text-xs">
                <li>Open <code className="rounded bg-[var(--color-border)] px-1">chrome://extensions</code></li>
                <li>Enable <strong>Developer mode</strong> (top right)</li>
                <li>Click <strong>Load unpacked</strong> and select the <code className="rounded bg-[var(--color-border)] px-1">apps/extension</code> folder</li>
                <li>Refresh this page</li>
              </ol>
            </div>
          </>
        )}
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
          Paste auth_token and ct0 from your browser&apos;s X cookies. Required for posting via twikit fallback.
        </p>
        <div className="flex flex-col gap-3">
          <MaskedInput
            label="auth_token"
            value={xCookieForm.auth_token}
            onChange={(v) => setXCookieForm((f) => ({ ...f, auth_token: v }))}
          />
          <MaskedInput
            label="ct0"
            value={xCookieForm.ct0}
            onChange={(v) => setXCookieForm((f) => ({ ...f, ct0: v }))}
          />
          <button
            onClick={async () => {
              if (!xCookieForm.auth_token || !xCookieForm.ct0) {
                toast.error('Both auth_token and ct0 are required')
                return
              }
              setSavingXCookies(true)
              try {
                await apiClient.post('/x/login-cookies', xCookieForm)
                toast.success('X cookies saved')
                setXCookieForm({ auth_token: '', ct0: '' })
                void queryClient.invalidateQueries({ queryKey: ['config'] })
              } catch {
                toast.error('Failed to save X cookies')
              } finally {
                setSavingXCookies(false)
              }
            }}
            disabled={savingXCookies}
            className="w-fit rounded-[var(--radius-button)] bg-[var(--color-accent)] px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-[var(--color-accent-hover)] disabled:opacity-50"
          >
            {savingXCookies ? 'Saving...' : 'Save X Cookies'}
          </button>
        </div>
      </section>

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
          Paste li_at and JSESSIONID from your browser&apos;s LinkedIn cookies. Required for posting via linkitin fallback.
        </p>
        <div className="flex flex-col gap-3">
          <MaskedInput
            label="li_at"
            value={liCookieForm.li_at}
            onChange={(v) => setLiCookieForm((f) => ({ ...f, li_at: v }))}
          />
          <MaskedInput
            label="JSESSIONID"
            value={liCookieForm.jsessionid}
            onChange={(v) => setLiCookieForm((f) => ({ ...f, jsessionid: v }))}
          />
          <button
            onClick={async () => {
              if (!liCookieForm.li_at || !liCookieForm.jsessionid) {
                toast.error('Both li_at and JSESSIONID are required')
                return
              }
              setSavingLiCookies(true)
              try {
                await apiClient.post('/linkedin/login-cookies', liCookieForm)
                toast.success('LinkedIn cookies saved')
                setLiCookieForm({ li_at: '', jsessionid: '' })
                void queryClient.invalidateQueries({ queryKey: ['config'] })
              } catch {
                toast.error('Failed to save LinkedIn cookies')
              } finally {
                setSavingLiCookies(false)
              }
            }}
            disabled={savingLiCookies}
            className="w-fit rounded-[var(--radius-button)] bg-[var(--color-accent)] px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-[var(--color-accent-hover)] disabled:opacity-50"
          >
            {savingLiCookies ? 'Saving...' : 'Save LinkedIn Cookies'}
          </button>
        </div>
      </section>

      <section className="mb-8">
        <h3 className="mb-4 text-sm font-semibold uppercase tracking-wide text-[var(--color-text-secondary)]">
          GitHub
        </h3>
        <div className="flex flex-col gap-4">
          <MaskedInput
            label="GitHub Personal Access Token"
            value={form.github_pat}
            onChange={(v) => setForm((f) => ({ ...f, github_pat: v }))}
          />
          <p className="text-xs text-[var(--color-text-secondary)]">
            Used to read repository commits for the Code to Post feature. Requires repo read scope.
          </p>
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

      {persona && (
        <section className="mb-8">
          <h3 className="mb-4 text-sm font-semibold uppercase tracking-wide text-[var(--color-text-secondary)]">
            Persona ({activePlatform})
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
