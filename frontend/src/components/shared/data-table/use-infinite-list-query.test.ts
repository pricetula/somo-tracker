// use-infinite-list-query.test.ts
import { renderHook, act } from "@testing-library/react";
import { vi } from "vitest";
import { useInfiniteListQuery } from "@/components/shared/data-table/use-infinite-list-query";

const mockQueryFn = vi.fn(() => Promise.resolve({}) as const);
const mockQueryFnWithTotal = vi.fn(() =>
    Promise.resolve({ items: [1, 2, 3], total: 100 } as const)
);
const mockQueryFnNoTotal = vi.fn(() => Promise.resolve({ items: [1, 2, 3] } as const));

describe("useInfiniteListQuery", () => {
    beforeEach(() => {
        vi.useFakeTimers();
    });

    afterEach(() => {
        vi.useRealTimers();
    });

    // Basic fetching
    it("fetches initial page with params and limit", () => {
        const { result } = renderHook(() =>
            useInfiniteListQuery({
                queryKey: ["test"],
                queryFn: mockQueryFnWithTotal,
                params: { filter: "a" },
                limit: 50,
            })
        );

        expect(result.current.rows).toEqual([1, 2, 3]);
        expect(result.current.total).toBe(100);
        expect(result.current.isPending).toBe(false);
    });

    // Pagination via total
    it("fetches next page when total > items", () => {
        const { result, findByText } = renderHook(() =>
            useInfiniteListQuery({
                queryKey: ["test"],
                queryFn: mockQueryFnWithTotal,
                params: { filter: "a" },
                limit: 50,
            })
        );

        expect(findByText("3 of 100 loaded")).toBeInTheDocument();

        // Scroll to trigger next page
        // (Simulated via virtualizer scroll API)
        result.current.fetchNextPage();
        act(() => {}); // Let DOM update

        expect(mockQueryFnWithTotal).toHaveBeenCalledWith({
            ...expect.any(Object),
            page: 2,
            limit: 50,
        });
        expect(result.current.rows.length).toBe(6); // 3+3
    });

    // hasMore flag
    it("respects hasMore flag", () => {
        const { result } = renderHook(() =>
            useInfiniteListQuery({
                queryKey: ["test"],
                queryFn: vi.fn(() => Promise.resolve({ items: [1, 2, 3], hasMore: true } as const)),
                limit: 50,
            })
        );

        expect(result.current.hasNextPage).toBe(true);
        result.current.fetchNextPage();
        expect(mockQueryFn).toHaveBeenCalled();
        expect(result.current.hasNextPage).toBe(true);
    });

    // Fallback behavior
    it("handles missing total/hasMore", () => {
        const { result } = renderHook(() =>
            useInfiniteListQuery({
                queryKey: ["test"],
                queryFn: mockQueryFnNoTotal,
                limit: 50,
            })
        );

        expect(result.current.rows).toEqual([1, 2, 3]);
        result.current.fetchNextPage();
        expect(mockQueryFnNoTotal).toHaveBeenCalledWith({ page: 2, limit: 50 });

        // Test last page short
        const shortQueryFn = vi.fn(() => Promise.resolve({ items: [1] } as const));
        const { result: shortResult } = renderHook(() =>
            useInfiniteListQuery({
                queryKey: ["test"],
                queryFn: shortQueryFn,
                limit: 50,
            })
        );
        expect(shortResult.current.requestCount).toBe(1); // Stops after first page

        // Test exact limit
        const exactQueryFn = vi.fn(() => Promise.resolve({ items: [1, 2, 3, 4, 50] } as const));
        const { result: exactResult } = renderHook(() =>
            useInfiniteListQuery({
                queryKey: ["test"],
                queryFn: exactQueryFn,
                limit: 50,
            })
        );
        expect(exactResult.current.rows.length).toBe(50); // Continues
    });

    // Normalization
    it("uses defaultNormalize", () => {
        const { result } = renderHook(() =>
            useInfiniteListQuery({
                queryKey: ["test"],
                queryFn: mockQueryFnWithTotal,
                normalize: (r) =>
                    ({ items: r.items, total: r.total, page: r.page, limit: r.limit }) as const,
            })
        );
        expect(result.current.rows).toEqual([1, 2, 3]);
    });

    it("handles custom normalization", () => {
        const customNormalize = (r) => ({
            items: r.students,
            total: r.total,
            page: r.page,
            limit: r.limit,
        });
        const { result } = renderHook(() =>
            useInfiniteListQuery({
                queryKey: ["test"],
                queryFn: vi.fn(() => Promise.resolve({ students: [1, 2, 3], total: 100 }) as const),
                normalize: customNormalize,
            })
        );
        expect(result.current.rows).toEqual([1, 2, 3]);
    });

    it("handles enabled flag", () => {
        const { result } = renderHook(() =>
            useInfiniteListQuery({
                queryKey: ["test"],
                queryFn: mockQueryFnWithTotal,
                enabled: false,
            })
        );

        expect(mockQueryFnWithTotal).not.toHaveBeenCalled();
        expect(result.current.isPending).toBe(true);
    });

    it("resets params correctly", () => {
        const { result, findByText } = renderHook(() =>
            useInfiniteListQuery({
                queryKey: ["test"],
                queryFn: vi.fn(() => Promise.resolve({ items: [1, 2, 3], total: 100 }) as const),
                params: { search: "a" },
            })
        );

        expect(findByText("3 of 100 loaded")).toBeInTheDocument();

        // Change params
        act(() => {
            result.current.params = { search: "ab" };
        });

        // Pending fetch should keep old data
        expect(findByText("3 of 100 loaded")).toBeInTheDocument();
        // New fetch should trigger
        expect(mockQueryFnWithTotal).toHaveBeenCalledWith({
            ...expect.any(Object),
            page: 1,
            limit: 50,
            search: "ab",
        });
    });

    it("includes params in query key", () => {
        const queryKeySpy = vi.spyOn(useInfiniteListQuery, "queryKey");
        const { result } = renderHook(() =>
            useInfiniteListQuery({
                queryKey: ["test"],
                queryFn: mockQueryFnWithTotal,
                params: { search: "a" },
            })
        );

        expect(queryKeySpy).toHaveBeenCalledWith(["test", { search: "a" }, 50]);
        result.current.params = { search: "b" };
        expect(queryKeySpy).toHaveBeenCalledWith(["test", { search: "b" }, 50]);
    });
});
