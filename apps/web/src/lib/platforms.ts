export interface Platform {
  id: string
  name: string
  icon: string
}

export const platforms: Platform[] = [
  { id: 'x', name: 'X', icon: 'Twitter' },
  { id: 'linkedin', name: 'LinkedIn', icon: 'Linkedin' },
]

export const defaultPlatform = 'x'
