import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import type { TrendingPost } from '@/lib/types'
import { usePlatformStore } from '@/stores/platform-store'
import { useTrendingQuery, useDiscoverMutation } from '@/hooks/useTrending'
import { TrendingFilters } from '@/components/trending/TrendingFilters'
import { TrendingList } from '@/components/trending/TrendingList'
import { Sparkles } from 'lucide-react'

export function Trending() {
  const { activePlatform } = usePlatformStore()
  const navigate = useNavigate()
  const [period, setPeriod] = useState('24h')
  const [minLikes, setMinLikes] = useState('')
  const [niche, setNiche] = useState('')
  const [selectedIds, setSelectedIds] = useState<Set<number>>(new Set())

  const { data: posts, isLoading } = useTrendingQuery({
    platform: activePlatform,
    period,
    min_likes: minLikes ? Number(minLikes) : undefined,
    niche: niche || undefined,
  })
  const discoverMutation = useDiscoverMutation()

  const handleDiscover = () => {
    discoverMutation.mutate({
      platform: activePlatform,
      period,
      min_likes: minLikes ? Number(minLikes) : undefined,
      niche: niche || undefined,
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
    navigate(`/generate?ids=${post.id}&step=1&repost=true`)
  }

  const handleConfigure = () => {
    const ids = Array.from(selectedIds).join(',')
    navigate(`/generate?ids=${ids}&step=1`)
  }

  return (
    <div className="mx-auto max-w-3xl p-6">
      <div className="mb-6">
        <h2 className="mb-4 text-lg font-semibold text-[var(--color-text)]">Trending</h2>
        <TrendingFilters
          period={period}
          minLikes={minLikes}
          niche={niche}
          onPeriodChange={setPeriod}
          onMinLikesChange={setMinLikes}
          onNicheChange={setNiche}
          onDiscover={handleDiscover}
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
      />
    </div>
  )
}
