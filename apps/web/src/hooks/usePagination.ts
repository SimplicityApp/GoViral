import { useState, useMemo, useEffect } from 'react'

interface UsePaginationResult<T> {
  page: number
  totalPages: number
  pageItems: T[]
  nextPage: () => void
  prevPage: () => void
  hasNext: boolean
  hasPrev: boolean
  setPage: (page: number) => void
}

export function usePagination<T>(items: T[], perPage = 10): UsePaginationResult<T> {
  const [page, setPage] = useState(1)

  const totalPages = Math.max(1, Math.ceil(items.length / perPage))

  // Reset to page 1 when item count changes (e.g. filter changes)
  useEffect(() => {
    setPage(1)
  }, [items.length])

  const pageItems = useMemo(() => {
    const start = (page - 1) * perPage
    return items.slice(start, start + perPage)
  }, [items, page, perPage])

  return {
    page,
    totalPages,
    pageItems,
    nextPage: () => setPage((p) => Math.min(p + 1, totalPages)),
    prevPage: () => setPage((p) => Math.max(p - 1, 1)),
    hasNext: page < totalPages,
    hasPrev: page > 1,
    setPage: (p: number) => setPage(Math.max(1, Math.min(p, totalPages))),
  }
}
