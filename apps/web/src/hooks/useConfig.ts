import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiClient } from '@/lib/api'

export interface AppConfig {
  claude: {
    api_key: string
    model: string
  }
  gemini: {
    api_key: string
    model: string
  }
  x: {
    api_key: string
    api_secret: string
    bearer_token: string
    client_id: string
    client_secret: string
    username: string
    has_auth: boolean
  }
  linkedin: {
    client_id: string
    client_secret: string
    has_auth: boolean
  }
  niches: string[]
}

export interface UpdateConfigPayload {
  claude?: { api_key?: string; model?: string }
  gemini?: { api_key?: string; model?: string }
  x?: {
    api_key?: string
    api_secret?: string
    bearer_token?: string
    client_id?: string
    client_secret?: string
    username?: string
  }
  linkedin?: { client_id?: string; client_secret?: string }
  niches?: string[]
}

export function useConfigQuery() {
  return useQuery({
    queryKey: ['config'],
    queryFn: () => apiClient.get<AppConfig>('/config'),
  })
}

export function useUpdateConfigMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (body: UpdateConfigPayload) =>
      apiClient.patch<AppConfig>('/config', body),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['config'] })
    },
  })
}
