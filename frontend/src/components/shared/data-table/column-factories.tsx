import type { ColumnDef } from '@tanstack/react-table'
import { Link } from '@tanstack/react-router'
import type { FilterWire, DataTableColumnMeta } from './column-meta'

export type { ColumnDef } from '@tanstack/react-table'

export function textColumn<TData>(config: {
  accessorKey: keyof TData & string
  label: string
  sortKeys?: string[]
  truncate?: boolean
  align?: 'left' | 'center' | 'right'
  width?: string
}): ColumnDef<any, any> {
  return {
    id: config.accessorKey as string,
    accessorKey: config.accessorKey as string,
    header: ({ column }: any) => {
      const isSorted = column.getIsSorted()
      return (
        <div className="flex items-center gap-1">
          {column.columnDef.meta?.label ?? config.label}
          {isSorted && (
            <span className="ml-1 text-xs text-muted-foreground">
              {isSorted === 'asc' ? '▲' : '▼'}
            </span>
          )}
        </div>
      )
    },
    cell: ({ getValue }: any) => {
      const value = getValue()
      return (
        <div
          className={`text-sm ${config.align === 'right' ? 'text-right' : config.align === 'center' ? 'text-center' : 'text-left'}${
            config.truncate ? ' truncate max-w-[200px]' : ''
          }`}
          title={typeof value === 'string' ? value : undefined}
        >
          {value ?? ''}
        </div>
      )
    },
    meta: {
      label: config.label,
      align: config.align ?? 'left',
      sortKeys: config.sortKeys ?? [],
      truncate: config.truncate,
      width: config.width,
    } as DataTableColumnMeta,
  }
}

export function numberColumn<TData>(config: {
  accessorKey: keyof TData & string
  label: string
  sortKeys?: string[]
  align?: 'left' | 'center' | 'right'
  width?: string
}): ColumnDef<any, any> {
  return {
    id: config.accessorKey as string,
    accessorKey: config.accessorKey as string,
    header: ({ column }: any) => {
      const isSorted = column.getIsSorted()
      return (
        <div className="flex items-center gap-1">
          {column.columnDef.meta?.label ?? config.label}
          {isSorted && (
            <span className="ml-1 text-xs text-muted-foreground">
              {isSorted === 'asc' ? '▲' : '▼'}
            </span>
          )}
        </div>
      )
    },
    cell: ({ getValue }: any) => {
      const value = getValue()
      return (
        <div className="text-sm text-right font-mono tabular-nums">
          {value ?? ''}
        </div>
      )
    },
    meta: {
      label: config.label,
      align: 'right',
      sortKeys: config.sortKeys ?? [],
      width: config.width,
    } as DataTableColumnMeta,
  }
}

export function dateColumn<TData>(config: {
  accessorKey: keyof TData & string
  label: string
  sortKeys?: string[]
  align?: 'left' | 'center' | 'right'
  width?: string
}): ColumnDef<any, any> {
  return {
    id: config.accessorKey as string,
    accessorKey: config.accessorKey as string,
    header: ({ column }: any) => {
      const isSorted = column.getIsSorted()
      return (
        <div className="flex items-center gap-1">
          {column.columnDef.meta?.label ?? config.label}
          {isSorted && (
            <span className="ml-1 text-xs text-muted-foreground">
              {isSorted === 'asc' ? '▲' : '▼'}
            </span>
          )}
        </div>
      )
    },
    cell: ({ getValue }: any) => {
      const value = getValue()
      return (
        <div className="text-sm text-muted-foreground">
          {value ?? ''}
        </div>
      )
    },
    meta: {
      label: config.label,
      align: config.align ?? 'left',
      sortKeys: config.sortKeys ?? [],
      date: true,
      width: config.width,
    } as DataTableColumnMeta,
  }
}

export function badgeColumn<TData>(config: {
  accessorKey: keyof TData & string
  label: string
  sortKeys?: string[]
  variants: Record<string, string>
  defaultVariant?: string
  width?: string
}): ColumnDef<any, any> {
  return {
    id: config.accessorKey as string,
    accessorKey: config.accessorKey as string,
    header: ({ column }: any) => {
      const isSorted = column.getIsSorted()
      return (
        <div className="flex items-center gap-1">
          {column.columnDef.meta?.label ?? config.label}
          {isSorted && (
            <span className="ml-1 text-xs text-muted-foreground">
              {isSorted === 'asc' ? '▲' : '▼'}
            </span>
          )}
        </div>
      )
    },
    cell: ({ getValue }: any) => {
      const value = getValue()
      const variantClass =
        config.variants[String(value)] ?? config.defaultVariant ?? 'bg-muted text-muted-foreground'
      return (
        <span
          className={`text-xs font-medium px-2 py-0.5 rounded-full ${variantClass}`}
        >
          {value}
        </span>
      )
    },
    meta: {
      label: config.label,
      align: 'left',
      sortKeys: config.sortKeys ?? [],
      width: config.width,
    } as DataTableColumnMeta,
  }
}

export function selectColumn<TData>(): ColumnDef<any, any> {
  return {
    id: 'selection',
    header: ({ getIsAllSelected, toggleSelectAll }: any) => (
      <div className="flex items-center justify-center">
        <input
          type="checkbox"
          checked={getIsAllSelected()}
          onChange={toggleSelectAll}
          className="w-4 h-4 rounded border-input text-primary"
          aria-label="Select all rows"
        />
      </div>
    ),
    cell: ({ row }: any) => (
      <div className="flex items-center justify-center">
        <input
          type="checkbox"
          checked={row.getIsSelected()}
          onChange={() => row.toggleSelected()}
          className="w-4 h-4 rounded border-input text-primary"
          aria-label={`Select row ${row.index + 1}`}
        />
      </div>
    ),
    meta: {
      label: '',
      align: 'center',
      sortKeys: [],
    } as DataTableColumnMeta,
  }
}

export function linkColumn<TData>(config: {
  accessorKey: keyof TData & string
  label: string
  sortKeys?: string[]
  to: (row: TData) => string
  width?: string
}): ColumnDef<any, any> {
  return {
    id: config.accessorKey as string,
    accessorKey: config.accessorKey as string,
    header: ({ column }: any) => {
      const isSorted = column.getIsSorted()
      return (
        <div className="flex items-center gap-1">
          {column.columnDef.meta?.label ?? config.label}
          {isSorted && (
            <span className="ml-1 text-xs text-muted-foreground">
              {isSorted === 'asc' ? '▲' : '▼'}
            </span>
          )}
        </div>
      )
    },
    cell: ({ getValue, row }: any) => {
      const value = getValue()
      const href = config.to(row.original)
      return (
        <Link to={href} className="text-sm text-primary hover:underline">
          {value ?? ''}
        </Link>
      )
    },
    meta: {
      label: config.label,
      align: 'left',
      sortKeys: config.sortKeys ?? [],
      width: config.width,
    } as DataTableColumnMeta,
  }
}
