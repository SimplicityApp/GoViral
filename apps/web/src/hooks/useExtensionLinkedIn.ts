import { useState, useCallback } from 'react'
import { apiClient } from '@/lib/api'

interface ExtensionLinkedInResult {
  posts: any[]
  error?: string
}

export function useExtensionLinkedIn() {
  const [isRunning, setIsRunning] = useState(false)
  const [progress, setProgress] = useState('')
  const [error, setError] = useState<string | null>(null)

  // Helper to send message via postMessage and await response
  function sendExtensionMessage(type: string, payload: Record<string, any> = {}): Promise<ExtensionLinkedInResult> {
    return new Promise((resolve, reject) => {
      const requestId = crypto.randomUUID()
      const responseType = type + '_RESULT'
      const timeout = setTimeout(() => {
        window.removeEventListener('message', handler)
        reject(new Error('Extension request timed out'))
      }, 60000) // 60s timeout for scraping

      function handler(event: MessageEvent) {
        if (event.origin !== window.location.origin) return
        if (event.data?.type !== responseType) return
        if (event.data?.requestId !== requestId) return
        clearTimeout(timeout)
        window.removeEventListener('message', handler)
        if (event.data.error) {
          reject(new Error(event.data.error))
        } else {
          resolve(event.data)
        }
      }

      window.addEventListener('message', handler)
      window.postMessage({ type, requestId, ...payload }, window.location.origin)
    })
  }

  const fetchMyPosts = useCallback(async (limit = 20) => {
    setIsRunning(true)
    setError(null)
    setProgress('Fetching posts from LinkedIn...')
    try {
      const result = await sendExtensionMessage('GOVIRAL_LINKEDIN_FETCH_POSTS', { count: limit })
      if (result.posts?.length) {
        setProgress('Saving posts...')
        await apiClient.post('/posts/ingest', { platform: 'linkedin', posts: result.posts })
      }
      setProgress('')
      return result.posts || []
    } catch (err: any) {
      setError(err.message)
      throw err
    } finally {
      setIsRunning(false)
    }
  }, [])

  const fetchFeed = useCallback(async (limit = 20) => {
    setIsRunning(true)
    setError(null)
    setProgress('Fetching LinkedIn feed...')
    try {
      const result = await sendExtensionMessage('GOVIRAL_LINKEDIN_FETCH_FEED', { count: limit })
      if (result.posts?.length) {
        setProgress('Saving trending posts...')
        await apiClient.post('/trending/ingest', { platform: 'linkedin', posts: result.posts })
      }
      setProgress('')
      return result.posts || []
    } catch (err: any) {
      setError(err.message)
      throw err
    } finally {
      setIsRunning(false)
    }
  }, [])

  const searchPosts = useCallback(async (keywords: string, limit = 20) => {
    setIsRunning(true)
    setError(null)
    setProgress('Searching LinkedIn posts...')
    try {
      const result = await sendExtensionMessage('GOVIRAL_LINKEDIN_SEARCH_POSTS', { keywords, count: limit })
      if (result.posts?.length) {
        setProgress('Saving trending posts...')
        await apiClient.post('/trending/ingest', { platform: 'linkedin', posts: result.posts })
      }
      setProgress('')
      return result.posts || []
    } catch (err: any) {
      setError(err.message)
      throw err
    } finally {
      setIsRunning(false)
    }
  }, [])

  const fetchTrending = useCallback(async (keywords: string, limit = 20) => {
    setIsRunning(true)
    setError(null)
    setProgress('Discovering trending LinkedIn posts...')
    try {
      const result = await sendExtensionMessage('GOVIRAL_LINKEDIN_FETCH_TRENDING', { keywords, count: limit })
      if (result.posts?.length) {
        setProgress('Saving trending posts...')
        await apiClient.post('/trending/ingest', { platform: 'linkedin', posts: result.posts })
      }
      setProgress('')
      return result.posts || []
    } catch (err: any) {
      setError(err.message)
      throw err
    } finally {
      setIsRunning(false)
    }
  }, [])

  return { fetchMyPosts, fetchFeed, searchPosts, fetchTrending, isRunning, progress, error }
}
