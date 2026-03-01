import { Check, X, Clock, MessageSquare } from 'lucide-react'
import type { DaemonBatch } from '@/lib/types'

const statusStyles: Record<string, string> = {
  pending: 'bg-yellow-500/20 text-yellow-400',
  notified: 'bg-blue-500/20 text-blue-400',
  awaiting_reply: 'bg-blue-500/20 text-blue-400',
  approved: 'bg-green-500/20 text-green-400',
  rejected: 'bg-red-500/20 text-red-400',
  posted: 'bg-green-500/20 text-green-400',
  scheduled: 'bg-purple-500/20 text-purple-400',
  archived: 'bg-gray-500/20 text-gray-400',
  failed: 'bg-red-500/20 text-red-400',
}

const CONTENT_PREVIEW_LENGTH = 200

function formatDate(value: string): string {
  const date = new Date(value)
  if (isNaN(date.getTime())) return ''
  return date.toLocaleString(undefined, {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

const actionableStatuses = new Set<DaemonBatch['status']>(['pending', 'notified', 'awaiting_reply'])

interface BatchCardProps {
  batch: DaemonBatch
  onApprove: (id: number) => void
  onReject: (id: number) => void
  isActing: boolean
}

export function BatchCard({ batch, onApprove, onReject, isActing }: BatchCardProps) {
  const isActionable = actionableStatuses.has(batch.status)
  const isPosted = batch.status === 'posted'

  return (
    <div className="rounded-[var(--radius-card)] border border-[var(--color-border)] bg-[var(--color-card)] p-4">
      <div className="mb-3 flex items-center justify-between gap-2">
        <div className="flex items-center gap-2">
          <span className="rounded-full border border-[var(--color-border)] bg-[var(--color-bg)] px-2.5 py-0.5 text-xs font-medium capitalize text-[var(--color-text)]">
            {batch.platform}
          </span>
          <span
            className={`rounded-full px-2.5 py-0.5 text-xs font-medium capitalize ${statusStyles[batch.status] ?? 'bg-gray-500/20 text-gray-400'}`}
          >
            {batch.status.replace(/_/g, ' ')}
          </span>
          {batch.batch_type === 'digest' && (
            <span className="rounded-full bg-amber-500/20 px-2.5 py-0.5 text-xs font-medium text-amber-400">
              Nightly Digest
            </span>
          )}
          {batch.telegram_message_id > 0 && (
            <span className="flex items-center gap-1 text-xs text-[var(--color-text-secondary)]">
              <MessageSquare size={12} />
              Telegram
            </span>
          )}
        </div>
        <div className="flex items-center gap-1 text-xs text-[var(--color-text-secondary)]">
          <Clock size={12} />
          {formatDate(batch.created_at)}
        </div>
      </div>

      {batch.contents && batch.contents.length > 0 && (
        <div className="mb-3 flex flex-col gap-2">
          {batch.contents.map((content) => (
            <p
              key={content.id}
              className="rounded-[var(--radius-button)] border border-[var(--color-border)] bg-[var(--color-bg)] px-3 py-2 text-sm text-[var(--color-text)]"
            >
              {content.generated_content.length > CONTENT_PREVIEW_LENGTH
                ? content.generated_content.slice(0, CONTENT_PREVIEW_LENGTH) + '...'
                : content.generated_content}
            </p>
          ))}
        </div>
      )}

      {batch.error_message && (
        <p className="mb-3 text-xs text-red-400">{batch.error_message}</p>
      )}

      <div className="flex items-center justify-end gap-2">
        {isPosted && (
          <span className="flex items-center gap-1 text-xs text-green-400">
            <Check size={14} />
            Posted
          </span>
        )}
        {isActionable && (
          <>
            <button
              onClick={() => onReject(batch.id)}
              disabled={isActing}
              className="flex items-center gap-1.5 rounded-[var(--radius-button)] border border-[var(--color-border)] px-3 py-1.5 text-sm font-medium text-[var(--color-text-secondary)] transition-colors hover:border-red-400 hover:text-red-400 disabled:opacity-50"
            >
              <X size={14} />
              Reject
            </button>
            <button
              onClick={() => onApprove(batch.id)}
              disabled={isActing}
              className="flex items-center gap-1.5 rounded-[var(--radius-button)] bg-[var(--color-accent)] px-3 py-1.5 text-sm font-medium text-white transition-colors hover:bg-[var(--color-accent-hover)] disabled:opacity-50"
            >
              <Check size={14} />
              Approve
            </button>
          </>
        )}
      </div>
    </div>
  )
}
