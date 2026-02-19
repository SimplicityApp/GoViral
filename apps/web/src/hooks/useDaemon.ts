import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiClient } from '@/lib/api'
import type { DaemonStatus, DaemonBatch, DaemonConfig } from '@/lib/types'

export function useDaemonStatusQuery() {
  return useQuery({
    queryKey: ['daemon-status'],
    queryFn: () => apiClient.get<DaemonStatus>('/daemon/status'),
    refetchInterval: 10000,
  })
}

export function useDaemonBatchesQuery(platform?: string, status?: string) {
  return useQuery({
    queryKey: ['daemon-batches', platform, status],
    queryFn: () =>
      apiClient.get<DaemonBatch[]>('/daemon/batches', {
        platform,
        status,
      }),
    refetchInterval: 15000,
  })
}

export function useDaemonBatchQuery(id: number) {
  return useQuery({
    queryKey: ['daemon-batch', id],
    queryFn: () => apiClient.get<DaemonBatch>(`/daemon/batches/${id}`),
  })
}

export function useBatchActionMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, action }: { id: number; action: string }) =>
      apiClient.post<DaemonBatch>(`/daemon/batches/${id}/action`, { action }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['daemon-batches'] })
    },
  })
}

export function useDaemonRunNowMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: () => apiClient.post<unknown>('/daemon/run'),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['daemon-status'] })
    },
  })
}

export function useDaemonStartMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: () => apiClient.post<unknown>('/daemon/start'),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['daemon-status'] })
    },
  })
}

export function useDaemonStopMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: () => apiClient.post<unknown>('/daemon/stop'),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['daemon-status'] })
    },
  })
}

export function useDaemonConfigQuery() {
  return useQuery({
    queryKey: ['daemon-config'],
    queryFn: () => apiClient.get<DaemonConfig>('/daemon/config'),
  })
}

export function useUpdateDaemonConfigMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (body: Partial<DaemonConfig>) =>
      apiClient.patch<DaemonConfig>('/daemon/config', body),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['daemon-config'] })
    },
  })
}
