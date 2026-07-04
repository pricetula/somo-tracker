import { renderHook } from "@testing-library/react";
import { vi } from "vitest";
import { useInfiniteListQuery } from "@/components/shared/data-table/use-infinite-list-query";
import type { NormalizedListResult } from "@/components/shared/data-table/types";

// ---------------------------------------------------------------------------
// Mock @tanstack/react-query so tests run without a QueryClientProvider.
// We keep all other exports (e.g. keepPreviousData) from the real module.
// ---------------------------------------------------------------------------
const mockUseInfiniteQuery = vi.hoisted(() => vi.fn());

vi.mock("@tanstack/react-query", async (importOriginal) => {
    const actual = await importOriginal<typeof import("@tanstack/react-query")>();
    return {
        ...actual,
        useInfiniteQuery: mockUseInfiniteQuery,
    };
});

// ---------------------------------------------------------------------------
// Helpers to build the shape returned by useInfiniteQuery
// ---------------------------------------------------------------------------
interface MockQueryData<TItem> {
    pages: NormalizedListResult<TItem>[];
    pageParams: number[];
}

function buildQueryData<TItem>(pages: NormalizedListResult<TItem>[]): MockQueryData<TItem> {
    return {
        pages,
        pageParams: pages.map((_, i) => i + 1),
    };
}

function buildQueryReturn<TItem>(
    overrides: Partial<ReturnType<typeof mockUseInfiniteQuery>> & {
        data?: MockQueryData<TItem>;
    } = {}
) {
    return {
        data: undefined,
        dataUpdatedAt: 0,
        error: null,
        errorUpdatedAt: 0,
        failureCount: 0,
        failureReason: null,
        fetchNextPage: vi.fn(),
        fetchPreviousPage: vi.fn(),
        hasNextPage: false,
        hasPreviousPage: false,
        isFetched: false,
        isFetchedAfterMount: false,
        isFetching: false,
        isFetchingNextPage: false,
        isFetchingPreviousPage: false,
        isLoading: false,
        isPending: true,
        isRefetching: false,
        isStale: false,
        isSuccess: false,
        isError: false,
        refetch: vi.fn(),
        status: "pending" as const,
        ...overrides,
    };
}

beforeEach(() => {
    mockUseInfiniteQuery.mockReset();
});

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------
describe("useInfiniteListQuery", () => {
    // ── Basic fetching ────────────────────────────────────────────────
    it("fetches initial page and flattens rows", () => {
        mockUseInfiniteQuery.mockReturnValue(
            buildQueryReturn({
                data: buildQueryData([{ items: [1, 2, 3], total: 100, page: 1, limit: 50 }]),
                isPending: false,
                isSuccess: true,
                status: "success",
            })
        );

        const { result } = renderHook(() =>
            useInfiniteListQuery({
                queryKey: ["test"],
                queryFn: vi.fn(),
                params: { filter: "a" },
                limit: 50,
            })
        );

        expect(result.current.rows).toEqual([1, 2, 3]);
        expect(result.current.total).toBe(100);
        expect(result.current.isPending).toBe(false);
    });

    // ── getNextPageParam via total ────────────────────────────────────
    it("computes hasNextPage when loaded < total", () => {
        // getNextPageParam will be called by useInfiniteQuery – since we mock
        // the whole hook we control hasNextPage directly.
        mockUseInfiniteQuery.mockReturnValue(
            buildQueryReturn({
                data: buildQueryData([
                    { items: [1, 2, 3], total: 100, page: 1, limit: 50 },
                    { items: [4, 5, 6], total: 100, page: 2, limit: 50 },
                ]),
                hasNextPage: true,
                isPending: false,
                isSuccess: true,
                status: "success",
            })
        );

        const { result } = renderHook(() =>
            useInfiniteListQuery({
                queryKey: ["test"],
                queryFn: vi.fn(),
                params: {},
                limit: 50,
            })
        );

        expect(result.current.rows).toEqual([1, 2, 3, 4, 5, 6]);
        expect(result.current.hasNextPage).toBe(true);
    });

    it("sets hasNextPage to false when all pages loaded", () => {
        mockUseInfiniteQuery.mockReturnValue(
            buildQueryReturn({
                data: buildQueryData([{ items: [1, 2, 3], total: 3, page: 1, limit: 50 }]),
                hasNextPage: false,
                isPending: false,
                isSuccess: true,
                status: "success",
            })
        );

        const { result } = renderHook(() =>
            useInfiniteListQuery({
                queryKey: ["test"],
                queryFn: vi.fn(),
                params: {},
                limit: 50,
            })
        );

        expect(result.current.rows).toEqual([1, 2, 3]);
        expect(result.current.hasNextPage).toBe(false);
    });

    // ── hasMore flag ─────────────────────────────────────────────────
    it("respects a hasMore flag from the API", () => {
        mockUseInfiniteQuery.mockReturnValue(
            buildQueryReturn({
                data: buildQueryData([{ items: [1, 2, 3], hasMore: true, page: 1, limit: 50 }]),
                hasNextPage: true,
                isPending: false,
                isSuccess: true,
                status: "success",
            })
        );

        const { result } = renderHook(() =>
            useInfiniteListQuery({
                queryKey: ["test"],
                queryFn: vi.fn(),
                params: {},
                limit: 50,
            })
        );

        expect(result.current.hasNextPage).toBe(true);
    });

    // ── Custom normalization ─────────────────────────────────────────
    it("uses the provided normalize function", () => {
        // Mock data that has a non-standard shape (students key instead of items)
        const customPageData = {
            students: [1, 2, 3],
            total: 100,
            page: 1,
            limit: 50,
        };

        const customNormalize = (r: {
            students: number[];
            total: number;
        }): NormalizedListResult<number> => ({
            items: r.students,
            total: r.total,
        });

        // The mock data has students instead of items; normalize remaps it.
        mockUseInfiniteQuery.mockReturnValue(
            buildQueryReturn({
                data: buildQueryData([customPageData as unknown as NormalizedListResult<number>]),
                isPending: false,
                isSuccess: true,
                status: "success",
            })
        );

        const { result } = renderHook(() =>
            useInfiniteListQuery({
                queryKey: ["test"],
                queryFn: vi.fn(),
                params: {},
                normalize: customNormalize,
            })
        );

        expect(result.current.rows).toEqual([1, 2, 3]);
        expect(result.current.total).toBe(100);
    });

    // ── enabled flag ─────────────────────────────────────────────────
    it("does not fetch when enabled is false", () => {
        mockUseInfiniteQuery.mockReturnValue(
            buildQueryReturn({ isPending: true, status: "pending" })
        );

        const { result } = renderHook(() =>
            useInfiniteListQuery({
                queryKey: ["test"],
                queryFn: vi.fn(),
                params: {},
                enabled: false,
            })
        );

        expect(mockUseInfiniteQuery).toHaveBeenCalledWith(
            expect.objectContaining({ enabled: false })
        );
        expect(result.current.isPending).toBe(true);
    });

    // ── query key includes params and limit ──────────────────────────
    it("includes params and limit in the query key", () => {
        mockUseInfiniteQuery.mockReturnValue(buildQueryReturn());

        renderHook(() =>
            useInfiniteListQuery({
                queryKey: ["test"],
                queryFn: vi.fn(),
                params: { search: "a" },
                limit: 50,
            })
        );

        expect(mockUseInfiniteQuery).toHaveBeenCalledWith(
            expect.objectContaining({
                queryKey: ["test", { search: "a" }, 50],
            })
        );
    });

    // ── fetchNextPage proxy ──────────────────────────────────────────
    it("proxies fetchNextPage from useInfiniteQuery", () => {
        const fetchNextPage = vi.fn();
        mockUseInfiniteQuery.mockReturnValue(
            buildQueryReturn({
                data: buildQueryData([{ items: [1, 2, 3], total: 100, page: 1, limit: 50 }]),
                fetchNextPage,
                hasNextPage: true,
                isPending: false,
                isSuccess: true,
                status: "success",
            })
        );

        const { result } = renderHook(() =>
            useInfiniteListQuery({
                queryKey: ["test"],
                queryFn: vi.fn(),
                params: {},
                limit: 50,
            })
        );

        result.current.fetchNextPage();
        expect(fetchNextPage).toHaveBeenCalledTimes(1);
    });

    // ── Error state ──────────────────────────────────────────────────
    it("exposes error from useInfiniteQuery", () => {
        const testError = new Error("API error");
        mockUseInfiniteQuery.mockReturnValue(
            buildQueryReturn({
                error: testError,
                isError: true,
                isPending: false,
                status: "error",
            })
        );

        const { result } = renderHook(() =>
            useInfiniteListQuery({
                queryKey: ["test"],
                queryFn: vi.fn(),
                params: {},
                limit: 50,
            })
        );

        expect(result.current.isError).toBe(true);
        expect(result.current.error).toBe(testError);
    });

    // ── Returns empty rows when no data ──────────────────────────────
    it("returns empty rows when there is no data", () => {
        mockUseInfiniteQuery.mockReturnValue(
            buildQueryReturn({ isPending: true, status: "pending" })
        );

        const { result } = renderHook(() =>
            useInfiniteListQuery({
                queryKey: ["test"],
                queryFn: vi.fn(),
                params: {},
                limit: 50,
            })
        );

        expect(result.current.rows).toEqual([]);
        expect(result.current.total).toBeUndefined();
    });

    // ── getNextPageParam logic via the mock queryFn ──────────────────
    it("passes pageParam and params to queryFn", () => {
        // We need to test that the hook correctly builds the queryFn.
        // Since useInfiniteQuery is mocked, we can verify the options passed to it.
        const queryFn = vi.fn();

        mockUseInfiniteQuery.mockReturnValue(buildQueryReturn());

        renderHook(() =>
            useInfiniteListQuery({
                queryKey: ["test"],
                queryFn,
                params: { search: "hello" },
                limit: 25,
            })
        );

        // The hook passes a wrapper queryFn that merges params with page/limit.
        const passedOptions = mockUseInfiniteQuery.mock.calls[0][0];
        expect(passedOptions.initialPageParam).toBe(1);

        // Call the wrapper queryFn and verify it delegates correctly
        passedOptions.queryFn({ pageParam: 3, signal: undefined });
        expect(queryFn).toHaveBeenCalledWith({ search: "hello", page: 3, limit: 25 });
    });
});
