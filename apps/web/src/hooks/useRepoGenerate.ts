import { useSSEMutation } from './useSSE'
import type { GeneratedContent } from '@/lib/types'

export function useFetchCommits(repoId: number | null) {
  const sse = useSSEMutation<null>(`/repos/${repoId}/fetch`)
  return {
    mutate: sse.mutate,
    isLoading: sse.isRunning,
    progress: sse.progress,
    error: sse.error,
    cancel: sse.cancel,
  }
}

export function useRepoGenerate() {
  const sse = useSSEMutation<GeneratedContent[]>('/repos/generate')
  return {
    mutate: sse.mutate,
    isLoading: sse.isRunning,
    progress: sse.progress,
    result: sse.result,
    error: sse.error,
    cancel: sse.cancel,
  }
}
