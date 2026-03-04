import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { toast } from 'sonner'
import type { TrendingPost, GeneratedContent } from '@/lib/types'
import { usePlatformParam } from '@/hooks/usePlatformParam'
import { useTrendingQuery, useDiscoverMutation } from '@/hooks/useTrending'
import { useConfigQuery } from '@/hooks/useConfig'
import { useGenerateCommentMutation, usePostCommentMutation } from '@/hooks/useComment'
import { TrendingFilters } from '@/components/trending/TrendingFilters'
import { TrendingList } from '@/components/trending/TrendingList'
import { Sparkles } from 'lucide-react'

export function Trending() {
  const platform = usePlatformParam()
  const navigate = useNavigate()
  const { data: config } = useConfigQuery()
  const platformNiches =
    platform === 'linkedin' ? (config?.linkedin_niches ?? []) : (config?.niches ?? [])
  const [period, setPeriod] = useState('24h')
  const [minLikes, setMinLikes] = useState('')
  const [niche, setNiche] = useState('')
  const [selectedIds, setSelectedIds] = useState<Set<number>>(new Set())
  const [commentingPost, setCommentingPost] = useState<TrendingPost | null>(null)
  const [commentVariations, setCommentVariations] = useState<GeneratedContent[]>([])

  const { data: posts, isLoading } = useTrendingQuery({
    platform: platform,
    period,
    min_likes: minLikes ? Number(minLikes) : undefined,
    niche: niche || undefined,
  })
  const discoverMutation = useDiscoverMutation()
  const generateComment = useGenerateCommentMutation(platform)
  const postComment = usePostCommentMutation(platform)

  const handleDiscover = () => {
    discoverMutation.mutate({
      platform: platform,
      period,
      min_likes: minLikes ? Number(minLikes) : undefined,
      niche: niche || undefined,
      niches: niche ? [niche] : platformNiches,
    })
  }

  const toggleSelect = (id: number) => {
    setSelectedIds((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  const handleRepost = (post: TrendingPost) => {
    if (post.platform !== platform) {
      toast.error(`This post is from ${post.platform}, not ${platform}`)
      return
    }
    navigate(`/${platform}/generate?ids=${post.id}&step=1&repost=true`)
  }

  const handleComment = (post: TrendingPost) => {
    if (post.platform !== platform) {
      toast.error(`This post is from ${post.platform}, not ${platform}`)
      return
    }
    setCommentingPost(post)
    generateComment.mutate(
      { trending_post_id: post.id, platform },
      {
        onSuccess: (data) => {
          setCommentVariations(data)
        },
        onError: (err) => {
          toast.error(`Failed to generate comments: ${err.message}`)
          setCommentingPost(null)
        },
      }
    )
  }

  const handlePostComment = (contentId: number) => {
    postComment.mutate(
      { content_id: contentId },
      {
        onSuccess: (data) => {
          toast.success(`Comment posted! URN: ${data.comment_urn}`)
          setCommentingPost(null)
          setCommentVariations([])
        },
        onError: (err) => {
          toast.error(`Failed to post comment: ${err.message}`)
        },
      }
    )
  }

  const handleConfigure = () => {
    const ids = Array.from(selectedIds).join(',')
    navigate(`/${platform}/generate?ids=${ids}&step=1`)
  }

  return (
    <div className="mx-auto max-w-3xl p-6">
      <div className="mb-6">
        <h2 className="mb-4 text-lg font-semibold text-[var(--color-text)]">Trending</h2>
        <TrendingFilters
          period={period}
          minLikes={minLikes}
          niche={niche}
          niches={platformNiches}
          onPeriodChange={setPeriod}
          onMinLikesChange={setMinLikes}
          onNicheChange={setNiche}
          onDiscover={handleDiscover}
          onManageNiches={() => navigate('/settings')}
          isDiscovering={discoverMutation.isRunning}
        />
      </div>

      {discoverMutation.progress && (
        <div className="mb-4 rounded-lg border border-[var(--color-border)] bg-[var(--color-card)] p-3">
          <div className="mb-1 text-sm text-[var(--color-text)]">
            {discoverMutation.progress.message}
          </div>
          <div className="h-1.5 w-full rounded-full bg-[var(--color-border)]">
            <div
              className="h-1.5 rounded-full bg-[var(--color-accent)] transition-all"
              style={{ width: `${discoverMutation.progress.percentage}%` }}
            />
          </div>
        </div>
      )}

      {selectedIds.size > 0 && (
        <div className="mb-4 flex items-center gap-3">
          <span className="text-sm text-[var(--color-accent)]">
            {selectedIds.size} post{selectedIds.size > 1 ? 's' : ''} selected
          </span>
          <button
            onClick={handleConfigure}
            className="flex items-center gap-1.5 rounded-[var(--radius-button)] bg-[var(--color-accent)] px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-[var(--color-accent-hover)]"
          >
            <Sparkles size={16} />
            Configure
          </button>
        </div>
      )}

      <TrendingList
        posts={posts}
        isLoading={isLoading}
        selectedIds={selectedIds}
        onToggleSelect={toggleSelect}
        onRepost={handleRepost}
        onComment={handleComment}
      />

      {commentingPost && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="mx-4 w-full max-w-lg rounded-xl border border-[var(--color-border)] bg-[var(--color-bg)] p-6">
            <div className="mb-4 flex items-center justify-between">
              <h3 className="text-lg font-semibold text-[var(--color-text)]">
                Comment on @{commentingPost.author_username}'s post
              </h3>
              <button
                onClick={() => { setCommentingPost(null); setCommentVariations([]) }}
                className="text-[var(--color-text-secondary)] hover:text-[var(--color-text)]"
              >
                ✕
              </button>
            </div>

            <div className="mb-4 max-h-24 overflow-y-auto rounded-lg bg-[var(--color-card)] p-3 text-xs text-[var(--color-text-secondary)]">
              {commentingPost.content.slice(0, 200)}
              {commentingPost.content.length > 200 ? '...' : ''}
            </div>

            {generateComment.isPending && (
              <div className="py-8 text-center text-sm text-[var(--color-text-secondary)]">
                Generating comment variations...
              </div>
            )}

            {commentVariations.length > 0 && (
              <div className="flex flex-col gap-3">
                {commentVariations.map((cv) => (
                  <div
                    key={cv.id}
                    className="rounded-lg border border-[var(--color-border)] bg-[var(--color-card)] p-3"
                  >
                    <p className="mb-2 whitespace-pre-wrap text-sm text-[var(--color-text)]">
                      {cv.generated_content}
                    </p>
                    <button
                      onClick={() => handlePostComment(cv.id)}
                      disabled={postComment.isPending}
                      className="rounded-[var(--radius-button)] bg-[var(--color-accent)] px-3 py-1 text-xs font-medium text-white transition-colors hover:bg-[var(--color-accent-hover)] disabled:opacity-50"
                    >
                      {postComment.isPending ? 'Posting...' : 'Post Comment'}
                    </button>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  )
}
