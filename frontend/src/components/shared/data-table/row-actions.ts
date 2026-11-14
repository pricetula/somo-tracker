export interface RowAction<TData> {
  id: string
  label: string
  href?: (row: TData) => string
  variant?: 'default' | 'destructive'
  confirm?: {
    title: string
    description?: (row: TData) => string
    confirmLabel: string
  }
  onSelect?: (row: TData) => void
}

export interface BulkAction {
  id: string
  label: string
  variant?: 'default' | 'destructive'
  confirm: {
    title: string
    description: (selectedCount: number) => string
    confirmLabel: string
  }
  onSelect: (ids: string[]) => void | Promise<void>
  maxChunkSize?: number // default 100; chunks oversized selections for API caps
}
