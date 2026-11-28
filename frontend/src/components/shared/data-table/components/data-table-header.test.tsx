import { describe, expect, it, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { DataTableHeader } from './data-table-header'
import type { ColumnDef, SortingState } from '@tanstack/react-table'

const columns = [
  { id: 'name', header: 'Name', accessorKey: 'name' },
  { id: 'status', header: 'Status', accessorKey: 'status' },
] as ColumnDef<any, any, unknown>[]

const defaultProps = {
  columns,
  sorting: [] as SortingState,
  setSorting: vi.fn(),
  enableRowSelection: true,
  someRowsSelected: false,
  allRowsSelected: false,
  onToggleSelectAll: vi.fn(),
  columnWidths: ['48px', '1fr', '1fr'],
  selectionColumnWidth: '48px',
  actionsColumnWidth: '56px',
}

describe('DataTableHeader', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders select-all checkbox when row selection is enabled', () => {
    render(<DataTableHeader {...defaultProps} />)

    const checkbox = screen.getByRole('checkbox', {
      name: /select all/i,
    })
    expect(checkbox).toBeInTheDocument()
  })

  it('does not render select-all checkbox when row selection is disabled', () => {
    render(<DataTableHeader {...defaultProps} enableRowSelection={false} />)

    expect(
      screen.queryByRole('checkbox', { name: /select all/i }),
    ).not.toBeInTheDocument()
  })

  it('calls onToggleSelectAll when select-all checkbox is clicked', () => {
    const onToggleSelectAll = vi.fn()
    render(<DataTableHeader {...defaultProps} onToggleSelectAll={onToggleSelectAll} />)

    const checkbox = screen.getByRole('checkbox', {
      name: /select all/i,
    })
    fireEvent.click(checkbox)

    expect(onToggleSelectAll).toHaveBeenCalledWith(true)
  })

  it('calls onToggleSelectAll with false when already selected and clicked again', () => {
    const onToggleSelectAll = vi.fn()
    render(
      <DataTableHeader
        {...defaultProps}
        allRowsSelected={true}
        onToggleSelectAll={onToggleSelectAll}
      />,
    )

    const checkbox = screen.getByRole('checkbox', {
      name: /select none/i,
    })
    fireEvent.click(checkbox)

    expect(onToggleSelectAll).toHaveBeenCalledWith(false)
  })

  it('updates aria-label based on selection state', () => {
    const { rerender } = render(<DataTableHeader {...defaultProps} />)

    expect(screen.getByRole('checkbox', { name: /select all/i })).toBeInTheDocument()

    rerender(<DataTableHeader {...defaultProps} allRowsSelected={true} />)

    expect(screen.getByRole('checkbox', { name: /select none/i })).toBeInTheDocument()
  })
})
