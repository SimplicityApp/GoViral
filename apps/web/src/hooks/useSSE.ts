import { useState, useCallback, useRef } from 'react'
import { apiClient } from '@/lib/api'
import type { ProgressEvent } from '@/lib/types'

interface SSEState<T> {
  progress: ProgressEvent | null
  isRunning: boolean
  result: T | null
  error: string | null
}

export function useSSEMutation<T>(path: string) {
  const [state, setState] = useState<SSEState<T>>({
    progress: null,
    isRunning: false,
    result: null,
    error: null,
  })
  const cancelRef = useRef<(() => void) | null>(null)

  const mutate = useCallback(
    (body: unknown) => {
      setState({ progress: null, isRunning: true, result: null, error: null })

      const { cancel } = apiClient.sse(path, body, (event) => {
        if (event.type === 'progress') {
          setState((prev) => ({ ...prev, progress: event }))
        } else if (event.type === 'complete') {
          setState({
            progress: event,
            isRunning: false,
            result: event.data as T,
            error: null,
          })
        } else if (event.type === 'error') {
          setState((prev) => ({
            ...prev,
            isRunning: false,
            error: event.message,
          }))
        }
      })

      cancelRef.current = cancel
    },
    [path],
  )

  const cancel = useCallback(() => {
    cancelRef.current?.()
    setState((prev) => ({ ...prev, isRunning: false }))
  }, [])

  return { ...state, mutate, cancel }
}
