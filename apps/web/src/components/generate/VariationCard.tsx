import { useState, useEffect } from 'react'
import type { GeneratedContent } from '@/lib/types'
import { BASE_URL } from '@/lib/api'
import { StatusBadge } from '@/components/shared/StatusBadge'
import { Check, X, Pencil } from 'lucide-react'

interface VariationCardProps {
  content: GeneratedContent
  onApprove: () => void
  onReject: () => void
  onEdit: (text: string, description?: string) => void
}

function CodeImagePreview({ contentId, commitId }: { contentId: number; commitId: number }) {
  const [blobUrl, setBlobUrl] = useState<string | null>(null)
  const [errored, setErrored] = useState(false)

  useEffect(() => {
    const apiKey = localStorage.getItem('goviral_api_key')
    const headers: Record<string, string> = {}
    if (apiKey) {
      headers['Authorization'] = `Bearer ${apiKey}`
    }

    // Prefer content-specific image (AI-selected snippet), fall back to commit-level
    const url = contentId > 0
      ? `${BASE_URL}/content/${contentId}/code-image`
      : `${BASE_URL}/repos/commits/${commitId}/image`

    fetch(url, { headers })
      .then((res) => {
        if (!res.ok) throw new Error('failed')
        return res.blob()
      })
      .then((blob) => {
        setBlobUrl(URL.createObjectURL(blob))
      })
      .catch(() => setErrored(true))

    return () => {
      setBlobUrl((prev) => {
        if (prev) URL.revokeObjectURL(prev)
        return null
      })
    }
  }, [contentId, commitId])

  if (errored) return null

  return (
    <div className="mb-4">
      {!blobUrl && (
        <div className="h-32 animate-pulse rounded-lg bg-[var(--color-border)]" />
      )}
      {blobUrl && (
        <img
          src={blobUrl}
          alt="Code diff"
          className="w-full rounded-lg border border-[var(--color-border)]"
        />
      )}
    </div>
  )
}

export function VariationCard({ content, onApprove, onReject, onEdit }: VariationCardProps) {
  const [editing, setEditing] = useState(false)
  const [editText, setEditText] = useState(content.generated_content)
  const [editDescription, setEditDescription] = useState(content.code_image_description ?? '')

  const handleSave = () => {
    onEdit(editText, editDescription || undefined)
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
          {content.source_type === 'commit' && content.source_commit_id > 0 && (
            <div className="mt-2">
              <label className="mb-1 block text-xs text-[var(--color-text-secondary)]">
                Image description (max 80 chars)
              </label>
              <input
                type="text"
                value={editDescription}
                onChange={(e) => setEditDescription(e.target.value.slice(0, 80))}
                maxLength={80}
                placeholder="e.g. Added retry logic with exponential backoff"
                className="w-full rounded-[var(--radius-input)] border border-[var(--color-border)] bg-[var(--color-bg)] px-3 py-1.5 text-sm text-[var(--color-text)]"
              />
            </div>
          )}
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
                setEditDescription(content.code_image_description ?? '')
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

      {content.source_type === 'commit' && content.source_commit_id > 0 && (
        <>
          <CodeImagePreview contentId={content.id} commitId={content.source_commit_id} />
          {!editing && content.code_image_description && (
            <p className="-mt-2 mb-4 text-xs text-[var(--color-text-secondary)]">
              {content.code_image_description}
            </p>
          )}
        </>
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
