import { useParams } from 'react-router-dom'
import { defaultPlatform } from '@/lib/platforms'

export function usePlatformParam() {
  const { platform } = useParams<{ platform: string }>()
  return platform ?? defaultPlatform
}
