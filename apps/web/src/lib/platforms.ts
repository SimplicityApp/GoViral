export interface Platform {
  id: string
  name: string
  icon: string
}

export const platforms: Platform[] = [
  { id: 'x', name: 'X', icon: 'Twitter' },
  { id: 'linkedin', name: 'LinkedIn', icon: 'Linkedin' },
  { id: 'youtube', name: 'YouTube', icon: 'Youtube' },
  { id: 'tiktok', name: 'TikTok', icon: 'Music' },
]

export const defaultPlatform = 'x'
