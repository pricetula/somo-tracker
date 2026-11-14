export interface FilterWire {
  op: 'eq' | 'in' | 'between' | 'contains' | 'is_null';
  v: string | string[] | null;
}

export interface DataTableColumnMeta {
  label: string;
  align: 'left' | 'center' | 'right';
  sortKeys: string[];
  truncate?: boolean;
  money?: boolean;
  date?: boolean;
}

export interface DataTablePage<TData> {
  items: TData[];
  nextCursor: string | null;
}
