import { useState } from 'react'
import type { GeneratedContent } from '@/lib/types'
import { StatusBadge } from '@/components/shared/StatusBadge'
import { Check, X, Pencil } from 'lucide-react'

interface VariationCardProps {
  content: GeneratedContent
  onApprove: () => void
  onReject: () => void
  onEdit: (text: string) => void
}

export function VariationCard({ content, onApprove, onReject, onEdit }: VariationCardProps) {
  const [editing, setEditing] = useState(false)
  const [editText, setEditText] = useState(content.generated_content)

  const handleSave = () => {
    onEdit(editText)
    setEditing(false)
  }

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
          {content.generated_content.length} chars
        </span>
      </div>

      {editing ? (
        <div className="mb-3">
          <textarea
            value={editText}
            onChange={(e) => setEditText(e.target.value)}
            rows={6}
            className="w-full resize-none rounded-[var(--radius-input)] border border-[var(--color-border)] bg-[var(--color-bg)] p-3 text-sm text-[var(--color-text)]"
          />
          <div className="mt-2 flex gap-2">
            <button
              onClick={handleSave}
              className="rounded-[var(--radius-button)] bg-[var(--color-accent)] px-3 py-1.5 text-xs font-medium text-white"
            >
              Save
            </button>
            <button
              onClick={() => {
                setEditText(content.generated_content)
                setEditing(false)
              }}
              className="rounded-[var(--radius-button)] border border-[var(--color-border)] px-3 py-1.5 text-xs text-[var(--color-text-secondary)]"
            >
              Cancel
            </button>
          </div>
        </div>
      ) : (
        <p className="mb-4 whitespace-pre-wrap text-sm text-[var(--color-text)]">
          {content.generated_content}
        </p>
      )}

      {!editing && content.status === 'draft' && (
        <div className="flex items-center gap-2">
          <button
            onClick={onApprove}
            className="flex items-center gap-1.5 rounded-[var(--radius-button)] bg-[var(--color-accent)] px-3 py-1.5 text-xs font-medium text-white transition-colors hover:bg-[var(--color-accent-hover)]"
          >
            <Check size={14} />
            Approve
          </button>
          <button
            onClick={() => setEditing(true)}
            className="flex items-center gap-1.5 rounded-[var(--radius-button)] border border-[var(--color-border)] px-3 py-1.5 text-xs text-[var(--color-text-secondary)] transition-colors hover:text-[var(--color-text)]"
          >
            <Pencil size={14} />
            Edit
          </button>
          <button
            onClick={onReject}
            className="flex items-center gap-1.5 rounded-[var(--radius-button)] border border-[var(--color-border)] px-3 py-1.5 text-xs text-red-400 transition-colors hover:text-red-300"
          >
            <X size={14} />
            Reject
          </button>
        </div>
      )}
    </div>
  )
}
