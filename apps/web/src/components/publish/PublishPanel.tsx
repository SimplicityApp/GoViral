import { useState, useEffect } from 'react'
import type { GeneratedContent } from '@/lib/types'
import { usePlatformParam } from '@/hooks/usePlatformParam'
import { usePublishMutation, useScheduleMutation } from '@/hooks/usePublish'
import { useUpdateContentMutation } from '@/hooks/useHistory'
import { ThreadPreview } from './ThreadPreview'
import { ScheduleCalendar } from './ScheduleCalendar'
import { Send, Clock, Save } from 'lucide-react'
import { toast } from 'sonner'

interface PublishPanelProps {
  items: GeneratedContent[]
  initialSelectedId?: number
}

export function PublishPanel({ items, initialSelectedId }: PublishPanelProps) {
  const activePlatform = usePlatformParam()
  const [selectedId, setSelectedId] = useState<number | null>(initialSelectedId ?? null)
  const [mode, setMode] = useState<'now' | 'schedule'>('now')
  const [scheduledAt, setScheduledAt] = useState('')
  const [editedContent, setEditedContent] = useState<string>('')

  const publishMutation = usePublishMutation(activePlatform)
  const scheduleMutation = useScheduleMutation()
  const updateContentMutation = useUpdateContentMutation()

  const selected = items.find((i) => i.id === selectedId)

  // Reset editedContent when selection changes
  useEffect(() => {
    setEditedContent(selected?.generated_content ?? '')
  }, [selected?.id, selected?.generated_content])

  const hasUnsavedChanges = selected && editedContent !== selected.generated_content

  const handleSave = () => {
    if (!selectedId || !hasUnsavedChanges) return
    updateContentMutation.mutate(
      { id: selectedId, generated_content: editedContent },
      {
        onSuccess: () => toast.success('Content saved'),
        onError: () => toast.error('Failed to save content'),
      },
    )
  }

  const handlePublish = () => {
    if (!selectedId) return
    if (mode === 'now') {
      publishMutation.mutate(
        { content_id: selectedId },
        {
          onSuccess: () => toast.success('Published successfully'),
          onError: () => toast.error('Failed to publish'),
        },
      )
    } else {
      if (!scheduledAt) {
        toast.error('Select a date and time')
        return
      }
      scheduleMutation.mutate(
        { content_id: selectedId, scheduled_at: new Date(scheduledAt).toISOString() },
        {
          onSuccess: () => toast.success('Scheduled successfully'),
          onError: () => toast.error('Failed to schedule'),
        },
      )
    }
  }

  return (
    <div className="flex flex-col gap-6">
      {/* Content selector */}
      <div>
        <label className="mb-2 block text-sm font-medium text-[var(--color-text)]">
          Select approved content
          {selected?.is_repost && (
            <span className="ml-2 rounded-full bg-cyan-500/10 px-2 py-0.5 text-xs font-medium text-cyan-400">
              {activePlatform === 'linkedin' ? 'Repost' : 'Quote Tweet'}
            </span>
          )}
        </label>
        <div className="flex flex-col gap-2">
          {items.map((item) => (
            <button
              key={item.id}
              onClick={() => setSelectedId(item.id)}
              className={`rounded-[var(--radius-card)] border p-3 text-left text-sm transition-colors ${
                selectedId === item.id
                  ? 'border-[var(--color-accent)] bg-[var(--color-card)]'
                  : 'border-[var(--color-border)] bg-[var(--color-card)] hover:bg-[var(--color-card-hover)]'
              }`}
            >
              <p className="line-clamp-2 text-[var(--color-text)]">
                {item.generated_content}
              </p>
            </button>
          ))}
        </div>
      </div>

      {/* Editable preview */}
      {selected && (
        <>
          <ThreadPreview
            content={editedContent}
            editable={true}
            onChange={setEditedContent}
            isRepost={selected?.is_repost}
            quoteTweetId={selected?.quote_tweet_id}
          />
          {hasUnsavedChanges && (
            <button
              onClick={handleSave}
              disabled={updateContentMutation.isPending}
              className="flex items-center justify-center gap-1.5 rounded-[var(--radius-button)] border border-[var(--color-border)] px-4 py-2 text-sm font-medium text-[var(--color-text)] transition-colors hover:bg-[var(--color-card-hover)] disabled:opacity-50"
            >
              <Save size={14} />
              {updateContentMutation.isPending ? 'Saving...' : 'Save changes'}
            </button>
          )}
        </>
      )}

      {/* Mode toggle */}
      <div className="flex gap-2">
        <button
          onClick={() => setMode('now')}
          className={`flex items-center gap-1.5 rounded-[var(--radius-button)] px-4 py-2 text-sm font-medium transition-colors ${
            mode === 'now'
              ? 'bg-[var(--color-accent)] text-white'
              : 'border border-[var(--color-border)] text-[var(--color-text-secondary)]'
          }`}
        >
          <Send size={14} />
          Publish Now
        </button>
        <button
          onClick={() => setMode('schedule')}
          className={`flex items-center gap-1.5 rounded-[var(--radius-button)] px-4 py-2 text-sm font-medium transition-colors ${
            mode === 'schedule'
              ? 'bg-[var(--color-accent)] text-white'
              : 'border border-[var(--color-border)] text-[var(--color-text-secondary)]'
          }`}
        >
          <Clock size={14} />
          Schedule
        </button>
      </div>

      {mode === 'schedule' && (
        <ScheduleCalendar scheduledAt={scheduledAt} onChange={setScheduledAt} />
      )}

      {mode === 'schedule' && selected?.is_repost && activePlatform === 'linkedin' && (
        <div className="flex items-start gap-2 rounded-[var(--radius-card)] border border-yellow-500/30 bg-yellow-500/10 px-3 py-2.5 text-sm text-yellow-400">
          <span className="mt-0.5 shrink-0">⚠</span>
          <span>
            LinkedIn does not support native scheduled reposts. This will be queued and
            posted at the scheduled time by GoViral's scheduler.
          </span>
        </div>
      )}

      {/* Action button */}
      <button
        onClick={handlePublish}
        disabled={!selectedId || publishMutation.isPending || scheduleMutation.isPending}
        className="rounded-[var(--radius-button)] bg-[var(--color-accent)] px-6 py-2.5 text-sm font-medium text-white transition-colors hover:bg-[var(--color-accent-hover)] disabled:opacity-50"
      >
        {publishMutation.isPending || scheduleMutation.isPending
          ? 'Processing...'
          : mode === 'now'
            ? 'Publish'
            : 'Schedule'}
      </button>
    </div>
  )
}
