import type { ProgressEvent } from './types'

export const BASE_URL = import.meta.env.VITE_API_URL || '/api/v1'

class ApiError extends Error {
  status: number
  body: unknown

  constructor(status: number, body: unknown) {
    super(`API error ${status}`)
    this.name = 'ApiError'
    this.status = status
    this.body = body
  }
}

function getUserID(): string {
  let uid = localStorage.getItem('goviral_user_id')
  if (!uid) {
    uid = crypto.randomUUID()
    localStorage.setItem('goviral_user_id', uid)
  }
  return uid
}

function getHeaders(): Record<string, string> {
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    'X-User-ID': getUserID(),
  }
  const apiKey = localStorage.getItem('goviral_api_key')
  if (apiKey) {
    headers['Authorization'] = `Bearer ${apiKey}`
  }
  return headers
}

async function handleResponse<T>(response: Response): Promise<T> {
  if (!response.ok) {
    let body: unknown
    try {
      body = await response.json()
    } catch {
      body = await response.text()
    }
    throw new ApiError(response.status, body)
  }
  return response.json() as Promise<T>
}

function buildQuery(params?: Record<string, string | number | boolean | undefined>): string {
  if (!params) return ''
  const search = new URLSearchParams()
  for (const [key, value] of Object.entries(params)) {
    if (value !== undefined) {
      search.set(key, String(value))
    }
  }
  const str = search.toString()
  return str ? `?${str}` : ''
}

export const apiClient = {
  async get<T>(path: string, params?: Record<string, string | number | boolean | undefined>): Promise<T> {
    const response = await fetch(`${BASE_URL}${path}${buildQuery(params)}`, {
      headers: getHeaders(),
    })
    return handleResponse<T>(response)
  },

  async post<T>(path: string, body?: unknown): Promise<T> {
    const response = await fetch(`${BASE_URL}${path}`, {
      method: 'POST',
      headers: getHeaders(),
      body: body ? JSON.stringify(body) : undefined,
    })
    return handleResponse<T>(response)
  },

  async patch<T>(path: string, body?: unknown): Promise<T> {
    const response = await fetch(`${BASE_URL}${path}`, {
      method: 'PATCH',
      headers: getHeaders(),
      body: body ? JSON.stringify(body) : undefined,
    })
    return handleResponse<T>(response)
  },

  async delete(path: string): Promise<void> {
    const response = await fetch(`${BASE_URL}${path}`, {
      method: 'DELETE',
      headers: getHeaders(),
    })
    if (!response.ok) {
      let body: unknown
      try {
        body = await response.json()
      } catch {
        body = await response.text()
      }
      throw new ApiError(response.status, body)
    }
  },

  sse(
    path: string,
    body: unknown,
    onEvent: (event: ProgressEvent) => void,
  ): { cancel: () => void } {
    const controller = new AbortController()

    const run = async () => {
      const response = await fetch(`${BASE_URL}${path}`, {
        method: 'POST',
        headers: {
          ...getHeaders(),
          Accept: 'text/event-stream',
        },
        body: JSON.stringify(body),
        signal: controller.signal,
      })

      if (!response.ok || !response.body) {
        let errBody: unknown
        const raw = await response.text()
        try {
          errBody = JSON.parse(raw)
        } catch {
          errBody = raw
        }
        throw new ApiError(response.status, errBody)
      }

      const reader = response.body.getReader()
      const decoder = new TextDecoder()
      let buffer = ''

      while (true) {
        const { done, value } = await reader.read()
        if (done) break

        buffer += decoder.decode(value, { stream: true })
        const lines = buffer.split('\n')
        buffer = lines.pop() ?? ''

        for (const line of lines) {
          if (line.startsWith('data: ')) {
            try {
              const event = JSON.parse(line.slice(6)) as ProgressEvent
              onEvent(event)
            } catch {
              // skip malformed events
            }
          }
        }
      }
    }

    run().catch((err) => {
      if (err instanceof DOMException && err.name === 'AbortError') return
      onEvent({ type: 'error', message: String(err), percentage: 0 })
    })

    return { cancel: () => controller.abort() }
  },
}

export { ApiError }
