import { useState } from 'react'
import type { Post, TrendingPost } from '@/lib/types'
import { MetricsBadge } from './MetricsBadge'
import { formatRelativeTime } from '@/lib/format'
import { usePlatformParam } from '@/hooks/usePlatformParam'
import { Repeat2, MessageCircle } from 'lucide-react'

function isTrendingPost(post: Post | TrendingPost): post is TrendingPost {
  return 'author_username' in post
}

interface PostCardProps {
  post: Post | TrendingPost
  selectable?: boolean
  selected?: boolean
  onSelect?: () => void
  onRepost?: () => void
  onComment?: () => void
}

export function PostCard({ post, selectable, selected, onSelect, onRepost, onComment }: PostCardProps) {
  const activePlatform = usePlatformParam()
  const isLinkedIn = activePlatform === 'linkedin'
  const [expanded, setExpanded] = useState(false)
  const previewLimit = isLinkedIn ? 400 : 280
  const isTruncated = post.content.length > previewLimit
  const content = !expanded && isTruncated
    ? post.content.slice(0, previewLimit) + '...'
    : post.content

  return (
    <div
      onClick={selectable ? onSelect : undefined}
      className={`rounded-[var(--radius-card)] border bg-[var(--color-card)] p-4 transition-colors hover:bg-[var(--color-card-hover)] ${
        selected
          ? 'border-[var(--color-accent)]'
          : 'border-[var(--color-border)]'
      } ${selectable ? 'cursor-pointer' : ''}`}
    >
      {isTrendingPost(post) && (
        <div className="mb-2 flex items-center gap-2">
          <span className="text-sm font-semibold text-[var(--color-text)]">
            {post.author_name}
          </span>
          <span className="text-xs text-[var(--color-text-secondary)]">
            @{post.author_username}
          </span>
          {!post.is_actionable && (
            <span
              title={isLinkedIn
                ? "Scraped without a direct LinkedIn URN — auto-posting unavailable. Generate a draft to post manually."
                : "Scraped post — direct engagement unavailable. Generate a draft to post manually."}
              className="rounded-full bg-amber-100 px-2 py-0.5 text-[10px] font-medium text-amber-700 dark:bg-amber-900/30 dark:text-amber-400 cursor-help"
            >
              View only
            </span>
          )}
        </div>
      )}

      <p className="mb-3 whitespace-pre-wrap text-sm text-[var(--color-text)]">
        {content}
        {isTruncated && (
          <button
            onClick={(e) => {
              e.stopPropagation()
              setExpanded(!expanded)
            }}
            className="ml-1 text-[var(--color-accent)] hover:underline"
          >
            {expanded ? 'less' : 'more'}
          </button>
        )}
      </p>

      <div className="flex items-center gap-4">
        <MetricsBadge
          type="likes"
          count={post.likes}
        />
        <MetricsBadge
          type="reposts"
          count={post.reposts}
        />
        <MetricsBadge
          type="comments"
          count={post.comments}
        />
        <MetricsBadge
          type="views"
          count={post.impressions}
        />
        <span className="text-xs text-[var(--color-text-secondary)]">
          {isLinkedIn ? 'Reactions' : null}
        </span>
        {isTrendingPost(post) && onRepost && (
          <button
            onClick={(e) => {
              e.stopPropagation()
              onRepost()
            }}
            className="ml-auto flex items-center gap-1 rounded-[var(--radius-button)] border border-[var(--color-border)] px-2 py-1 text-xs text-[var(--color-text-secondary)] transition-colors hover:border-[var(--color-accent)] hover:text-[var(--color-accent)]"
          >
            <Repeat2 size={12} />
            Repost
          </button>
        )}
        {isTrendingPost(post) && onComment && (
          <button
            onClick={(e) => {
              e.stopPropagation()
              onComment()
            }}
            className="flex items-center gap-1 rounded-[var(--radius-button)] border border-[var(--color-border)] px-2 py-1 text-xs text-[var(--color-text-secondary)] transition-colors hover:border-[var(--color-accent)] hover:text-[var(--color-accent)]"
          >
            <MessageCircle size={12} />
            Comment
          </button>
        )}
        <span className="text-xs text-[var(--color-text-secondary)]">
          {formatRelativeTime(post.posted_at)}
        </span>
      </div>
    </div>
  )
}
