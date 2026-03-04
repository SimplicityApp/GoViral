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
    has_twikit_auth: boolean
    auth_token?: string
    ct0?: string
  }
  linkedin: {
    client_id: string
    client_secret: string
    has_auth: boolean
    has_linkitin_auth: boolean
    li_at?: string
    jsessionid?: string
  }
  niches: string[]
  linkedin_niches: string[]
  github?: {
    personal_access_token: string
  }
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
  linkedin_niches?: string[]
  github?: { personal_access_token?: string }
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
