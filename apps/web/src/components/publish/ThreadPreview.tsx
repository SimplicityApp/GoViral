import { usePlatformStore } from '@/stores/platform-store'

interface ThreadPreviewProps {
  content: string
  editable?: boolean
  onChange?: (content: string) => void
  isRepost?: boolean
  quoteTweetId?: string
}

export function ThreadPreview({ content, editable, onChange, isRepost, quoteTweetId }: ThreadPreviewProps) {
  const { activePlatform } = usePlatformStore()
  const maxChars = activePlatform === 'x' ? 280 : 3000

  if (isRepost) {
    return (
      <div className="flex flex-col gap-3">
        {editable && (
          <div>
            <label className="mb-1 block text-sm font-medium text-[var(--color-text-secondary)]">
              Edit content
            </label>
            <textarea
              value={content}
              onChange={(e) => onChange?.(e.target.value)}
              rows={6}
              className="w-full resize-y rounded-[var(--radius-card)] border border-[var(--color-border)] bg-[var(--color-card)] p-3 text-sm text-[var(--color-text)] placeholder-[var(--color-text-secondary)] focus:border-[var(--color-accent)] focus:outline-none"
            />
          </div>
        )}
        <h4 className="text-sm font-medium text-[var(--color-text-secondary)]">
          Preview (Quote Tweet)
        </h4>
        <div className="rounded-[var(--radius-card)] border border-[var(--color-border)] bg-[var(--color-card)] p-4">
          <p className="whitespace-pre-wrap text-sm text-[var(--color-text)]">{content}</p>
          {quoteTweetId && (
            <div className="mt-3 rounded-lg border border-[var(--color-border)] bg-[var(--color-bg)] p-3 text-xs text-[var(--color-text-secondary)]">
              Quoting: https://x.com/i/status/{quoteTweetId}
            </div>
          )}
          <div className="mt-2 text-right text-xs text-[var(--color-text-secondary)]">
            {content.length}/280
          </div>
        </div>
      </div>
    )
  }

  const parts: string[] = []
  if (content.length <= maxChars) {
    parts.push(content)
  } else {
    let remaining = content
    while (remaining.length > 0) {
      if (remaining.length <= maxChars) {
        parts.push(remaining)
        break
      }
      let splitAt = remaining.lastIndexOf(' ', maxChars)
      if (splitAt === -1) splitAt = maxChars
      parts.push(remaining.slice(0, splitAt))
      remaining = remaining.slice(splitAt).trimStart()
    }
  }

  return (
    <div className="flex flex-col gap-3">
      {editable && (
        <div>
          <label className="mb-1 block text-sm font-medium text-[var(--color-text-secondary)]">
            Edit content
          </label>
          <textarea
            value={content}
            onChange={(e) => onChange?.(e.target.value)}
            rows={6}
            className="w-full resize-y rounded-[var(--radius-card)] border border-[var(--color-border)] bg-[var(--color-card)] p-3 text-sm text-[var(--color-text)] placeholder-[var(--color-text-secondary)] focus:border-[var(--color-accent)] focus:outline-none"
          />
        </div>
      )}
      <h4 className="text-sm font-medium text-[var(--color-text-secondary)]">
        Preview ({parts.length} part{parts.length > 1 ? 's' : ''})
      </h4>
      {parts.map((part, i) => (
        <div
          key={i}
          className="rounded-[var(--radius-card)] border border-[var(--color-border)] bg-[var(--color-card)] p-4"
        >
          <p className="whitespace-pre-wrap text-sm text-[var(--color-text)]">{part}</p>
          <div className="mt-2 flex justify-between text-xs text-[var(--color-text-secondary)]">
            <span>
              Part {i + 1} of {parts.length}
            </span>
            <span
              className={
                part.length > maxChars ? 'text-red-400' : ''
              }
            >
              {part.length}/{maxChars}
            </span>
          </div>
        </div>
      ))}
    </div>
  )
}
