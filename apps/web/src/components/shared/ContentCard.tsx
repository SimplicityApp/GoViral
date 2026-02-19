import type { GeneratedContent } from '@/lib/types'
import { StatusBadge } from './StatusBadge'
import { formatRelativeTime } from '@/lib/format'
import { Check, CheckSquare, Pencil, Send, Square, Trash2 } from 'lucide-react'

interface ContentCardProps {
  content: GeneratedContent
  onStatusChange?: (status: 'draft' | 'approved' | 'posted') => void
  onEdit?: () => void
  onDelete?: () => void
  isSelected?: boolean
  onToggleSelect?: () => void
}

export function ContentCard({ content, onStatusChange, onEdit, onDelete, isSelected, onToggleSelect }: ContentCardProps) {
  const selectionMode = onToggleSelect !== undefined
  return (
    <div
      onClick={selectionMode ? onToggleSelect : undefined}
      className={`rounded-[var(--radius-card)] border bg-[var(--color-card)] p-4 transition-colors ${
        selectionMode ? 'cursor-pointer' : ''
      } ${
        isSelected
          ? 'border-[var(--color-accent)] ring-1 ring-[var(--color-accent)]'
          : 'border-[var(--color-border)]'
      }`}
    >
      <div className="mb-3 flex items-center justify-between">
        <div className="flex items-center gap-2">
          {selectionMode && (
            isSelected
              ? <CheckSquare size={16} className="shrink-0 text-[var(--color-accent)]" />
              : <Square size={16} className="shrink-0 text-[var(--color-text-secondary)]" />
          )}
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
        {!selectionMode && content.status === 'draft' && onStatusChange && (
          <button
            onClick={() => onStatusChange('approved')}
            className="flex items-center gap-1.5 rounded-[var(--radius-button)] bg-[var(--color-accent)] px-3 py-1.5 text-xs font-medium text-white transition-colors hover:bg-[var(--color-accent-hover)]"
          >
            <Check size={14} />
            Approve
          </button>
        )}
        {!selectionMode && onEdit && content.status !== 'posted' && (
          <button
            onClick={onEdit}
            className="flex items-center gap-1.5 rounded-[var(--radius-button)] border border-[var(--color-border)] px-3 py-1.5 text-xs font-medium text-[var(--color-text-secondary)] transition-colors hover:text-[var(--color-text)]"
          >
            <Pencil size={14} />
            Edit
          </button>
        )}
        {!selectionMode && content.status === 'approved' && onStatusChange && (
          <button
            onClick={() => onStatusChange('posted')}
            className="flex items-center gap-1.5 rounded-[var(--radius-button)] bg-[var(--color-accent)] px-3 py-1.5 text-xs font-medium text-white transition-colors hover:bg-[var(--color-accent-hover)]"
          >
            <Send size={14} />
            Publish
          </button>
        )}
        {!selectionMode && onDelete && (
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
