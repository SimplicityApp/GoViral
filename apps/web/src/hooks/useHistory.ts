import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiClient } from '@/lib/api'
import type { GeneratedContent } from '@/lib/types'

export function useHistoryQuery(status?: string, limit?: number, platform?: string) {
  return useQuery({
    queryKey: ['history', status, limit, platform],
    queryFn: () =>
      apiClient.get<GeneratedContent[]>('/history', {
        ...(status && { status }),
        ...(limit && { limit }),
        ...(platform && { platform }),
      }),
  })
}

export function useUpdateStatusMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, status }: { id: number; status: string }) =>
      apiClient.patch<GeneratedContent>(`/history/${id}`, { status }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['history'] })
    },
  })
}

export function useUpdateContentMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, generated_content, code_image_description }: { id: number; generated_content?: string; code_image_description?: string }) =>
      apiClient.patch<GeneratedContent>(`/history/${id}`, {
        ...(generated_content !== undefined && { generated_content }),
        ...(code_image_description !== undefined && { code_image_description }),
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['history'] })
    },
  })
}

export function useDeleteContentMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => apiClient.delete(`/history/${id}`),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['history'] })
    },
  })
}
