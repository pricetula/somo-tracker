import { describe, expect, it, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { QueryClient } from "@tanstack/react-query";
import {
  useInfiniteQuery,
  useMutation,
  useQueryClient,
} from "@tanstack/react-query";
import { useVirtualizer } from "@tanstack/react-virtual";

import { useDataTable } from "./use-data-table";
import { useDataTableUrlState } from "./use-data-table-url-state";
import type { DataTablePage } from "../types";

vi.mock("@tanstack/react-query", () => ({
  useInfiniteQuery: vi.fn(),
  useMutation: vi.fn((options: any) => {
    return {
      mutate: vi.fn((ids: string[]) => {
        if (options.mutationFn) {
          options.mutationFn(ids);
        }
      }),
      isSuccess: false,
      isError: false,
      isPending: false,
      data: undefined,
    };
  }),
  useQueryClient: vi.fn(),
}));
vi.mock("@tanstack/react-virtual", () => ({
  useVirtualizer: vi.fn(),
}));
vi.mock("./use-data-table-url-state", () => ({
  useDataTableUrlState: vi.fn(),
}));

const mockFetchPage = vi.fn();
const mockOnDelete = vi.fn();
const mockUrlState = {
  search: "",
  setSearch: vi.fn(),
  sorting: [],
  setSorting: vi.fn(),
  filters: {},
  setFilters: vi.fn(),
};

const mockQueryClient = {
  cancelQueries: vi.fn().mockResolvedValue(undefined),
  getQueryData: vi.fn(),
  setQueryData: vi.fn(),
  invalidateQueries: vi.fn(),
};

const mockVirtualizer = {
  getVirtualItems: vi.fn(() => []),
  measure: vi.fn(),
};

const buildResult = (overrides: Partial<DataTablePage<{ id: string }>> = {}) => ({
  data: [{ id: "1" }, { id: "2" }],
  nextCursor: null,
  hasMore: false,
  ...overrides,
});

let mutationOverride: any = null;

vi.mock("@tanstack/react-query", () => ({
  useInfiniteQuery: vi.fn(),
  useMutation: vi.fn((options: any) => {
    return {
      mutate: vi.fn((ids: string[]) => {
        if (options.mutationFn) {
          options.mutationFn(ids);
        }
      }),
      isSuccess: mutationOverride?.isSuccess ?? false,
      isError: mutationOverride?.isError ?? false,
      isPending: mutationOverride?.isPending ?? false,
      data: mutationOverride?.data ?? undefined,
    };
  }),
  useQueryClient: vi.fn(),
}));

const setupHook = (
  queryResult: any,
  mutationResult: any,
  urlState: any = mockUrlState,
) => {
  mutationOverride = mutationResult;
  (useInfiniteQuery as ReturnType<typeof vi.fn>).mockReturnValue(queryResult);
  (useQueryClient as ReturnType<typeof vi.fn>).mockReturnValue(mockQueryClient);
  (useDataTableUrlState as ReturnType<typeof vi.fn>).mockReturnValue(urlState);
  (useVirtualizer as ReturnType<typeof vi.fn>).mockReturnValue(mockVirtualizer);
};

describe("useDataTable", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockFetchPage.mockReset();
    mockOnDelete.mockReset();
    mockUrlState.setSearch.mockClear();
    mockUrlState.setSorting.mockClear();
    mockUrlState.setFilters.mockClear();
    mockQueryClient.cancelQueries.mockClear();
    mockQueryClient.getQueryData.mockClear();
    mockQueryClient.setQueryData.mockClear();
    mockQueryClient.invalidateQueries.mockClear();
    mockVirtualizer.getVirtualItems.mockClear();
    mockVirtualizer.measure.mockClear();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("returns initial loading state before first page resolves", () => {
    setupHook(
      {
        isFetching: true,
        data: { pages: [] },
        error: null,
        fetchNextPage: vi.fn(),
      },
      { isSuccess: false, isError: false, isPending: false, mutate: vi.fn() },
    );

    const { result } = renderHook(() =>
      useDataTable<{ id: string }>({
        queryKey: ["test"],
        columns: [],
        fetchPage: mockFetchPage,
        getRowId: (row) => row.id,
      }),
    );

    expect(result.current.isInitialLoading).toBe(true);
    expect(result.current.isLoadingMore).toBe(false);
    expect(result.current.allRows).toEqual([]);
  });

  it("returns loaded rows from all pages", () => {
    setupHook(
      {
        isFetching: false,
        data: {
          pages: [
            { data: [{ id: "1" }], nextCursor: "c1", hasMore: true },
            { data: [{ id: "2" }], nextCursor: null, hasMore: false },
          ],
        },
        error: null,
        fetchNextPage: vi.fn(),
      },
      { isSuccess: false, isError: false, isPending: false, mutate: vi.fn() },
    );

    const { result } = renderHook(() =>
      useDataTable<{ id: string }>({
        queryKey: ["test"],
        columns: [],
        fetchPage: mockFetchPage,
        getRowId: (row) => row.id,
      }),
    );

    expect(result.current.allRows).toEqual([{ id: "1" }, { id: "2" }]);
    expect(result.current.totalRowCount).toBe(2);
    expect(result.current.hasMorePages).toBe(false);
  });

  it("toggles row selection", () => {
    setupHook(
      {
        isFetching: false,
        data: { pages: [{ data: [{ id: "1" }], nextCursor: null, hasMore: false }] },
        error: null,
        fetchNextPage: vi.fn(),
      },
      { isSuccess: false, isError: false, isPending: false, mutate: vi.fn() },
    );

    const { result } = renderHook(() =>
      useDataTable<{ id: string }>({
        queryKey: ["test"],
        columns: [],
        fetchPage: mockFetchPage,
        getRowId: (row) => row.id,
        enableRowSelection: true,
      }),
    );

    act(() => {
      result.current.toggleRowSelection("1", true);
    });

    expect(result.current.selectedIds).toEqual(new Set(["1"]));
    expect(result.current.selectedCount).toBe(1);
    expect(result.current.someRowsSelected).toBe(true);
    expect(result.current.allRowsSelected).toBe(true);

    act(() => {
      result.current.toggleRowSelection("1", false);
    });

    expect(result.current.selectedIds).toEqual(new Set());
    expect(result.current.selectedCount).toBe(0);
    expect(result.current.someRowsSelected).toBe(false);
    expect(result.current.allRowsSelected).toBe(false);
  });

  it("select all selects every loaded row", () => {
    setupHook(
      {
        isFetching: false,
        data: {
          pages: [
            {
              data: [{ id: "1" }, { id: "2" }],
              nextCursor: null,
              hasMore: false,
            },
          ],
        },
        error: null,
        fetchNextPage: vi.fn(),
      },
      { isSuccess: false, isError: false, isPending: false, mutate: vi.fn() },
    );

    const { result } = renderHook(() =>
      useDataTable<{ id: string }>({
        queryKey: ["test"],
        columns: [],
        fetchPage: mockFetchPage,
        getRowId: (row) => row.id,
        enableRowSelection: true,
      }),
    );

    act(() => {
      result.current.toggleSelectAll(true);
    });

    expect(result.current.selectedIds).toEqual(new Set(["1", "2"]));
    expect(result.current.allRowsSelected).toBe(true);

    act(() => {
      result.current.toggleSelectAll(false);
    });

    expect(result.current.selectedIds).toEqual(new Set());
  });

  it("clears selection when search changes", () => {
    // First render: mounted, selection set.
    setupHook(
      {
        isFetching: false,
        data: {
          pages: [{ data: [{ id: "1" }], nextCursor: null, hasMore: false }],
        },
        error: null,
        fetchNextPage: vi.fn(),
      },
      { isSuccess: false, isError: false, isPending: false, mutate: vi.fn() },
      { ...mockUrlState, search: "" },
    );

    const { result } = renderHook(() =>
      useDataTable<{ id: string }>({
        queryKey: ["test"],
        columns: [],
        fetchPage: mockFetchPage,
        getRowId: (row) => row.id,
        enableRowSelection: true,
      }),
    );

    act(() => {
      result.current.toggleRowSelection("1", true);
    });
    expect(result.current.selectedIds).toEqual(new Set(["1"]));

    // Second render: search changed -> selection should clear.
    setupHook(
      {
        isFetching: false,
        data: {
          pages: [{ data: [{ id: "1" }], nextCursor: null, hasMore: false }],
        },
        error: null,
        fetchNextPage: vi.fn(),
      },
      { isSuccess: false, isError: false, isPending: false, mutate: vi.fn() },
      { ...mockUrlState, search: "a" },
    );

    const { result: result2 } = renderHook(() =>
      useDataTable<{ id: string }>({
        queryKey: ["test"],
        columns: [],
        fetchPage: mockFetchPage,
        getRowId: (row) => row.id,
        enableRowSelection: true,
      }),
    );

    expect(result2.current.selectedIds).toEqual(new Set());
  });

  it("calls mutate when deleteMutation.mutate is invoked", () => {
    setupHook(
      {
        isFetching: false,
        data: {
          pages: [{ data: [{ id: "1" }], nextCursor: null, hasMore: false }],
        },
        error: null,
        fetchNextPage: vi.fn(),
      },
      {
        isSuccess: false,
        isError: false,
        isPending: false,
        mutate: vi.fn(),
        data: { deletedIds: ["1"], failed: [] },
      },
    );

    const { result } = renderHook(() =>
      useDataTable<{ id: string }>({
        queryKey: ["test"],
        columns: [],
        fetchPage: mockFetchPage,
        getRowId: (row) => row.id,
        enableRowSelection: true,
        onDelete: mockOnDelete,
      }),
    );

    act(() => {
      result.current.deleteMutation.mutate(["1"]);
    });

    // The mutation's mutationFn wraps onDelete; calling mutate triggers it.
    expect(mockOnDelete).toHaveBeenCalledWith(["1"]);
    expect(result.current.isDeleting).toBe(false);
  });

  it("loadMore calls fetchNextPage", () => {
    const fetchNextPage = vi.fn();
    setupHook(
      {
        isFetching: false,
        data: {
          pages: [{ data: [{ id: "1" }], nextCursor: "c1", hasMore: true }],
        },
        error: null,
        fetchNextPage,
      },
      { isSuccess: false, isError: false, isPending: false, mutate: vi.fn() },
    );

    const { result } = renderHook(() =>
      useDataTable<{ id: string }>({
        queryKey: ["test"],
        columns: [],
        fetchPage: mockFetchPage,
        getRowId: (row) => row.id,
      }),
    );

    act(() => {
      result.current.loadMore();
    });

    expect(fetchNextPage).toHaveBeenCalled();
  });

  it("refresh invalidates queries", () => {
    setupHook(
      {
        isFetching: false,
        data: { pages: [] },
        error: null,
        fetchNextPage: vi.fn(),
      },
      { isSuccess: false, isError: false, isPending: false, mutate: vi.fn() },
    );

    const { result } = renderHook(() =>
      useDataTable<{ id: string }>({
        queryKey: ["test"],
        columns: [],
        fetchPage: mockFetchPage,
        getRowId: (row) => row.id,
      }),
    );

    act(() => {
      result.current.refresh();
    });

    expect(mockQueryClient.invalidateQueries).toHaveBeenCalled();
  });

  it("generates column widths with min-width fallback", () => {
    setupHook(
      {
        isFetching: false,
        data: { pages: [] },
        error: null,
        fetchNextPage: vi.fn(),
      },
      { isSuccess: false, isError: false, isPending: false, mutate: vi.fn() },
    );

    const { result } = renderHook(() =>
      useDataTable<{ id: string }>({
        queryKey: ["test"],
        columns: [
          { id: "name", header: "Name", meta: { width: "200px" } },
          { id: "status", header: "Status" },
        ] as any,
        fetchPage: mockFetchPage,
        getRowId: (row) => row.id,
      }),
    );

    expect(result.current.allColumnWidths).toEqual([
      "minmax(100px, 200px)",
      "minmax(100px, 1fr)",
    ]);
  });
});
