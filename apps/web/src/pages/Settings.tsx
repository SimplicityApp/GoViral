import { useState, useEffect } from 'react'
import { useConfigQuery, useUpdateConfigMutation } from '@/hooks/useConfig'
import type { UpdateConfigPayload } from '@/hooks/useConfig'
import { usePersonaQuery } from '@/hooks/usePersona'
import { usePlatformStore } from '@/stores/platform-store'
import { LoadingSpinner } from '@/components/shared/LoadingSpinner'
import { Eye, EyeOff, X, Plus } from 'lucide-react'
import { toast } from 'sonner'

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

export function Settings() {
  const { activePlatform } = usePlatformStore()
  const { data: config, isLoading } = useConfigQuery()
  const { data: persona } = usePersonaQuery(activePlatform)
  const updateConfig = useUpdateConfigMutation()

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
    niches: [] as string[],
  })
  const [newTag, setNewTag] = useState('')

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
        niches: config.niches || [],
      })
    }
  }, [config])

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
      niches: form.niches,
    }

    updateConfig.mutate(payload, {
      onSuccess: () => toast.success('Settings saved'),
      onError: () => toast.error('Failed to save settings'),
    })
  }

  const addTag = () => {
    const tag = newTag.trim()
    if (tag && !form.niches.includes(tag)) {
      setForm((f) => ({ ...f, niches: [...f.niches, tag] }))
      setNewTag('')
    }
  }

  const removeTag = (tag: string) => {
    setForm((f) => ({ ...f, niches: f.niches.filter((t) => t !== tag) }))
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
        <div className="mt-3 flex gap-2">
          <a
            href="/api/v1/oauth/x/login"
            className="rounded-[var(--radius-button)] border border-[var(--color-border)] px-4 py-2 text-sm font-medium text-[var(--color-text)] transition-colors hover:bg-[var(--color-card)]"
          >
            Connect X
          </a>
          <a
            href="/api/v1/oauth/linkedin/login"
            className="rounded-[var(--radius-button)] border border-[var(--color-border)] px-4 py-2 text-sm font-medium text-[var(--color-text)] transition-colors hover:bg-[var(--color-card)]"
          >
            Connect LinkedIn
          </a>
        </div>
      </section>

      <section className="mb-8">
        <h3 className="mb-4 text-sm font-semibold uppercase tracking-wide text-[var(--color-text-secondary)]">
          Niche Tags
        </h3>
        <div className="mb-3 flex flex-wrap gap-2">
          {form.niches.map((tag) => (
            <span
              key={tag}
              className="flex items-center gap-1 rounded-full border border-[var(--color-border)] bg-[var(--color-card)] px-3 py-1 text-sm text-[var(--color-text)]"
            >
              {tag}
              <button
                onClick={() => removeTag(tag)}
                className="text-[var(--color-text-secondary)] hover:text-[var(--color-text)]"
              >
                <X size={14} />
              </button>
            </span>
          ))}
        </div>
        <div className="flex gap-2">
          <input
            type="text"
            placeholder="Add tag"
            value={newTag}
            onChange={(e) => setNewTag(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && addTag()}
            className="flex-1 rounded-[var(--radius-input)] border border-[var(--color-border)] bg-[var(--color-card)] px-3 py-2 text-sm text-[var(--color-text)] placeholder:text-[var(--color-text-secondary)]"
          />
          <button
            onClick={addTag}
            className="rounded-[var(--radius-button)] border border-[var(--color-border)] px-3 py-2 text-[var(--color-text-secondary)] hover:text-[var(--color-text)]"
          >
            <Plus size={16} />
          </button>
        </div>
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
