import { apiFetch } from '@/lib/api/client'
import type { FilterWire, DataTablePage } from './column-meta'

function serializeFilters(filters: Record<string, FilterWire>): string {
  const wire: Record<string, FilterWire> = {};
  for (const [field, filter] of Object.entries(filters)) {
    const snakeField = field.replace(/[A-Z]/g, (m) => `_${m.toLowerCase()}`);
    wire[snakeField] = filter;
  }
  return btoa(JSON.stringify(wire)).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
}

export function createListQueryFn<TData>(endpoint: string) {
  return async function listQueryFn({
    queryKey,
    signal,
  }: {
    queryKey: readonly unknown[];
    signal: AbortSignal;
  }): Promise<DataTablePage<TData>> {
    const [, limit, cursor, sort, filters] = queryKey as unknown as [
      string,
      number,
      string | null,
      string,
      Record<string, FilterWire>,
    ];

    const params = new URLSearchParams();
    params.set('limit', String(limit));
    if (cursor) {
      params.set('cursor', cursor);
    }
    if (sort) {
      params.set('sort', sort);
    }
    const filtersObj = filters as Record<string, FilterWire>;
    if (filtersObj && Object.keys(filtersObj).length > 0) {
      params.set('filters', serializeFilters(filtersObj));
    }

    const path = `${endpoint}?${params.toString()}`;

    const response = await apiFetch<DataTablePage<TData>>(path, {
      signal,
    });

    return response;
  };
}
