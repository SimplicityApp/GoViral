import { useState, useEffect } from 'react'
import { Eye, EyeOff } from 'lucide-react'
import { toast } from 'sonner'
import { useDaemonConfigQuery, useUpdateDaemonConfigMutation } from '@/hooks/useDaemon'
import type { DaemonConfig } from '@/lib/types'
import { LoadingSpinner } from '@/components/shared/LoadingSpinner'

const PLATFORMS = ['x', 'linkedin']
const PERIOD_OPTIONS = ['day', 'week', 'month']

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
      <label className="mb-1 block text-sm font-medium text-[var(--color-text)]">{label}</label>
      <div className="flex items-center gap-2">
        <input
          type={visible ? 'text' : 'password'}
          value={value}
          onChange={(e) => onChange(e.target.value)}
          className="flex-1 rounded-[var(--radius-input)] border border-[var(--color-border)] bg-[var(--color-card)] px-3 py-2 text-sm text-[var(--color-text)]"
        />
        <button
          type="button"
          onClick={() => setVisible((v) => !v)}
          className="text-[var(--color-text-secondary)] hover:text-[var(--color-text)]"
        >
          {visible ? <EyeOff size={16} /> : <Eye size={16} />}
        </button>
      </div>
    </div>
  )
}

interface FormState {
  enabled: boolean
  schedules: Record<string, string>
  max_per_batch: number
  auto_skip_after: string
  trending_limit: number
  min_likes: number
  period: string
  bot_token: string
  chat_id: number
  webhook_url: string
}

function configToForm(config: DaemonConfig): FormState {
  return {
    enabled: config.daemon.enabled,
    schedules: { ...config.daemon.schedules },
    max_per_batch: config.daemon.max_per_batch,
    auto_skip_after: config.daemon.auto_skip_after,
    trending_limit: config.daemon.trending_limit,
    min_likes: config.daemon.min_likes,
    period: config.daemon.period,
    bot_token: config.telegram.bot_token,
    chat_id: config.telegram.chat_id,
    webhook_url: config.telegram.webhook_url,
  }
}

export function DaemonConfigPanel() {
  const { data: config, isLoading } = useDaemonConfigQuery()
  const updateConfig = useUpdateDaemonConfigMutation()

  const [form, setForm] = useState<FormState>({
    enabled: false,
    schedules: {},
    max_per_batch: 3,
    auto_skip_after: '2h',
    trending_limit: 10,
    min_likes: 100,
    period: 'day',
    bot_token: '',
    chat_id: 0,
    webhook_url: '',
  })

  useEffect(() => {
    if (config) {
      setForm(configToForm(config))
    }
  }, [config])

  const handleSave = () => {
    const payload: Partial<DaemonConfig> = {
      daemon: {
        enabled: form.enabled,
        schedules: form.schedules,
        max_per_batch: form.max_per_batch,
        auto_skip_after: form.auto_skip_after,
        trending_limit: form.trending_limit,
        min_likes: form.min_likes,
        period: form.period,
      },
      telegram: {
        bot_token: form.bot_token,
        chat_id: form.chat_id,
        webhook_url: form.webhook_url,
        connected: config?.telegram.connected ?? false,
      },
    }
    updateConfig.mutate(payload, {
      onSuccess: () => toast.success('Daemon config saved'),
      onError: () => toast.error('Failed to save daemon config'),
    })
  }

  if (isLoading) return <LoadingSpinner />

  return (
    <div className="rounded-[var(--radius-card)] border border-[var(--color-border)] bg-[var(--color-card)] p-5">
      <h3 className="mb-5 text-sm font-semibold text-[var(--color-text)]">Daemon Configuration</h3>

      <div className="mb-6 flex flex-col gap-5">
        {/* Enabled toggle */}
        <div className="flex items-center justify-between">
          <label className="text-sm font-medium text-[var(--color-text)]">Enabled</label>
          <button
            type="button"
            role="switch"
            aria-checked={form.enabled}
            onClick={() => setForm((f) => ({ ...f, enabled: !f.enabled }))}
            className={`relative h-5 w-9 rounded-full transition-colors ${form.enabled ? 'bg-[var(--color-accent)]' : 'bg-[var(--color-border)]'}`}
          >
            <span
              className={`absolute top-0.5 left-0.5 h-4 w-4 rounded-full bg-white shadow transition-transform ${form.enabled ? 'translate-x-4' : 'translate-x-0'}`}
            />
          </button>
        </div>

        {/* Schedules */}
        <div>
          <label className="mb-2 block text-sm font-semibold uppercase tracking-wide text-[var(--color-text-secondary)]">
            Cron Schedules
          </label>
          <div className="flex flex-col gap-3">
            {PLATFORMS.map((platform) => (
              <div key={platform}>
                <label className="mb-1 block text-sm font-medium capitalize text-[var(--color-text)]">
                  {platform}
                </label>
                <input
                  type="text"
                  value={form.schedules[platform] ?? ''}
                  onChange={(e) =>
                    setForm((f) => ({
                      ...f,
                      schedules: { ...f.schedules, [platform]: e.target.value },
                    }))
                  }
                  placeholder="0 9 * * *"
                  className="w-full rounded-[var(--radius-input)] border border-[var(--color-border)] bg-[var(--color-bg)] px-3 py-2 font-mono text-sm text-[var(--color-text)] placeholder:text-[var(--color-text-secondary)]"
                />
              </div>
            ))}
          </div>
        </div>

        {/* Numeric fields */}
        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="mb-1 block text-sm font-medium text-[var(--color-text)]">
              Max per batch
            </label>
            <input
              type="number"
              min={1}
              value={form.max_per_batch}
              onChange={(e) =>
                setForm((f) => ({ ...f, max_per_batch: Number(e.target.value) }))
              }
              className="w-full rounded-[var(--radius-input)] border border-[var(--color-border)] bg-[var(--color-bg)] px-3 py-2 text-sm text-[var(--color-text)]"
            />
          </div>
          <div>
            <label className="mb-1 block text-sm font-medium text-[var(--color-text)]">
              Trending limit
            </label>
            <input
              type="number"
              min={1}
              value={form.trending_limit}
              onChange={(e) =>
                setForm((f) => ({ ...f, trending_limit: Number(e.target.value) }))
              }
              className="w-full rounded-[var(--radius-input)] border border-[var(--color-border)] bg-[var(--color-bg)] px-3 py-2 text-sm text-[var(--color-text)]"
            />
          </div>
          <div>
            <label className="mb-1 block text-sm font-medium text-[var(--color-text)]">
              Min likes
            </label>
            <input
              type="number"
              min={0}
              value={form.min_likes}
              onChange={(e) =>
                setForm((f) => ({ ...f, min_likes: Number(e.target.value) }))
              }
              className="w-full rounded-[var(--radius-input)] border border-[var(--color-border)] bg-[var(--color-bg)] px-3 py-2 text-sm text-[var(--color-text)]"
            />
          </div>
          <div>
            <label className="mb-1 block text-sm font-medium text-[var(--color-text)]">
              Auto-skip after
            </label>
            <input
              type="text"
              value={form.auto_skip_after}
              onChange={(e) => setForm((f) => ({ ...f, auto_skip_after: e.target.value }))}
              placeholder="2h"
              className="w-full rounded-[var(--radius-input)] border border-[var(--color-border)] bg-[var(--color-bg)] px-3 py-2 text-sm text-[var(--color-text)] placeholder:text-[var(--color-text-secondary)]"
            />
          </div>
        </div>

        {/* Period */}
        <div>
          <label className="mb-1 block text-sm font-medium text-[var(--color-text)]">Period</label>
          <select
            value={form.period}
            onChange={(e) => setForm((f) => ({ ...f, period: e.target.value }))}
            className="w-full rounded-[var(--radius-input)] border border-[var(--color-border)] bg-[var(--color-bg)] px-3 py-2 text-sm text-[var(--color-text)]"
          >
            {PERIOD_OPTIONS.map((p) => (
              <option key={p} value={p} className="capitalize">
                {p}
              </option>
            ))}
          </select>
        </div>
      </div>

      {/* Telegram section */}
      <div className="mb-6 border-t border-[var(--color-border)] pt-5">
        <div className="mb-4 flex items-center justify-between">
          <h3 className="text-sm font-semibold text-[var(--color-text)]">Telegram</h3>
          {config?.telegram.connected ? (
            <span className="rounded-full bg-green-500/15 px-3 py-0.5 text-xs font-medium text-green-400">
              Connected
            </span>
          ) : (
            <span className="rounded-full bg-gray-500/15 px-3 py-0.5 text-xs font-medium text-gray-400">
              Not connected
            </span>
          )}
        </div>
        <div className="flex flex-col gap-4">
          <MaskedInput
            label="Bot Token"
            value={form.bot_token}
            onChange={(v) => setForm((f) => ({ ...f, bot_token: v }))}
          />
          <div>
            <label className="mb-1 block text-sm font-medium text-[var(--color-text)]">
              Chat ID
            </label>
            <input
              type="number"
              value={form.chat_id || ''}
              onChange={(e) => setForm((f) => ({ ...f, chat_id: Number(e.target.value) }))}
              className="w-full rounded-[var(--radius-input)] border border-[var(--color-border)] bg-[var(--color-bg)] px-3 py-2 text-sm text-[var(--color-text)]"
            />
          </div>
          <div>
            <label className="mb-1 block text-sm font-medium text-[var(--color-text)]">
              Webhook URL
            </label>
            <input
              type="url"
              value={form.webhook_url}
              onChange={(e) => setForm((f) => ({ ...f, webhook_url: e.target.value }))}
              placeholder="https://..."
              className="w-full rounded-[var(--radius-input)] border border-[var(--color-border)] bg-[var(--color-bg)] px-3 py-2 text-sm text-[var(--color-text)] placeholder:text-[var(--color-text-secondary)]"
            />
          </div>
        </div>
      </div>

      <button
        onClick={handleSave}
        disabled={updateConfig.isPending}
        className="rounded-[var(--radius-button)] bg-[var(--color-accent)] px-6 py-2.5 text-sm font-medium text-white transition-colors hover:bg-[var(--color-accent-hover)] disabled:opacity-50"
      >
        {updateConfig.isPending ? 'Saving...' : 'Save Config'}
      </button>
    </div>
  )
}
