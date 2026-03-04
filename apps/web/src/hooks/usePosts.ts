import { useQuery, useQueryClient } from '@tanstack/react-query'
import { apiClient } from '@/lib/api'
import { useSSEMutation } from './useSSE'
import { useExtensionCookies } from './useExtensionCookies'
import { useExtensionLinkedIn } from './useExtensionLinkedIn'
import type { Post } from '@/lib/types'

export function usePostsQuery(platform?: string) {
  return useQuery({
    queryKey: ['posts', platform],
    queryFn: () => apiClient.get<Post[]>('/posts', platform ? { platform } : undefined),
  })
}

export function useFetchPostsMutation() {
  const queryClient = useQueryClient()
  const { extension } = useExtensionCookies()
  const extensionLinkedIn = useExtensionLinkedIn()

  const sseMutation = useSSEMutation<Post[]>('/posts/fetch', {
    onComplete: () => {
      void queryClient.invalidateQueries({ queryKey: ['posts'] })
    },
  })

  return {
    ...sseMutation,
    mutate: (body: { platform: string }) => {
      if (body.platform === 'linkedin' && extension.available) {
        extensionLinkedIn.fetchMyPosts().then(() => {
          void queryClient.invalidateQueries({ queryKey: ['posts'] })
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
