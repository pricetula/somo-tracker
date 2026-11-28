import type { ColumnDef, RowData, SortingState } from "@tanstack/react-table";
import type { QueryKey } from "@tanstack/react-query";

/**
 * ADR-0027: Reusable Server-Driven DataTable
 *
 * Shared type contract. The component is generic over the record type T so the
 * same component can render students, teachers, classes, fees, etc.
 */

export interface DataTablePage<T> {
  data: T[];
  nextCursor: string | null;
  hasMore: boolean;
}

export interface DataTableBulkActionContext {
  selectedIds: string[];
  selectedCount: number;
  clearSelection: () => void;
  refresh: () => void;
}

export interface DataTableRowActionContext<T> {
  row: T;
  rowId: string;
  refresh: () => void;
}

export interface BulkDeleteResult {
  deletedIds: string[];
  failed: {
    id: string;
    code?: string;
    message: string;
  }[];
}

export interface DataTableFilterOption {
  label: string;
  value: string;
  disabled?: boolean;
}

export interface DataTableFilter {
  id: string;
  label: string;
  type: "select" | "multi-select";
  options: DataTableFilterOption[];
  searchable?: boolean;
  placeholder?: string;
  group?: string;
  defaultValue?: string | string[];
}

export interface DataTableProps<T extends RowData> {
  queryKey: QueryKey;

  columns: ColumnDef<any, T, unknown>[];

  fetchPage: (params: {
    cursor: string | null;
    limit: number;
    search?: string;
    sorting: SortingState;
    filters: Record<string, unknown>;
  }) => Promise<DataTablePage<T>>;

  getRowId: (row: T) => string;

  pageSize?: number;

  initialFilters?: Record<string, unknown>;

  initialSorting?: SortingState;

  filters?: DataTableFilter[];

  renderBulkActions?: (context: DataTableBulkActionContext) => React.ReactNode;

  renderRowActions?: (context: DataTableRowActionContext<T>) => React.ReactNode;

  onDelete?: (ids: string[]) => Promise<BulkDeleteResult>;

  /**
   * Optional row click handler. Not triggered when clicking the selection
   * checkbox or the row actions dropdown.
   */
  onRowClick?: (row: T, rowId: string) => void;

  enableRowSelection?: boolean;

  enableVirtualization?: boolean;
}
