import { Play, Square, Zap } from 'lucide-react'
import type { DaemonStatus } from '@/lib/types'

function formatTime(value: string | null): string {
  if (!value) return 'Never'
  const date = new Date(value)
  if (isNaN(date.getTime())) return 'Never'
  return date.toLocaleString(undefined, {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

interface DaemonStatusCardProps {
  status: DaemonStatus
  onStart: () => void
  onStop: () => void
  onRunNow: () => void
  isStarting: boolean
  isStopping: boolean
  isRunning: boolean
}

export function DaemonStatusCard({
  status,
  onStart,
  onStop,
  onRunNow,
  isStarting,
  isStopping,
  isRunning,
}: DaemonStatusCardProps) {
  return (
    <div className="rounded-[var(--radius-card)] border border-[var(--color-border)] bg-[var(--color-card)] p-5">
      <div className="mb-4 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <span
            className={`h-2.5 w-2.5 rounded-full ${status.running ? 'bg-green-400' : 'bg-red-400'}`}
          />
          <span className="text-sm font-semibold text-[var(--color-text)]">
            {status.running ? 'Running' : 'Stopped'}
          </span>
        </div>
        <div className="flex items-center gap-2">
          {status.running ? (
            <button
              onClick={onStop}
              disabled={isStopping}
              className="flex items-center gap-1.5 rounded-[var(--radius-button)] border border-[var(--color-border)] px-3 py-1.5 text-sm font-medium text-[var(--color-text)] transition-colors hover:bg-[var(--color-border)] disabled:opacity-50"
            >
              <Square size={14} />
              Stop
            </button>
          ) : (
            <button
              onClick={onStart}
              disabled={isStarting}
              className="flex items-center gap-1.5 rounded-[var(--radius-button)] border border-[var(--color-border)] px-3 py-1.5 text-sm font-medium text-[var(--color-text)] transition-colors hover:bg-[var(--color-border)] disabled:opacity-50"
            >
              <Play size={14} />
              Start
            </button>
          )}
          <button
            onClick={onRunNow}
            disabled={isRunning}
            className="flex items-center gap-1.5 rounded-[var(--radius-button)] bg-[var(--color-accent)] px-3 py-1.5 text-sm font-medium text-white transition-colors hover:bg-[var(--color-accent-hover)] disabled:opacity-50"
          >
            <Zap size={14} />
            Run Now
          </button>
        </div>
      </div>

      {Object.keys(status.platforms).length > 0 && (
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
          {Object.entries(status.platforms).map(([platform, info]) => (
            <div
              key={platform}
              className="rounded-[var(--radius-button)] border border-[var(--color-border)] bg-[var(--color-bg)] px-4 py-3"
            >
              <div className="mb-2 text-xs font-semibold uppercase tracking-wide text-[var(--color-text-secondary)]">
                {platform}
              </div>
              <div className="flex flex-col gap-1 text-xs text-[var(--color-text-secondary)]">
                <div className="flex items-center justify-between">
                  <span>Schedule</span>
                  <span className="font-mono text-[var(--color-text)]">
                    {info.schedule || 'Not set'}
                  </span>
                </div>
                <div className="flex items-center justify-between">
                  <span>Next run</span>
                  <span className="text-[var(--color-text)]">{formatTime(info.next_run)}</span>
                </div>
                <div className="flex items-center justify-between">
                  <span>Last run</span>
                  <span className="text-[var(--color-text)]">{formatTime(info.last_run)}</span>
                </div>
                {info.next_digest && (
                  <div className="flex items-center justify-between">
                    <span>Next digest</span>
                    <span className="text-amber-400">{formatTime(info.next_digest)}</span>
                  </div>
                )}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
