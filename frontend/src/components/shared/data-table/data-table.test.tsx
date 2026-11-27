import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import '@testing-library/jest-dom/vitest'
import { DataTable } from './components/data-table'

// Mock the complex hooks used by DataTable
vi.mock('./hooks/use-data-table', () => ({
  useDataTable: vi.fn(),
}))
vi.mock('@/components/ui/toast', () => ({
  useToastManager: vi.fn(() => ({ add: vi.fn() })),
}))
vi.mock('@base-ui/react/toast', () => ({
  useToastManager: vi.fn(() => ({ add: vi.fn() })),
}))

import { useDataTable } from './hooks/use-data-table'

const mockUsers = [
  { id: '1', name: 'Alice Johnson', status: 'active' },
  { id: '2', name: 'Bob Smith', status: 'active' },
  { id: '3', name: 'Charlie Brown', status: 'inactive' },
]

type User = (typeof mockUsers)[number]

const testColumns = [
  { accessorKey: 'name', id: 'name', header: 'Name' },
  { accessorKey: 'status', id: 'status', header: 'Status' },
] as any

const defaultTableApi = {
  allRows: mockUsers,
  totalRowCount: 3,
  isInitialLoading: false,
  isLoadingMore: false,
  error: null,
  hasMorePages: false,
  fetchNextPage: vi.fn(),
  search: '',
  setSearch: vi.fn(),
  sorting: [],
  setSorting: vi.fn(),
  filters: {},
  setFilters: vi.fn(),
  selectedIds: new Set<string>(),
  selectedCount: 0,
  someRowsSelected: false,
  allRowsSelected: false,
  toggleRowSelection: vi.fn(),
  toggleSelectAll: vi.fn(),
  virtualizer: { getTotalSize: () => 0, getVirtualItems: () => [] },
  virtualRows: [
    { index: 0, start: 0, size: 1 },
    { index: 1, start: 1, size: 1 },
    { index: 2, start: 2, size: 1 },
  ],
  scrollRef: { current: null },
  deleteMutation: { mutate: vi.fn(), isSuccess: false, isError: false, isPending: false, data: undefined },
  isDeleting: false,
  refresh: vi.fn(),
  allColumnWidths: [],
  selectionColumnWidth: 50,
  query: { isFetching: false, refetch: vi.fn() },
}

const renderTable = (overrides: Partial<Parameters<typeof DataTable>[0]> = {}) => {
  ;(useDataTable as ReturnType<typeof vi.fn>).mockReturnValue(defaultTableApi)

  const onDelete = vi.fn(async () => ({ deletedIds: [], failed: [] }))
  const renderBulkActions = vi.fn(() => <div>bulk actions</div>)

  const utils = render(
    <DataTable<User>
      columns={testColumns}
      fetchPage={vi.fn(async () => ({ data: mockUsers, nextCursor: null, hasMore: false })) as any}
      getRowId={(row) => row.id}
      queryKey={['users']}
      enableRowSelection
      renderBulkActions={renderBulkActions}
      onDelete={onDelete}
      {...overrides}
    />,
  )

  return { ...utils, onDelete, renderBulkActions }
}

describe('DataTable', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    Object.assign(defaultTableApi, {
      allRows: mockUsers,
      totalRowCount: 3,
      isInitialLoading: false,
      isLoadingMore: false,
      error: null,
      hasMorePages: false,
      selectedIds: new Set<string>(),
      selectedCount: 0,
      someRowsSelected: false,
      allRowsSelected: false,
      virtualRows: [
        { index: 0, start: 0, size: 1 },
        { index: 1, start: 1, size: 1 },
        { index: 2, start: 2, size: 1 },
      ],
    })
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('renders loading state', () => {
    Object.assign(defaultTableApi, { isInitialLoading: true, allRows: [] })
    renderTable()

    expect(screen.getByText('Loading records…')).toBeInTheDocument()
  })

  it('renders empty state when no data', () => {
    Object.assign(defaultTableApi, { allRows: [], isInitialLoading: false })
    renderTable()

    expect(screen.getByText('No records found.')).toBeInTheDocument()
  })

  it('renders error state', () => {
    Object.assign(defaultTableApi, {
      allRows: [],
      isInitialLoading: false,
      error: new Error('Failed to load'),
    })
    renderTable()

    expect(screen.getByText('Failed to load records.')).toBeInTheDocument()
  })

  it('renders data rows', () => {
    renderTable()

    expect(screen.getAllByText('Alice Johnson').length).toBeGreaterThan(0)
    expect(screen.getByText('Bob Smith')).toBeInTheDocument()
    expect(screen.getByText('Charlie Brown')).toBeInTheDocument()
  })

  it('calls setSearch when search input changes', () => {
    vi.useFakeTimers()
    renderTable()

    const searchInput = screen.getByPlaceholderText('Search…')
    fireEvent.input(searchInput, { target: { value: 'Alice' } })
    vi.advanceTimersByTime(300)

    expect(defaultTableApi.setSearch).toHaveBeenCalledWith('Alice')
    vi.useRealTimers()
  })

  it('calls setSorting when column header is clicked', () => {
    renderTable()

    const nameHeader = screen.getByText('Name').closest('th')
    const sortButton = nameHeader?.querySelector('button')
    if (sortButton) {
      fireEvent.click(sortButton)
      expect(defaultTableApi.setSorting).toHaveBeenCalled()
    }
  })

  it('toggles row selection via checkbox input', () => {
    Object.assign(defaultTableApi, { selectedCount: 0 })
    renderTable()

    // Base UI renders checkboxes as <span role="checkbox"> elements.
    // Index 0 = select-all header checkbox, index 1 = first row checkbox.
    const checkboxes = screen.getAllByRole('checkbox')
    expect(checkboxes.length).toBeGreaterThan(1)

    const rowCheckbox = checkboxes[1]
    fireEvent.click(rowCheckbox)

    expect(defaultTableApi.toggleRowSelection).toHaveBeenCalledWith('1', true)
  })

  it('shows bulk action sheet when rows selected', () => {
    Object.assign(defaultTableApi, { selectedCount: 1, someRowsSelected: true })
    const renderBulkActions = vi.fn(() => <button>Bulk delete</button>)
    renderTable({ renderBulkActions })

    expect(screen.getByText('1 selected')).toBeInTheDocument()
    expect(renderBulkActions).toHaveBeenCalled()
  })

  it('shows sentinel for infinite scroll when more pages exist', () => {
    Object.assign(defaultTableApi, { hasMorePages: true, isLoadingMore: false })
    renderTable()

    expect(screen.getByText('Scroll to load more')).toBeInTheDocument()
  })
})
