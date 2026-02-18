import { Heart, Repeat2, MessageCircle, Eye } from 'lucide-react'
import { formatCount } from '@/lib/format'

const iconMap = {
  likes: Heart,
  reposts: Repeat2,
  comments: MessageCircle,
  views: Eye,
}

const colorMap = {
  likes: 'var(--color-like)',
  reposts: 'var(--color-repost)',
  comments: 'var(--color-comment)',
  views: 'var(--color-view)',
}

interface MetricsBadgeProps {
  type: 'likes' | 'reposts' | 'comments' | 'views'
  count: number
}

export function MetricsBadge({ type, count }: MetricsBadgeProps) {
  const Icon = iconMap[type]
  const color = colorMap[type]

  return (
    <span className="flex items-center gap-1 text-xs" style={{ color }}>
      <Icon size={14} />
      {formatCount(count)}
    </span>
  )
}
