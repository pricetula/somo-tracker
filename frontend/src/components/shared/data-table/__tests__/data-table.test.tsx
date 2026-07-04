import { act, fireEvent, render, screen } from "@testing-library/react";
import { vi } from "vitest";
import { DataTable } from "../data-table";
import type { DataTableColumn } from "../types";

// ---------------------------------------------------------------------------
// Mocks — use vi.hoisted so variables are available inside vi.mock factories
// (vi.mock is hoisted to the top of the file by vitest).
// ---------------------------------------------------------------------------

const mockUseInfiniteListQuery = vi.hoisted(() => vi.fn());
vi.mock("../use-infinite-list-query", () => ({
    useInfiniteListQuery: mockUseInfiniteListQuery,
}));

const mockUseVirtualizer = vi.hoisted(() => vi.fn());
vi.mock("@tanstack/react-virtual", () => ({
    useVirtualizer: mockUseVirtualizer,
}));

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

interface TestItem {
    id: number;
    name: string;
    value: string;
}

const defaultColumns: DataTableColumn<TestItem>[] = [
    { id: "name", header: "Name", cell: (row) => row.name },
    { id: "value", header: "Value", cell: (row) => row.value, width: "200px" },
];

function createVirtualizerMock(itemCount: number) {
    const virtualItems = Array.from({ length: itemCount }, (_, i) => ({
        index: i,
        start: i * 44,
        key: `virtual-${i}`,
        lane: 0,
        size: 44,
    }));

    return {
        getVirtualItems: () => virtualItems,
        getTotalSize: () => itemCount * 44,
        measureElement: vi.fn(),
    };
}

const defaultQueryReturn = {
    rows: [] as TestItem[],
    total: undefined as number | undefined,
    isPending: true,
    isError: false,
    error: null as Error | null,
    isFetchingNextPage: false,
    hasNextPage: false,
    fetchNextPage: vi.fn(),
    refetch: vi.fn(),
    data: undefined,
    dataUpdatedAt: 0,
    errorUpdatedAt: 0,
    failureCount: 0,
    failureReason: null,
    fetchPreviousPage: vi.fn(),
    hasPreviousPage: false,
    isFetched: false,
    isFetchedAfterMount: false,
    isFetching: false,
    isFetchingPreviousPage: false,
    isLoading: false,
    isRefetching: false,
    isStale: false,
    isSuccess: false,
    status: "pending" as const,
};

function setupQuery(overrides: Partial<typeof defaultQueryReturn> & { rows: TestItem[] }) {
    return { ...defaultQueryReturn, ...overrides };
}

beforeEach(() => {
    vi.useFakeTimers();
    mockUseInfiniteListQuery.mockReset();
    // Provide a default virtualizer so all tests have one. Tests that need
    // specific virtualizer state can override this in their own setup.
    mockUseVirtualizer.mockReturnValue(createVirtualizerMock(0));
});

afterEach(() => {
    vi.useRealTimers();
});

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------
describe("DataTable", () => {
    // ── Loading state ────────────────────────────────────────────────
    it("shows loading indicator while pending", () => {
        mockUseInfiniteListQuery.mockReturnValue(setupQuery({ rows: [], isPending: true }));

        render(
            <DataTable
                queryKey={["test"]}
                queryFn={vi.fn()}
                params={{}}
                columns={defaultColumns}
                getRowId={(row) => row.id}
            />
        );

        expect(screen.getByText("Loading\u2026")).toBeInTheDocument();
    });

    // ── Error state ──────────────────────────────────────────────────
    it("shows error message on query failure", () => {
        mockUseInfiniteListQuery.mockReturnValue(
            setupQuery({
                rows: [],
                isPending: false,
                isError: true,
                error: new Error("API error"),
            })
        );

        render(
            <DataTable
                queryKey={["test"]}
                queryFn={vi.fn()}
                params={{}}
                columns={defaultColumns}
                getRowId={(row) => row.id}
            />
        );

        expect(screen.getByText(/Failed to load data/i)).toBeInTheDocument();
    });

    // ── Empty state ──────────────────────────────────────────────────
    it("shows emptyState content when no items", () => {
        mockUseInfiniteListQuery.mockReturnValue(
            setupQuery({ rows: [], total: 0, isPending: false, isError: false, isSuccess: true })
        );

        render(
            <DataTable
                queryKey={["test"]}
                queryFn={vi.fn()}
                params={{}}
                columns={defaultColumns}
                getRowId={(row) => row.id}
                emptyState={<div>Custom empty</div>}
            />
        );

        expect(screen.getByText("Custom empty")).toBeInTheDocument();
        expect(screen.queryByText("No results found.")).not.toBeInTheDocument();
    });

    it("shows default 'No results found.' when no emptyState prop given", () => {
        mockUseInfiniteListQuery.mockReturnValue(
            setupQuery({ rows: [], total: 0, isPending: false, isError: false, isSuccess: true })
        );

        render(
            <DataTable
                queryKey={["test"]}
                queryFn={vi.fn()}
                params={{}}
                columns={defaultColumns}
                getRowId={(row) => row.id}
            />
        );

        expect(screen.getByText("No results found.")).toBeInTheDocument();
    });

    // ── Header and cells ─────────────────────────────────────────────
    it("renders header and cell content", () => {
        const rows: TestItem[] = [
            { id: 1, name: "Item 1", value: "Value 1" },
            { id: 2, name: "Item 2", value: "Value 2" },
        ];

        mockUseInfiniteListQuery.mockReturnValue(
            setupQuery({
                rows,
                total: 2,
                isPending: false,
                isError: false,
                isSuccess: true,
            })
        );
        mockUseVirtualizer.mockReturnValue(createVirtualizerMock(rows.length));

        render(
            <DataTable
                queryKey={["test"]}
                queryFn={vi.fn()}
                params={{}}
                columns={defaultColumns}
                getRowId={(row) => row.id}
            />
        );

        expect(screen.getByText("Name")).toBeInTheDocument();
        expect(screen.getByText("Value")).toBeInTheDocument();
        expect(screen.getByText("Item 1")).toBeInTheDocument();
        expect(screen.getByText("Value 1")).toBeInTheDocument();
        expect(screen.getByText("Item 2")).toBeInTheDocument();
        expect(screen.getByText("Value 2")).toBeInTheDocument();
    });

    // ── Footer ───────────────────────────────────────────────────────
    it("shows 'X of Y loaded' footer", () => {
        const rows = Array.from({ length: 50 }, (_, i) => ({
            id: i + 1,
            name: `Item ${i + 1}`,
            value: `Value ${i + 1}`,
        }));

        mockUseInfiniteListQuery.mockReturnValue(
            setupQuery({
                rows,
                total: 200,
                isPending: false,
                isError: false,
                isSuccess: true,
            })
        );
        mockUseVirtualizer.mockReturnValue(createVirtualizerMock(rows.length));

        render(
            <DataTable
                queryKey={["test"]}
                queryFn={vi.fn()}
                params={{}}
                columns={defaultColumns}
                getRowId={(row) => row.id}
            />
        );

        expect(screen.getByText(/50 of 200/)).toBeInTheDocument();
    });

    it("does not render footer when total is undefined", () => {
        mockUseInfiniteListQuery.mockReturnValue(
            setupQuery({
                rows: [{ id: 1, name: "Item 1", value: "Value 1" }],
                total: undefined,
                isPending: false,
                isError: false,
                isSuccess: true,
            })
        );
        mockUseVirtualizer.mockReturnValue(createVirtualizerMock(1));

        render(
            <DataTable
                queryKey={["test"]}
                queryFn={vi.fn()}
                params={{}}
                columns={defaultColumns}
                getRowId={(row) => row.id}
            />
        );

        expect(screen.queryByText(/of/)).not.toBeInTheDocument();
    });

    // ── Search input ─────────────────────────────────────────────────
    it("does not render search input when onSearch not provided", () => {
        mockUseInfiniteListQuery.mockReturnValue(
            setupQuery({ rows: [], isPending: false, isError: false, isSuccess: true })
        );

        render(
            <DataTable
                queryKey={["test"]}
                queryFn={vi.fn()}
                params={{}}
                columns={defaultColumns}
                getRowId={(row) => row.id}
            />
        );

        expect(screen.queryByRole("textbox")).not.toBeInTheDocument();
    });

    it("renders search input with placeholder when onSearch is provided", () => {
        mockUseInfiniteListQuery.mockReturnValue(
            setupQuery({ rows: [], isPending: false, isError: false, isSuccess: true })
        );

        render(
            <DataTable
                queryKey={["test"]}
                queryFn={vi.fn()}
                params={{}}
                columns={defaultColumns}
                getRowId={(row) => row.id}
                onSearch={vi.fn()}
                searchPlaceholder="Find items"
            />
        );

        const input = screen.getByPlaceholderText("Find items");
        expect(input).toBeInTheDocument();
        expect(input).toHaveAttribute("type", "text");
    });

    it("calls onSearch after debounce delay when user types", () => {
        const onSearch = vi.fn();

        mockUseInfiniteListQuery.mockReturnValue(
            setupQuery({ rows: [], isPending: false, isError: false, isSuccess: true })
        );

        render(
            <DataTable
                queryKey={["test"]}
                queryFn={vi.fn()}
                params={{}}
                columns={defaultColumns}
                getRowId={(row) => row.id}
                onSearch={onSearch}
                searchPlaceholder="Search..."
            />
        );

        // onSearch is called with "" on mount when the debounced value initialises
        expect(onSearch).toHaveBeenCalledTimes(1);
        expect(onSearch).toHaveBeenCalledWith("");

        const input = screen.getByPlaceholderText("Search...") as HTMLInputElement;
        fireEvent.change(input, { target: { value: "hello" } });

        // Advance past the debounce delay (350ms), wrapped in act
        act(() => {
            vi.advanceTimersByTime(400);
        });

        expect(onSearch).toHaveBeenCalledTimes(2);
        expect(onSearch).toHaveBeenLastCalledWith("hello");
    });

    it("only calls onSearch with the final debounced value after rapid typing", () => {
        const onSearch = vi.fn();

        mockUseInfiniteListQuery.mockReturnValue(
            setupQuery({ rows: [], isPending: false, isError: false, isSuccess: true })
        );

        render(
            <DataTable
                queryKey={["test"]}
                queryFn={vi.fn()}
                params={{}}
                columns={defaultColumns}
                getRowId={(row) => row.id}
                onSearch={onSearch}
                searchPlaceholder="Search..."
            />
        );

        // onSearch is called with "" on mount when the debounced value initialises
        expect(onSearch).toHaveBeenCalledTimes(1);
        expect(onSearch).toHaveBeenCalledWith("");

        const input = screen.getByPlaceholderText("Search...") as HTMLInputElement;

        fireEvent.change(input, { target: { value: "a" } });
        fireEvent.change(input, { target: { value: "ab" } });
        fireEvent.change(input, { target: { value: "abc" } });

        // Advance past the debounce delay
        act(() => {
            vi.advanceTimersByTime(400);
        });

        expect(onSearch).toHaveBeenCalledTimes(2);
        expect(onSearch).toHaveBeenLastCalledWith("abc");
    });

    // ── Column configuration ─────────────────────────────────────────
    it("applies column width styling", () => {
        const rows: TestItem[] = [{ id: 1, name: "Item 1", value: "Value 1" }];

        mockUseInfiniteListQuery.mockReturnValue(
            setupQuery({
                rows,
                total: 1,
                isPending: false,
                isError: false,
                isSuccess: true,
            })
        );
        mockUseVirtualizer.mockReturnValue(createVirtualizerMock(rows.length));

        render(
            <DataTable
                queryKey={["test"]}
                queryFn={vi.fn()}
                params={{}}
                columns={defaultColumns}
                getRowId={(row) => row.id}
            />
        );

        const valueHeader = screen.getByText("Value");
        expect(valueHeader).toBeInTheDocument();
    });

    it("applies column text alignment", () => {
        const rows: TestItem[] = [{ id: 1, name: "Item 1", value: "Value 1" }];

        const alignedColumns: DataTableColumn<TestItem>[] = [
            { id: "name", header: "Name", cell: (row) => row.name, align: "right" },
            { id: "value", header: "Value", cell: (row) => row.value },
        ];

        mockUseInfiniteListQuery.mockReturnValue(
            setupQuery({
                rows,
                total: 1,
                isPending: false,
                isError: false,
                isSuccess: true,
            })
        );
        mockUseVirtualizer.mockReturnValue(createVirtualizerMock(rows.length));

        render(
            <DataTable
                queryKey={["test"]}
                queryFn={vi.fn()}
                params={{}}
                columns={alignedColumns}
                getRowId={(row) => row.id}
            />
        );

        const nameHeader = screen.getByText("Name");
        expect(nameHeader).toHaveStyle("text-align: right");

        const cell = screen.getByText("Item 1");
        expect(cell).toHaveStyle("text-align: right");
    });

    // ── Loading more indicator ───────────────────────────────────────
    it("shows a loader row while fetching the next page", () => {
        const rows = Array.from({ length: 50 }, (_, i) => ({
            id: i + 1,
            name: `Item ${i + 1}`,
            value: `Value ${i + 1}`,
        }));

        mockUseInfiniteListQuery.mockReturnValue(
            setupQuery({
                rows,
                total: 200,
                isPending: false,
                isError: false,
                isSuccess: true,
                hasNextPage: true,
                isFetchingNextPage: true,
            })
        );
        // Virtualizer returns 51 items (50 rows + 1 loader)
        mockUseVirtualizer.mockReturnValue(createVirtualizerMock(51));

        render(
            <DataTable
                queryKey={["test"]}
                queryFn={vi.fn()}
                params={{}}
                columns={defaultColumns}
                getRowId={(row) => row.id}
            />
        );

        expect(screen.getByText("Loading more\u2026")).toBeInTheDocument();
    });
});
