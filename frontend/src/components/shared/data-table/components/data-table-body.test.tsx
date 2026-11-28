import { describe, expect, it, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { DataTableBody } from './data-table-body'
import type { ColumnDef, RowData } from '@tanstack/react-table'

type User = { id: string; name: string; status: string }

const columns = [
  { id: 'name', header: 'Name', accessorKey: 'name' },
  { id: 'status', header: 'Status', accessorKey: 'status' },
] as ColumnDef<any, User, unknown>[]

const rows: User[] = [
  { id: '1', name: 'Alice', status: 'active' },
  { id: '2', name: 'Bob', status: 'inactive' },
]

const defaultProps = {
  columns,
  virtualRows: [
    { index: 0, start: 0, size: 44 },
    { index: 1, start: 44, size: 44 },
  ],
  allRows: rows,
  getRowId: (row: User) => row.id,
  enableRowSelection: true,
  virtualized: false,
  selectedIds: new Set<string>(),
  onToggleRowSelection: vi.fn(),
  columnWidths: ['48px', '1fr', '1fr'],
  selectionColumnWidth: '48px',
  actionsColumnWidth: '56px',
  refresh: vi.fn(),
}

describe('DataTableBody', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders row checkboxes when row selection is enabled', () => {
    render(<DataTableBody {...defaultProps} />)

    const checkboxes = screen.getAllByRole('checkbox', {
      name: /select row /i,
    })
    expect(checkboxes).toHaveLength(2)
  })

  it('does not render row checkboxes when row selection is disabled', () => {
    render(<DataTableBody {...defaultProps} enableRowSelection={false} />)

    expect(
      screen.queryAllByRole('checkbox', { name: /select row /i }),
    ).toHaveLength(0)
  })

  it('calls onToggleRowSelection when a row checkbox is clicked', () => {
    const onToggleRowSelection = vi.fn()
    render(<DataTableBody {...defaultProps} onToggleRowSelection={onToggleRowSelection} />)

    const checkboxes = screen.getAllByRole('checkbox', {
      name: /select row /i,
    })
    fireEvent.click(checkboxes[0])

    expect(onToggleRowSelection).toHaveBeenCalledWith('1', true)
  })

  it('calls onToggleRowSelection with false when checkbox is already checked', () => {
    const onToggleRowSelection = vi.fn()
    render(
      <DataTableBody
        {...defaultProps}
        selectedIds={new Set(['1'])}
        onToggleRowSelection={onToggleRowSelection}
      />,
    )

    const checkboxes = screen.getAllByRole('checkbox', {
      name: /select row /i,
    })
    fireEvent.click(checkboxes[0])

    expect(onToggleRowSelection).toHaveBeenCalledWith('1', false)
  })

  it('shows selected state on checked row', () => {
    render(
      <DataTableBody
        {...defaultProps}
        selectedIds={new Set(['1'])}
      />,
    )

    const firstRow = screen.getByText('Alice').closest('[role="row"]')
    expect(firstRow).toHaveAttribute('data-selected', 'true')

    const secondRow = screen.getByText('Bob').closest('[role="row"]')
    expect(secondRow).not.toHaveAttribute('data-selected', 'true')
  })

  it('does not call onToggleRowSelection when row selection is disabled', () => {
    const onToggleRowSelection = vi.fn()
    render(
      <DataTableBody
        {...defaultProps}
        enableRowSelection={false}
        onToggleRowSelection={onToggleRowSelection}
      />,
    )

    const checkboxes = screen.queryAllByRole('checkbox', {
      name: /select row /i,
    })
    expect(checkboxes).toHaveLength(0)
    expect(onToggleRowSelection).not.toHaveBeenCalled()
  })
})
