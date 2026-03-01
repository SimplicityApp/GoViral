import { useMutation, useQueryClient } from '@tanstack/react-query'
import { apiClient } from '@/lib/api'
import type { GeneratedContent } from '@/lib/types'

export function useGenerateCommentMutation(platform: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (body: { trending_post_id: number; platform?: string; count?: number }) =>
      apiClient.post<GeneratedContent[]>(`/${platform}/comment/generate`, body),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['history'] })
    },
  })
}

export function usePostCommentMutation(platform: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (body: { content_id: number }) =>
      apiClient.post<{ comment_urn: string; content: GeneratedContent }>(`/${platform}/comment`, body),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['history'] })
    },
  })
}
