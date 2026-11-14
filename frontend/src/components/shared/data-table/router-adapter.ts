import type { FilterWire } from './column-meta';

interface TableState {
  limit: number;
  cursor: string | null;
  sort: string;
  filters: Record<string, FilterWire>;
}

function serializeFilters(filters: Record<string, FilterWire>): string {
  const wire: Record<string, FilterWire> = {};
  for (const [field, filter] of Object.entries(filters)) {
    const snakeField = field.replace(/[A-Z]/g, (m) => `_${m.toLowerCase()}`);
    wire[snakeField] = filter;
  }
  return btoa(JSON.stringify(wire)).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
}

function parseFilters(encoded: string): Record<string, FilterWire> {
  try {
    const decoded = atob(encoded.replace(/-/g, '+').replace(/_/g, '/'));
    return JSON.parse(decoded);
  } catch {
    return {};
  }
}

export function getSearchParams(): TableState {
  if (typeof window === 'undefined') {
    return { limit: 50, cursor: null, sort: '', filters: {} };
  }

  const params = new URLSearchParams(window.location.search);
  const limit = parseInt(params.get('limit') ?? '50', 10) || 50;
  const cursor = params.get('cursor') || null;
  const sort = params.get('sort') || '';
  const filters = params.get('filters') ? parseFilters(params.get('filters')!) : {};

  return { limit, cursor, sort, filters };
}

export function setSearchParams(
  state: Partial<TableState>,
  opts: { push?: boolean } = {}
): void {
  if (typeof window === 'undefined') {
    return;
  }

  const params = new URLSearchParams(window.location.search);

  if (state.limit !== undefined) {
    params.set('limit', String(state.limit));
  }
  if (state.cursor !== undefined) {
    if (state.cursor) {
      params.set('cursor', state.cursor);
    } else {
      params.delete('cursor');
    }
  }
  if (state.sort !== undefined) {
    params.set('sort', state.sort);
  }
  if (state.filters !== undefined) {
    if (Object.keys(state.filters).length > 0) {
      params.set('filters', serializeFilters(state.filters));
    } else {
      params.delete('filters');
    }
  }

  const url = `${window.location.pathname}?${params.toString()}`;
  if (opts.push) {
    window.history.pushState({}, '', url);
  } else {
    window.history.replaceState({}, '', url);
  }
}

export function serialize(state: TableState): string {
  const params = new URLSearchParams();
  params.set('limit', String(state.limit));
  if (state.cursor) params.set('cursor', state.cursor);
  if (state.sort) params.set('sort', state.sort);
  if (state.filters && Object.keys(state.filters).length > 0) {
    params.set('filters', serializeFilters(state.filters));
  }
  return params.toString();
}

export function parse(searchParams: string): TableState {
  const params = new URLSearchParams(searchParams);
  return {
    limit: parseInt(params.get('limit') ?? '50', 10) || 50,
    cursor: params.get('cursor') || null,
    sort: params.get('sort') || '',
    filters: params.get('filters') ? parseFilters(params.get('filters')!) : {},
  };
}
