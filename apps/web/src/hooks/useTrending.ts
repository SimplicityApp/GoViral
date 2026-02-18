import { useQuery } from '@tanstack/react-query'
import { apiClient } from '@/lib/api'
import { useSSEMutation } from './useSSE'
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
  return useSSEMutation<TrendingPost[]>('/trending/discover')
}
