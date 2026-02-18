import type { Post, TrendingPost } from '@/lib/types'
import { MetricsBadge } from './MetricsBadge'
import { formatRelativeTime } from '@/lib/format'
import { usePlatformStore } from '@/stores/platform-store'
import { Repeat2 } from 'lucide-react'

function isTrendingPost(post: Post | TrendingPost): post is TrendingPost {
  return 'author_username' in post
}

interface PostCardProps {
  post: Post | TrendingPost
  selectable?: boolean
  selected?: boolean
  onSelect?: () => void
  onRepost?: () => void
}

export function PostCard({ post, selectable, selected, onSelect, onRepost }: PostCardProps) {
  const { activePlatform } = usePlatformStore()
  const isLinkedIn = activePlatform === 'linkedin'
  const content =
    post.content.length > 280
      ? post.content.slice(0, 280) + '...'
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
        </div>
      )}

      <p className="mb-3 whitespace-pre-wrap text-sm text-[var(--color-text)]">
        {content}
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
        <span className="text-xs text-[var(--color-text-secondary)]">
          {formatRelativeTime(post.posted_at)}
        </span>
      </div>
    </div>
  )
}
