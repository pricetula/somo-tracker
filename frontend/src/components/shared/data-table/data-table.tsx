'use client'

import { useCallback, useEffect, useRef } from 'react'
import { useVirtualizer } from '@tanstack/react-virtual'
import { cn } from 'cn'
import type { FilterWire } from './column-meta'
import type { SelectionStore } from './selection-store'

interface DataTableProps<TData, TFilterValue = Record<string, FilterWire>> {
  columns: any[]
  data: TData[]
  nextCursor: string | null
  isLoading: boolean
  isFetchingNextPage: boolean
  fetchNextPage: () => void
  onSortChange: (sort: string) => void
  onFilterChange: (filters: TFilterValue) => void
  rowHeight?: number
  selectionStore?: SelectionStore
  getRowId: (index: number, data: TData) => string
}

export function DataTable<TData, TFilterValue = Record<string, FilterWire>>({
  columns,
  data,
  nextCursor: _nextCursor,
  isLoading,
  isFetchingNextPage,
  fetchNextPage,
  onSortChange,
  onFilterChange: _onFilterChange,
  rowHeight = 48,
  selectionStore,
  getRowId,
}: DataTableProps<TData, TFilterValue>) {
  const parentRef = useRef<HTMLDivElement>(null)

  // Current sort direction, derived from the first column that declares sortKeys
  const currentSort = (() => {
    for (const col of columns) {
      const keys = col.meta?.sortKeys ?? []
      if (keys.length > 0) {
        return keys[0]
      }
    }
    return ''
  })()

  // TanStack Virtual v3: row-only virtualization over the flattened client cache.
  // count = rows loaded so far (grows as pages append). No DB total needed.
  const rowVirtualizer = useVirtualizer({
    count: data.length,
    getScrollElement: () => parentRef.current,
    estimateSize: () => rowHeight,
    overscan: 10,
  })

  const handleScroll = useCallback(() => {
    const scrollEl = parentRef.current
    if (!scrollEl) return
    const { scrollTop, scrollHeight, clientHeight } = scrollEl
    if (
      scrollTop + clientHeight >= scrollHeight - 200 &&
      !isFetchingNextPage
    ) {
      fetchNextPage()
    }
  }, [fetchNextPage, isFetchingNextPage])

  useEffect(() => {
    const scrollEl = parentRef.current
    if (!scrollEl) return
    scrollEl.addEventListener('scroll', handleScroll)
    return () => scrollEl.removeEventListener('scroll', handleScroll)
  }, [handleScroll])

  // Correct TanStack Virtual v3 API: getVirtualItems() returns items shaped
  // { index, start, end, size, offset }. There is no getItemPosition() or
  // getItemSize() on the v3 Virtualizer — calling them crashes with
  // 'getItemPosition is not a function'. Row positioning uses
  // virtualItem.start / virtualItem.size; scroll spacer uses getTotalSize().
  const virtualItems = rowVirtualizer.getVirtualItems()

  // Build a shared grid template so header cells and body cells line up
  // vertically. Columns with an explicit width use it; others share 1fr.
  const gridTemplateColumns = columns
    .map((col) => col.meta?.width ?? '1fr')
    .join(' ')

  const isRowSelected = (id: string) => {
    if (!selectionStore) return false
    return selectionStore.selected.has(id)
  }

  const toggleRowSelection = (id: string) => {
    if (!selectionStore) return
    selectionStore.toggle(id)
  }

  const toggleSelectAll = () => {
    if (!selectionStore) return
    const allIds = data.map((_, i) => getRowId(i, data[i]))
    const allSelected = allIds.every((id) => selectionStore.selected.has(id))
    if (allSelected) {
      selectionStore.selectNone()
    } else {
      selectionStore.selectAllLoaded(allIds)
    }
  }

  const allSelected =
    data.length > 0 &&
    data.every((_, i) => selectionStore?.selected.has(getRowId(i, data[i])) ?? false)

  const handleSort = (sortKey: string) => {
    const lastUnderscore = sortKey.lastIndexOf('_')
    if (lastUnderscore > 0) {
      const field = sortKey.slice(0, lastUnderscore)
      const dir = sortKey.slice(lastUnderscore + 1)
      onSortChange(`${field}_${dir}`)
    }
  }

  const renderHeaderCell = (column: any) => {
    if (column.id === 'selection') {
      return (
        <div
          key={column.id}
          className="px-2 text-center flex items-center justify-center"
          style={{ width: column.meta?.width ?? '48px' }}
        >
          <input
            type="checkbox"
            checked={allSelected}
            onChange={toggleSelectAll}
            className="w-4 h-4 rounded border-input text-primary"
            aria-label="Select all rows"
          />
        </div>
      )
    }
    const sortKeys = column.meta?.sortKeys ?? []
    const isSortedAsc = currentSort === `${column.id}_asc`
    const isSortedDesc = currentSort === `${column.id}_desc`
    return (
      <div
        key={column.id}
        className={cn(
          'px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider cursor-pointer select-none',
          column.meta?.align === 'right' ? 'text-right' : '',
          column.meta?.align === 'center' ? 'text-center' : ''
        )}
        style={{ width: column.meta?.width }}
        onClick={() => {
          if (sortKeys.length > 0) {
            const currentIdx = sortKeys.indexOf(currentSort)
            const nextIdx = (currentIdx + 1) % sortKeys.length
            handleSort(sortKeys[nextIdx])
          }
        }}
      >
        <div className="flex items-center gap-1">
          {column.meta?.label ?? column.id}
          {isSortedAsc && (
            <span className="ml-1 text-xs text-muted-foreground">▲</span>
          )}
          {isSortedDesc && (
            <span className="ml-1 text-xs text-muted-foreground">▼</span>
          )}
        </div>
      </div>
    )
  }

  const renderBodyCell = (column: any, row: TData, id: string, isSelected: boolean) => {
    if (column.id === 'selection') {
      return (
        <div
          key={column.id}
          className="px-2 text-center flex items-center justify-center"
          style={{ width: column.meta?.width ?? '48px' }}
        >
          <input
            type="checkbox"
            checked={isSelected}
            onChange={() => toggleRowSelection(id)}
            className="w-4 h-4 rounded border-input text-primary"
            aria-label="Select row"
          />
        </div>
      )
    }
    return (
      <div
        key={column.id}
        className={cn(
          'px-4 py-3 text-sm overflow-hidden text-ellipsis whitespace-nowrap',
          column.meta?.align === 'right' ? 'text-right' : '',
          column.meta?.align === 'center' ? 'text-center' : '',
          column.meta?.align === 'left' ? 'text-left' : ''
        )}
        style={{
          width: column.meta?.width,
          textAlign:
            column.meta?.align === 'right'
              ? 'right'
              : column.meta?.align === 'center'
                ? 'center'
                : 'left',
        }}
      >
        {(row as any)[column.accessorKey ?? column.id] ?? ''}
      </div>
    )
  }

  const rowClasses = (isSelected: boolean) =>
    cn(
      'transition-colors hover:bg-muted/50',
      isSelected && 'bg-accent/50 text-accent-foreground'
    )

  return (
    <div
      ref={parentRef}
      className="relative w-full bg-background border border-border rounded-md overflow-hidden flex flex-col"
      style={{ height: 'calc(100vh - 200px)' }}
    >
      {/* Header */}
      <div
        className="border-b border-border bg-background/50 z-10"
        style={{ display: 'grid', gridTemplateColumns }}
      >
        {columns.map(renderHeaderCell)}
      </div>

      {/* Body */}
      <div className="w-full overflow-y-auto" style={{ flex: 1, minHeight: 0 }}>
        <div
          className="relative w-full min-h-full"
          style={{
            height: `${rowVirtualizer.getTotalSize()}px`,
            width: '100%',
          }}
        >
          {isLoading && data.length === 0 && (
            <div className="flex items-center justify-center h-48">
              <div className="animate-pulse bg-muted rounded w-full h-8" />
            </div>
          )}

          {data.length === 0 && !isLoading && (
            <div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
              <p className="text-sm">No results</p>
            </div>
          )}

          {virtualItems.map((virtualItem) => {
            const globalIndex = virtualItem.index
            const row = data[globalIndex]
            if (!row) return null
            const id = getRowId(globalIndex, row)
            const isSelected = isRowSelected(id)

            return (
              <div
                key={id}
                className={cn('absolute left-0 right-0 border-b border-border', rowClasses(isSelected))}
                style={{
                  top: `${virtualItem.start}px`,
                  height: `${virtualItem.size}px`,
                  display: 'grid',
                  gridTemplateColumns,
                }}
              >
                {columns.map((column) =>
                  renderBodyCell(column, row, id, isSelected)
                )}
              </div>
            )
          })}
        </div>
      </div>

      {isFetchingNextPage && (
        <div className="px-4 py-2 text-center text-xs text-muted-foreground bg-background/50 border-t border-border">
          Loading more…
        </div>
      )}
    </div>
  )
}
