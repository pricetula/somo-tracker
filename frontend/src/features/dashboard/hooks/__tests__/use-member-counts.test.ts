import { renderHook } from "@testing-library/react";
import { vi } from "vitest";
import { useMemberCounts } from "@/features/dashboard/hooks/use-member-counts";
import type { MemberCounts } from "@/lib/api/members";

// ---------------------------------------------------------------------------
// Mock @tanstack/react-query so tests run without a QueryClientProvider.
// ---------------------------------------------------------------------------
const mockUseQuery = vi.hoisted(() => vi.fn());

vi.mock("@tanstack/react-query", async (importOriginal) => {
    const actual = await importOriginal<typeof import("@tanstack/react-query")>();
    return {
        ...actual,
        useQuery: mockUseQuery,
    };
});

vi.mock("@/lib/api/members", () => ({
    getMemberCounts: vi.fn(),
}));

import { getMemberCounts } from "@/lib/api/members";

const counts: MemberCounts = {
    students: 120,
    admins: 3,
    nurses: 2,
    teachers: 15,
    parents: 40,
    finance: 1,
};

describe("useMemberCounts", () => {
    beforeEach(() => {
        vi.clearAllMocks();
        mockUseQuery.mockReturnValue({
            data: counts,
            isLoading: false,
            isError: false,
            error: null,
        });
    });

    test("invokes useQuery with the memberCounts query key", () => {
        renderHook(() => useMemberCounts());

        const options = mockUseQuery.mock.calls[0][0];
        expect(options.queryKey).toEqual(["memberCounts"]);
        expect(options.staleTime).toBe(60_000);
    });

    test("queryFn returns response.data from the API", async () => {
        vi.mocked(getMemberCounts).mockResolvedValue({
            code: "success",
            message: "Member counts retrieved",
            data: counts,
        });

        renderHook(() => useMemberCounts());

        const options = mockUseQuery.mock.calls[0][0];
        const result = await options.queryFn();
        expect(result).toEqual(counts);
        expect(getMemberCounts).toHaveBeenCalledTimes(1);
    });

    test("surfaces data, loading and error states from useQuery", () => {
        const { result } = renderHook(() => useMemberCounts());

        expect(result.current.data).toEqual(counts);
        expect(result.current.isLoading).toBe(false);
        expect(result.current.isError).toBe(false);
    });
});
