import { useQuery, useQueryClient } from '@tanstack/react-query'
import { apiClient } from '@/lib/api'
import { useSSEMutation } from './useSSE'
import { useExtensionCookies } from './useExtensionCookies'
import { useExtensionLinkedIn } from './useExtensionLinkedIn'
import type { TrendingPost } from '@/lib/types'

interface TrendingFilters {
  platform?: string
  period?: string
  min_likes?: number
  niche?: string
}

export function useTrendingQuery(filters?: TrendingFilters) {
  return useQuery({
    queryKey: ['trending', filters],
    queryFn: () =>
      apiClient.get<TrendingPost[]>('/trending', filters as Record<string, string | number>),
  })
}

export function useDiscoverMutation() {
  const queryClient = useQueryClient()
  const { extension } = useExtensionCookies()
  const extensionLinkedIn = useExtensionLinkedIn()

  const sseMutation = useSSEMutation<TrendingPost[]>('/trending/discover')

  return {
    ...sseMutation,
    mutate: (body: { platform: string; period?: string; min_likes?: number; niche?: string; keywords?: string; limit?: number; niches?: string[] }) => {
      if (body.platform === 'linkedin' && extension.available) {
        const niches = body.niches || (body.niche ? [body.niche] : (body.keywords ? body.keywords.split(/[\s,]+/).filter(Boolean) : []))
        extensionLinkedIn.fetchTrending(niches, body.period || '24h', body.limit).then(() => {
          void queryClient.invalidateQueries({ queryKey: ['trending'] })
        })
      } else {
        sseMutation.mutate(body)
      }
    },
    isRunning: sseMutation.isRunning || extensionLinkedIn.isRunning,
    progress: extensionLinkedIn.isRunning
      ? { type: 'progress' as const, message: extensionLinkedIn.progress, percentage: 0 }
      : sseMutation.progress,
    error: extensionLinkedIn.error || sseMutation.error,
  }
}
