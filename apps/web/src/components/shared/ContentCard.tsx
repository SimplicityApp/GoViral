import type { GeneratedContent } from '@/lib/types'
import { StatusBadge } from './StatusBadge'
import { formatRelativeTime } from '@/lib/format'
import { Check, Pencil, Send, Trash2 } from 'lucide-react'

interface ContentCardProps {
  content: GeneratedContent
  onStatusChange?: (status: 'draft' | 'approved' | 'posted') => void
  onEdit?: () => void
  onDelete?: () => void
}

export function ContentCard({ content, onStatusChange, onEdit, onDelete }: ContentCardProps) {
  return (
    <div className="rounded-[var(--radius-card)] border border-[var(--color-border)] bg-[var(--color-card)] p-4">
      <div className="mb-3 flex items-center justify-between">
        <div className="flex items-center gap-2">
          <StatusBadge status={content.status} />
          {content.is_repost && (
            <span className="rounded-full bg-cyan-500/10 px-2 py-0.5 text-xs font-medium text-cyan-400">
              REPOST
            </span>
          )}
        </div>
        <span className="text-xs text-[var(--color-text-secondary)]">
          {formatRelativeTime(content.created_at)}
        </span>
      </div>

      <p className="mb-4 whitespace-pre-wrap text-sm text-[var(--color-text)]">
        {content.generated_content}
      </p>

      <div className="flex items-center gap-2">
        {content.status === 'draft' && onStatusChange && (
          <button
            onClick={() => onStatusChange('approved')}
            className="flex items-center gap-1.5 rounded-[var(--radius-button)] bg-[var(--color-accent)] px-3 py-1.5 text-xs font-medium text-white transition-colors hover:bg-[var(--color-accent-hover)]"
          >
            <Check size={14} />
            Approve
          </button>
        )}
        {onEdit && content.status !== 'posted' && (
          <button
            onClick={onEdit}
            className="flex items-center gap-1.5 rounded-[var(--radius-button)] border border-[var(--color-border)] px-3 py-1.5 text-xs font-medium text-[var(--color-text-secondary)] transition-colors hover:text-[var(--color-text)]"
          >
            <Pencil size={14} />
            Edit
          </button>
        )}
        {content.status === 'approved' && onStatusChange && (
          <button
            onClick={() => onStatusChange('posted')}
            className="flex items-center gap-1.5 rounded-[var(--radius-button)] bg-[var(--color-accent)] px-3 py-1.5 text-xs font-medium text-white transition-colors hover:bg-[var(--color-accent-hover)]"
          >
            <Send size={14} />
            Publish
          </button>
        )}
        {onDelete && (
          <>
            <div className="ml-auto" />
            <button
              onClick={onDelete}
              className="flex items-center gap-1.5 rounded-[var(--radius-button)] border border-[var(--color-border)] px-3 py-1.5 text-xs text-red-400 transition-colors hover:text-red-300"
            >
              <Trash2 size={14} />
              Delete
            </button>
          </>
        )}
      </div>
    </div>
  )
}
