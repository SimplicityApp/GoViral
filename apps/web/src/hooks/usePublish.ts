import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiClient } from '@/lib/api'
import type { ScheduledPost } from '@/lib/types'

export function usePublishMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (body: { content_id: number; platform: string }) =>
      apiClient.post<{ post_id: string }>('/publish', body),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['history'] })
    },
  })
}

export function useScheduleQuery() {
  return useQuery({
    queryKey: ['schedule'],
    queryFn: () => apiClient.get<ScheduledPost[]>('/schedule'),
  })
}

export function useScheduleMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (body: { content_id: number; scheduled_at: string }) =>
      apiClient.post<ScheduledPost>('/schedule', body),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['schedule'] })
    },
  })
}

export function useRunDueMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: () =>
      apiClient.post<{ executed: number; results: unknown[] }>('/schedule/run'),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['schedule'] })
    },
  })
}

export function useAcknowledgeScheduleMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => apiClient.patch(`/schedule/${id}/ack`),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['schedule'] })
    },
  })
}

export function useCancelScheduleMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => apiClient.delete(`/schedule/${id}`),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['schedule'] })
    },
  })
}
