import { act, fireEvent, render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { vi } from "vitest";
import { DataTable } from "../data-table";
import type { DataTableColumn } from "../types";

// ---------------------------------------------------------------------------
// Mocks
// ---------------------------------------------------------------------------

const mockUseInfiniteListQuery = vi.hoisted(() => vi.fn());
vi.mock("../use-infinite-list-query", () => ({
    useInfiniteListQuery: mockUseInfiniteListQuery,
}));

const mockUseVirtualizer = vi.hoisted(() => vi.fn());
vi.mock("@tanstack/react-virtual", () => ({
    useVirtualizer: mockUseVirtualizer,
}));

vi.mock("sonner", () => ({
    toast: { error: vi.fn() },
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

function createQueryClient() {
    return new QueryClient({ defaultOptions: { queries: { retry: false } } });
}

function renderWithClient(ui: React.ReactElement) {
    const queryClient = createQueryClient();
    return render(<QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>);
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
    mockUseVirtualizer.mockReset();
    mockUseVirtualizer.mockReturnValue(createVirtualizerMock(0));
    vi.mocked(vi.fn()); // reset sonner mock
});

afterEach(() => {
    vi.useRealTimers();
});

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("DataTable", () => {
    // ── Initial loading ──────────────────────────────────────────────
    it("shows skeleton rows while pending", () => {
        mockUseInfiniteListQuery.mockReturnValue(setupQuery({ rows: [], isPending: true }));

        const { container } = renderWithClient(
            <DataTable
                queryKey={["test"]}
                queryFn={vi.fn()}
                params={{}}
                columns={defaultColumns}
                getRowId={(row) => row.id}
            />
        );

        // Should render skeleton divs (not "Loading..." text)
        const skeletons = container.querySelectorAll('[data-slot="skeleton"]');
        expect(skeletons.length).toBeGreaterThan(0);
        expect(screen.queryByText("Loading\u2026")).not.toBeInTheDocument();
    });

    // ── Initial error ────────────────────────────────────────────────
    it("shows error message on initial query failure", () => {
        mockUseInfiniteListQuery.mockReturnValue(
            setupQuery({
                rows: [],
                isPending: false,
                isError: true,
                error: new Error("API error"),
            })
        );

        renderWithClient(
            <DataTable
                queryKey={["test"]}
                queryFn={vi.fn()}
                params={{}}
                columns={defaultColumns}
                getRowId={(row) => row.id}
            />
        );

        expect(screen.getByText(/API error/)).toBeInTheDocument();
    });

    // ── Empty state ──────────────────────────────────────────────────
    it("shows emptyState when no data and no search/filters active", () => {
        mockUseInfiniteListQuery.mockReturnValue(
            setupQuery({ rows: [], total: 0, isPending: false, isError: false, isSuccess: true })
        );

        renderWithClient(
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
    });

    it("shows noResultsState when search/filters are active but no data", () => {
        mockUseInfiniteListQuery.mockReturnValue(
            setupQuery({ rows: [], total: 0, isPending: false, isError: false, isSuccess: true })
        );

        renderWithClient(
            <DataTable
                queryKey={["test"]}
                queryFn={vi.fn()}
                params={{}}
                columns={defaultColumns}
                getRowId={(row) => row.id}
                isSearchable
                emptyState={<div>Custom empty</div>}
                noResultsState={<div>Custom no results</div>}
            />
        );

        // Type in search to trigger "no results" mode
        const input = screen.getByPlaceholderText("Search...");
        fireEvent.change(input, { target: { value: "xyz" } });

        // Advance past the debounce delay (300ms) to flip the active search flag
        act(() => {
            vi.advanceTimersByTime(400);
        });

        expect(screen.getByText("Custom no results")).toBeInTheDocument();
    });

    it("shows default 'No data.' when no emptyState prop given", () => {
        mockUseInfiniteListQuery.mockReturnValue(
            setupQuery({ rows: [], total: 0, isPending: false, isError: false, isSuccess: true })
        );

        renderWithClient(
            <DataTable
                queryKey={["test"]}
                queryFn={vi.fn()}
                params={{}}
                columns={defaultColumns}
                getRowId={(row) => row.id}
            />
        );

        expect(screen.getByText("No data.")).toBeInTheDocument();
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

        renderWithClient(
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
        expect(screen.getByText("Item 2")).toBeInTheDocument();
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

        renderWithClient(
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

        renderWithClient(
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
    it("does not render search input when isSearchable not set", () => {
        mockUseInfiniteListQuery.mockReturnValue(
            setupQuery({ rows: [], isPending: false, isError: false, isSuccess: true })
        );

        renderWithClient(
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

    it("renders search input with placeholder when isSearchable", () => {
        mockUseInfiniteListQuery.mockReturnValue(
            setupQuery({ rows: [], isPending: false, isError: false, isSuccess: true })
        );

        renderWithClient(
            <DataTable
                queryKey={["test"]}
                queryFn={vi.fn()}
                params={{}}
                columns={defaultColumns}
                getRowId={(row) => row.id}
                isSearchable
                searchPlaceholder="Find items"
            />
        );

        const input = screen.getByPlaceholderText("Find items");
        expect(input).toBeInTheDocument();
        expect(input).toHaveAttribute("type", "text");
    });

    // ── Toolbar disabled state ───────────────────────────────────────
    it("disables toolbar while initial data is pending", () => {
        mockUseInfiniteListQuery.mockReturnValue(setupQuery({ rows: [], isPending: true }));

        renderWithClient(
            <DataTable
                queryKey={["test"]}
                queryFn={vi.fn()}
                params={{}}
                columns={defaultColumns}
                getRowId={(row) => row.id}
                isSearchable
                addHref="/new"
            />
        );

        // Search input should be disabled
        const searchInput = screen.getByPlaceholderText("Search...");
        expect(searchInput).toBeDisabled();

        // Add button link should have disabled styling (pointer-events-none from Button)
        const addButton = screen.getByRole("link");
        expect(addButton.className).toContain("pointer-events-none");
        expect(addButton.className).toContain("opacity-50");
    });

    it("enables toolbar once data has loaded", () => {
        mockUseInfiniteListQuery.mockReturnValue(
            setupQuery({
                rows: [{ id: 1, name: "Item", value: "Val" }],
                total: 1,
                isPending: false,
                isError: false,
                isSuccess: true,
            })
        );
        mockUseVirtualizer.mockReturnValue(createVirtualizerMock(1));

        renderWithClient(
            <DataTable
                queryKey={["test"]}
                queryFn={vi.fn()}
                params={{}}
                columns={defaultColumns}
                getRowId={(row) => row.id}
                isSearchable
                addHref="/new"
            />
        );

        const searchInput = screen.getByPlaceholderText("Search...");
        expect(searchInput).not.toBeDisabled();
    });

    // ── Checkbox / Selection ─────────────────────────────────────────
    it("renders checkboxes when isCheckable is true", () => {
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

        renderWithClient(
            <DataTable
                queryKey={["test"]}
                queryFn={vi.fn()}
                params={{}}
                columns={defaultColumns}
                getRowId={(row) => row.id}
                isCheckable
            />
        );

        const checkboxes = screen.getAllByRole("checkbox");
        // Header checkbox + 1 row checkbox
        expect(checkboxes).toHaveLength(2);
    });

    it("header checkbox selects all loaded rows", () => {
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

        renderWithClient(
            <DataTable
                queryKey={["test"]}
                queryFn={vi.fn()}
                params={{}}
                columns={defaultColumns}
                getRowId={(row) => row.id}
                isCheckable
            />
        );

        const checkboxes = screen.getAllByRole("checkbox");
        const headerCheckbox = checkboxes[0];

        // Click header checkbox to select all
        fireEvent.click(headerCheckbox);
        const rowCheckboxes = screen.getAllByRole("checkbox").slice(1);
        rowCheckboxes.forEach((cb) => {
            expect(cb).toBeChecked();
        });
    });

    // ── Add button ───────────────────────────────────────────────────
    it("renders add button as a Link when addHref is provided", () => {
        mockUseInfiniteListQuery.mockReturnValue(
            setupQuery({
                rows: [{ id: 1, name: "Item", value: "Val" }],
                total: 1,
                isPending: false,
                isError: false,
                isSuccess: true,
            })
        );
        mockUseVirtualizer.mockReturnValue(createVirtualizerMock(1));

        renderWithClient(
            <DataTable
                queryKey={["test"]}
                queryFn={vi.fn()}
                params={{}}
                columns={defaultColumns}
                getRowId={(row) => row.id}
                addHref="/dashboard/items/new"
            />
        );

        const link = screen.getByRole("link");
        expect(link).toHaveAttribute("href", "/dashboard/items/new");
    });

    it("does not render add button when addHref is not provided", () => {
        mockUseInfiniteListQuery.mockReturnValue(
            setupQuery({
                rows: [{ id: 1, name: "Item", value: "Val" }],
                total: 1,
                isPending: false,
                isError: false,
                isSuccess: true,
            })
        );
        mockUseVirtualizer.mockReturnValue(createVirtualizerMock(1));

        renderWithClient(
            <DataTable
                queryKey={["test"]}
                queryFn={vi.fn()}
                params={{}}
                columns={defaultColumns}
                getRowId={(row) => row.id}
            />
        );

        expect(screen.queryByRole("link")).not.toBeInTheDocument();
    });

    // ── Delete button ────────────────────────────────────────────────
    it("shows delete button with count when items are selected and deleteFn provided", () => {
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

        renderWithClient(
            <DataTable
                queryKey={["test"]}
                queryFn={vi.fn()}
                params={{}}
                columns={defaultColumns}
                getRowId={(row) => row.id}
                isCheckable
                deleteFn={vi.fn()}
            />
        );

        // Select all rows
        const checkboxes = screen.getAllByRole("checkbox");
        fireEvent.click(checkboxes[0]);

        // Delete button should appear with count
        expect(screen.getByText("Delete 2")).toBeInTheDocument();
    });

    it("does not show delete button when no items selected", () => {
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

        renderWithClient(
            <DataTable
                queryKey={["test"]}
                queryFn={vi.fn()}
                params={{}}
                columns={defaultColumns}
                getRowId={(row) => row.id}
                isCheckable
                deleteFn={vi.fn()}
            />
        );

        expect(screen.queryByText(/Delete/)).not.toBeInTheDocument();
    });

    it("does not show delete button when deleteFn not provided", () => {
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

        renderWithClient(
            <DataTable
                queryKey={["test"]}
                queryFn={vi.fn()}
                params={{}}
                columns={defaultColumns}
                getRowId={(row) => row.id}
                isCheckable
            />
        );

        // Select the row to make sure it doesn't show
        const checkboxes = screen.getAllByRole("checkbox");
        fireEvent.click(checkboxes[0]);

        expect(screen.queryByText(/Delete/)).not.toBeInTheDocument();
    });

    // ── Search clears selection ──────────────────────────────────────
    it("clears selectedIds when search input changes", () => {
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

        renderWithClient(
            <DataTable
                queryKey={["test"]}
                queryFn={vi.fn()}
                params={{}}
                columns={defaultColumns}
                getRowId={(row) => row.id}
                isSearchable
                isCheckable
                deleteFn={vi.fn()}
            />
        );

        // Select all rows
        const headerCheckbox = screen.getAllByRole("checkbox")[0];
        fireEvent.click(headerCheckbox);

        // Delete button should appear
        expect(screen.getByText(/Delete/)).toBeInTheDocument();

        // Type in search
        const searchInput = screen.getByPlaceholderText("Search...");
        fireEvent.change(searchInput, { target: { value: "x" } });

        // Delete button should disappear (selection cleared)
        expect(screen.queryByText(/Delete/)).not.toBeInTheDocument();
    });

    // ── Loading more ─────────────────────────────────────────────────
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
        mockUseVirtualizer.mockReturnValue(createVirtualizerMock(51));

        renderWithClient(
            <DataTable
                queryKey={["test"]}
                queryFn={vi.fn()}
                params={{}}
                columns={defaultColumns}
                getRowId={(row) => row.id}
            />
        );

        expect(screen.getByText("Loading more...")).toBeInTheDocument();
    });
});
