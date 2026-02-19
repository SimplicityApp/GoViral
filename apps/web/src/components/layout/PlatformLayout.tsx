import { useEffect } from 'react'
import { Outlet, Navigate, useParams } from 'react-router-dom'
import { platforms, defaultPlatform } from '@/lib/platforms'
import { usePlatformStore } from '@/stores/platform-store'

const validPlatformIds = new Set(platforms.map((p) => p.id))

export function PlatformLayout() {
  const { platform } = useParams<{ platform: string }>()
  const setActivePlatform = usePlatformStore((s) => s.setActivePlatform)

  useEffect(() => {
    if (platform && validPlatformIds.has(platform)) {
      setActivePlatform(platform)
    }
  }, [platform, setActivePlatform])

  if (!platform || !validPlatformIds.has(platform)) {
    return <Navigate to={`/${defaultPlatform}/dashboard`} replace />
  }

  return <Outlet />
}
