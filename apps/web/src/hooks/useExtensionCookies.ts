import { useState, useEffect, useCallback, useRef } from 'react'

interface ExtensionInfo {
  available: boolean
  version: string | null
}

interface ExtractedCookies {
  x: { auth_token: string; ct0: string } | null
  linkedin: { li_at: string; jsessionid: string } | null
}

const EXTRACT_TIMEOUT = 10_000

export function useExtensionCookies() {
  const [extension, setExtension] = useState<ExtensionInfo>({ available: false, version: null })
  const [extracting, setExtracting] = useState(false)
  const pendingRef = useRef<{
    resolve: (v: ExtractedCookies) => void
    reject: (e: Error) => void
    requestId: string
  } | null>(null)

  useEffect(() => {
    function handleMessage(event: MessageEvent) {
      if (event.origin !== window.location.origin) return
      if (event.source !== window) return

      const { type, version, requestId, success, cookies, error } = event.data || {}

      if (type === 'GOVIRAL_EXTENSION_AVAILABLE' || type === 'GOVIRAL_PONG') {
        setExtension({ available: true, version: version || null })
        return
      }

      if (type === 'GOVIRAL_COOKIES_RESULT' && pendingRef.current) {
        if (requestId !== pendingRef.current.requestId) return
        const { resolve, reject } = pendingRef.current
        pendingRef.current = null
        setExtracting(false)

        if (success) {
          resolve(cookies as ExtractedCookies)
        } else {
          reject(new Error(error || 'Cookie extraction failed'))
        }
      }
    }

    window.addEventListener('message', handleMessage)

    // Ping the extension to check availability
    window.postMessage({ type: 'GOVIRAL_PING', requestId: 'init' }, window.location.origin)

    return () => window.removeEventListener('message', handleMessage)
  }, [])

  const extractCookies = useCallback((): Promise<ExtractedCookies> => {
    return new Promise((resolve, reject) => {
      const requestId = crypto.randomUUID()
      pendingRef.current = { resolve, reject, requestId }
      setExtracting(true)

      window.postMessage(
        { type: 'GOVIRAL_EXTRACT_COOKIES', requestId },
        window.location.origin,
      )

      setTimeout(() => {
        if (pendingRef.current?.requestId === requestId) {
          pendingRef.current = null
          setExtracting(false)
          reject(new Error('Cookie extraction timed out'))
        }
      }, EXTRACT_TIMEOUT)
    })
  }, [])

  return { extension, extracting, extractCookies }
}
