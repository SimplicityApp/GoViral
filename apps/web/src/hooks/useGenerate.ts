import { useSSEMutation } from './useSSE'
import type { GeneratedContent } from '@/lib/types'

export function useGenerateMutation() {
  const sse = useSSEMutation<GeneratedContent[]>('/generate')
  return {
    mutate: sse.mutate,
    progress: sse.progress,
    isGenerating: sse.isRunning,
    result: sse.result,
    error: sse.error,
    cancel: sse.cancel,
  }
}
