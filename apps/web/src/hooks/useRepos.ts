import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiClient } from '@/lib/api'
import type { AvailableRepo, CodeImageOptions, CodeImagePreviewsResponse, Repo, RepoCommit, RepoLink } from '@/lib/types'

export function useAvailableReposQuery() {
  return useQuery<AvailableRepo[]>({
    queryKey: ['available-repos'],
    queryFn: () => apiClient.get('/repos/available'),
    staleTime: 5 * 60 * 1000, // match server cache
  })
}

export function useReposQuery() {
  return useQuery<Repo[]>({
    queryKey: ['repos'],
    queryFn: () => apiClient.get('/repos'),
  })
}

export function useAddRepoMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (data: { owner: string; name: string }) =>
      apiClient.post<Repo>('/repos', data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['repos'] })
    },
  })
}

export function useDeleteRepoMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => apiClient.delete(`/repos/${id}`),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['repos'] })
    },
  })
}

export function useUpdateRepoSettingsMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (data: { id: number; target_audience: string; links: RepoLink[] }) =>
      apiClient.patch<Repo>(`/repos/${data.id}/settings`, {
        target_audience: data.target_audience,
        links: data.links,
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['repos'] })
    },
  })
}

export function useRepoCommitsQuery(repoId: number | null) {
  return useQuery<RepoCommit[]>({
    queryKey: ['repo-commits', repoId],
    queryFn: () => apiClient.get(`/repos/${repoId}/commits`),
    enabled: !!repoId,
  })
}

export function useCodeImageOptionsQuery() {
  return useQuery<CodeImageOptions>({
    queryKey: ['code-image-options'],
    queryFn: () => apiClient.get('/repos/code-image-options'),
    staleTime: Infinity, // templates/themes don't change at runtime
  })
}

export function useCodeImagePreviewsQuery(theme: string) {
  return useQuery<CodeImagePreviewsResponse>({
    queryKey: ['code-image-previews', theme],
    queryFn: () => apiClient.get('/repos/code-image-previews', { theme }),
    staleTime: Infinity,
    enabled: !!theme,
  })
}
