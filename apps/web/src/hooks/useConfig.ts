import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiClient } from '@/lib/api'

export interface AppConfig {
  claude: {
    has_global_key: boolean;
    user_api_key: string;
    model: string;
    daily_limit: number;
    daily_used: number;
  };
  gemini: {
    has_global_key: boolean;
    user_api_key: string;
    model: string;
    daily_limit: number;
    daily_used: number;
  };
  x: {
    has_api_key: boolean;
    has_api_secret: boolean;
    has_bearer_token: boolean;
    has_client_id: boolean;
    has_client_secret: boolean;
    username: string;
    has_auth: boolean;
    has_twikit_auth: boolean;
    auth_token?: string;
    ct0?: string;
  };
  linkedin: {
    has_client_id: boolean;
    has_client_secret: boolean;
    has_auth: boolean;
    has_linkitin_auth: boolean;
    li_at?: string;
    jsessionid?: string;
  };
  github?: {
    has_pat: boolean;
    has_oauth: boolean;
    has_auth: boolean;
    default_owner: string;
    default_repo: string;
  };
  youtube?: {
    has_client_id: boolean;
    has_auth: boolean;
    channel_id: string;
  };
  tiktok?: {
    has_client_key: boolean;
    has_auth: boolean;
    username: string;
  };
  niches: string[];
  linkedin_niches: string[];
  self_description: string;
}

export interface UpdateConfigPayload {
  claude?: { api_key?: string; model?: string }
  gemini?: { api_key?: string; model?: string }
  x?: { username?: string }
  linkedin?: { person_urn?: string }
  youtube?: { channel_id?: string }
  tiktok?: { username?: string }
  niches?: string[]
  linkedin_niches?: string[]
  self_description?: string
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
