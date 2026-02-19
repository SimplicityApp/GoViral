import { useQuery } from '@tanstack/react-query'
import { apiClient } from '@/lib/api'
import { useSSEMutation } from './useSSE'
import type { Persona } from '@/lib/types'

export function usePersonaQuery(platform?: string) {
  return useQuery({
    queryKey: ['persona', platform],
    queryFn: () => apiClient.get<Persona>('/persona', platform ? { platform } : undefined),
  })
}

export function useBuildPersonaMutation(options?: { onComplete?: () => void }) {
  return useSSEMutation<Persona>('/persona/build', options)
}
