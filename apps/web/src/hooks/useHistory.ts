import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiClient } from '@/lib/api'
import type { GeneratedContent } from '@/lib/types'

export function useHistoryQuery(status?: string, limit?: number) {
  return useQuery({
    queryKey: ['history', status, limit],
    queryFn: () =>
      apiClient.get<GeneratedContent[]>('/history', {
        ...(status && { status }),
        ...(limit && { limit }),
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
    mutationFn: ({ id, generated_content }: { id: number; generated_content: string }) =>
      apiClient.patch<GeneratedContent>(`/history/${id}`, { generated_content }),
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
