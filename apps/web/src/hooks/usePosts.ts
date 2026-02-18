import { useQuery } from '@tanstack/react-query'
import { apiClient } from '@/lib/api'
import { useSSEMutation } from './useSSE'
import type { Post } from '@/lib/types'

export function usePostsQuery(platform?: string) {
  return useQuery({
    queryKey: ['posts', platform],
    queryFn: () => apiClient.get<Post[]>('/posts', platform ? { platform } : undefined),
  })
}

export function useFetchPostsMutation() {
  return useSSEMutation<Post[]>('/posts/fetch')
}
