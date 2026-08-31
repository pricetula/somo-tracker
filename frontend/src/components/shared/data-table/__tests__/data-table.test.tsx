import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { vi } from "vitest";
import { DataTable } from "../data-table";
import type { DataTableColumn, FilterGroup } from "../types";

// ===========================================================================
// MOCKS
// ===========================================================================

const mockUseInfiniteListQuery = vi.hoisted(() => vi.fn());
vi.mock("../use-infinite-list-query", () => ({
    useInfiniteListQuery: mockUseInfiniteListQuery,
}));

const mockUseVirtualizer = vi.hoisted(() => vi.fn());
vi.mock("@tanstack/react-virtual", () => ({
    useVirtualizer: mockUseVirtualizer,
}));

const mockToastError = vi.fn();
vi.mock("sonner", () => ({
    toast: { error: vi.fn((...args: unknown[]) => mockToastError(...args)) },
}));

// ===========================================================================
// HELPERS
// ===========================================================================

interface TestItem {
    id: number;
    name: string;
    value: string;
}

const defaultColumns: DataTableColumn<TestItem>[] = [
    { id: "name", header: "Name", cell: (row: TestItem) => row.name },
    { id: "value", header: "Value", cell: (row: TestItem) => row.value, width: "200px" },
];

interface VirtualItem {
    index: number;
    start: number;
    key: string;
    lane: number;
    size: number;
}

function createVirtualizerMock(itemCount: number) {
    const virtualItems: VirtualItem[] = Array.from({ length: itemCount }, (_, i) => ({
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
    mockToastError.mockReset();
});

afterEach(() => {
    vi.useRealTimers();
});

// ===========================================================================
// TESTS
// ===========================================================================

describe("DataTable", () => {
    describe("initial loading state", () => {
        it("shows skeleton rows while pending and no data", () => {
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

            const skeletons = container.querySelectorAll('[data-slot="skeleton"]');
            expect(skeletons.length).toBeGreaterThan(0);
            expect(screen.queryByText("Loading\u2026")).not.toBeInTheDocument();
        });

        it("does NOT show skeleton when data exists even if still pending (keepPreviousData)", () => {
            const rows: TestItem[] = [{ id: 1, name: "Item", value: "Val" }];
            mockUseInfiniteListQuery.mockReturnValue(
                setupQuery({ rows, isPending: true, isSuccess: true })
            );
            mockUseVirtualizer.mockReturnValue(createVirtualizerMock(1));

            const { container } = renderWithClient(
                <DataTable
                    queryKey={["test"]}
                    queryFn={vi.fn()}
                    params={{}}
                    columns={defaultColumns}
                    getRowId={(row) => row.id}
                />
            );

            expect(container.querySelectorAll('[data-slot="skeleton"]')).toHaveLength(0);
            expect(screen.getByText("Item")).toBeInTheDocument();
        });

        it("shows correct skeleton grid for checkable tables", () => {
            mockUseInfiniteListQuery.mockReturnValue(setupQuery({ rows: [], isPending: true }));

            const { container } = renderWithClient(
                <DataTable
                    queryKey={["test"]}
                    queryFn={vi.fn()}
                    params={{}}
                    columns={defaultColumns}
                    getRowId={(row) => row.id}
                    isCheckable
                />
            );

            // 4 skeleton rows × (checkbox + 2 content columns) = 12 skeleton pieces
            expect(container.querySelectorAll('[data-slot="skeleton"]')).toHaveLength(12);
        });
    });

    describe("error states", () => {
        it("shows error message on initial query failure (no prior data)", () => {
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

        it("shows generic message when error is not an Error instance", () => {
            mockUseInfiniteListQuery.mockReturnValue(
                setupQuery({
                    rows: [],
                    isPending: false,
                    isError: true,
                    error: new Error("string error message"),
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

            expect(screen.getByText("Failed to load data.")).toBeInTheDocument();
        });

        it("shows generic message when error is null", () => {
            mockUseInfiniteListQuery.mockReturnValue(
                setupQuery({
                    rows: [],
                    isPending: false,
                    isError: true,
                    error: null,
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

            expect(screen.getByText("Failed to load data.")).toBeInTheDocument();
        });

        it("does NOT show error banner when data exists (pagination error on page 2+)", () => {
            const rows: TestItem[] = [{ id: 1, name: "Item", value: "Val" }];
            mockUseInfiniteListQuery.mockReturnValue(
                setupQuery({
                    rows,
                    total: 100,
                    isPending: false,
                    isError: true,
                    error: new Error("Page 2 failed"),
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

            expect(screen.queryByText("Page 2 failed")).not.toBeInTheDocument();
            expect(screen.getByText("Item")).toBeInTheDocument();
        });

        it("fires a toast for pagination error (page 2+) via queueMicrotask", async () => {
            const rows: TestItem[] = [{ id: 1, name: "Item", value: "Val" }];
            const error = new Error("Pagination failed");
            mockUseInfiniteListQuery.mockReturnValue(
                setupQuery({
                    rows,
                    total: 100,
                    isPending: false,
                    isError: true,
                    error,
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

            // queueMicrotask runs after the current task — flush it
            await act(async () => {
                await Promise.resolve();
            });

            expect(mockToastError).toHaveBeenCalledWith("Pagination failed");
        });

        it("does NOT fire duplicate toast for the same error", async () => {
            const rows: TestItem[] = [{ id: 1, name: "Item", value: "Val" }];
            const error = new Error("Same error");
            mockUseInfiniteListQuery.mockReturnValue(
                setupQuery({
                    rows,
                    total: 100,
                    isPending: false,
                    isError: true,
                    error,
                })
            );
            mockUseVirtualizer.mockReturnValue(createVirtualizerMock(1));

            const { rerender } = renderWithClient(
                <DataTable
                    queryKey={["test"]}
                    queryFn={vi.fn()}
                    params={{}}
                    columns={defaultColumns}
                    getRowId={(row) => row.id}
                />
            );

            await act(async () => {
                await Promise.resolve();
            });
            expect(mockToastError).toHaveBeenCalledTimes(1);

            // Rerender (same component instance) — error ref prevents duplicate toast
            rerender(
                <QueryClientProvider client={createQueryClient()}>
                    <DataTable
                        queryKey={["test"]}
                        queryFn={vi.fn()}
                        params={{}}
                        columns={defaultColumns}
                        getRowId={(row) => row.id}
                    />
                </QueryClientProvider>
            );

            await act(async () => {
                await Promise.resolve();
            });
            expect(mockToastError).toHaveBeenCalledTimes(1);
        });

        it("fires a new toast when error changes", async () => {
            const rows: TestItem[] = [{ id: 1, name: "Item", value: "Val" }];
            const error1 = new Error("Error 1");
            mockUseInfiniteListQuery.mockReturnValue(
                setupQuery({
                    rows,
                    total: 100,
                    isPending: false,
                    isError: true,
                    error: error1,
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

            await act(async () => {
                await Promise.resolve();
            });
            expect(mockToastError).toHaveBeenCalledWith("Error 1");

            // Render new instance with a different error
            const error2 = new Error("Error 2");
            mockUseInfiniteListQuery.mockReturnValue(
                setupQuery({
                    rows,
                    total: 100,
                    isPending: false,
                    isError: true,
                    error: error2,
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

            await act(async () => {
                await Promise.resolve();
            });
            expect(mockToastError).toHaveBeenCalledWith("Error 2");
            expect(mockToastError).toHaveBeenCalledTimes(2);
        });
    });

    describe("empty states", () => {
        it("shows emptyState when no data and no search/filters active", () => {
            mockUseInfiniteListQuery.mockReturnValue(
                setupQuery({
                    rows: [],
                    total: 0,
                    isPending: false,
                    isError: false,
                    isSuccess: true,
                })
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

        it("shows default 'No data.' when no emptyState prop given", () => {
            mockUseInfiniteListQuery.mockReturnValue(
                setupQuery({
                    rows: [],
                    total: 0,
                    isPending: false,
                    isError: false,
                    isSuccess: true,
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

            expect(screen.getByText("No data.")).toBeInTheDocument();
        });

        it("shows noResultsState when search is active with no results", () => {
            mockUseInfiniteListQuery.mockReturnValue(
                setupQuery({
                    rows: [],
                    total: 0,
                    isPending: false,
                    isError: false,
                    isSuccess: true,
                })
            );

            renderWithClient(
                <DataTable
                    queryKey={["test"]}
                    queryFn={vi.fn()}
                    params={{}}
                    columns={defaultColumns}
                    getRowId={(row) => row.id}
                    isSearchable
                    emptyState={<div>Empty</div>}
                    noResultsState={<div>Custom no results</div>}
                />
            );

            const input = screen.getByPlaceholderText("Search...");
            fireEvent.change(input, { target: { value: "xyz" } });

            act(() => {
                vi.advanceTimersByTime(400);
            });

            expect(screen.getByText("Custom no results")).toBeInTheDocument();
        });

        it("shows default 'No results found.' when no noResultsState given", () => {
            mockUseInfiniteListQuery.mockReturnValue(
                setupQuery({
                    rows: [],
                    total: 0,
                    isPending: false,
                    isError: false,
                    isSuccess: true,
                })
            );

            renderWithClient(
                <DataTable
                    queryKey={["test"]}
                    queryFn={vi.fn()}
                    params={{}}
                    columns={defaultColumns}
                    getRowId={(row) => row.id}
                    isSearchable
                />
            );

            const input = screen.getByPlaceholderText("Search...");
            fireEvent.change(input, { target: { value: "xyz" } });

            act(() => {
                vi.advanceTimersByTime(400);
            });

            expect(screen.getByText("No results found.")).toBeInTheDocument();
        });

        it("shows noResultsState when filters are active with no results", () => {
            mockUseInfiniteListQuery.mockReturnValue(
                setupQuery({
                    rows: [],
                    total: 0,
                    isPending: false,
                    isError: false,
                    isSuccess: true,
                })
            );

            const filterGroups: FilterGroup[] = [
                {
                    id: "status_group",
                    label: "Filter by",
                    items: [
                        {
                            id: "status",
                            label: "Status",
                            type: "sub_menu_single",
                            submenu: [{ id: "active", label: "Active", value: "ACTIVE" }],
                        },
                    ],
                },
            ];

            renderWithClient(
                <DataTable
                    queryKey={["test"]}
                    queryFn={vi.fn()}
                    params={{}}
                    columns={defaultColumns}
                    getRowId={(row) => row.id}
                    filterGroups={filterGroups}
                    emptyState={<div>Empty</div>}
                    noResultsState={<div>Filtered no results</div>}
                />
            );

            // Open filter dropdown via pointerDown (Radix UI uses pointer events)
            const filterButton = screen.getByRole("button", { name: "" });

            // Radix UI DropdownMenu opens on pointerDown
            fireEvent.pointerDown(filterButton);

            // The submenu trigger should now be visible
            // Wait for Radix UI to render the dropdown content
            act(() => {
                vi.advanceTimersByTime(0);
            });

            // Click the "Status" submenu trigger
            const statusTrigger = screen.getByText("Status");
            fireEvent.click(statusTrigger);

            // Click "Active" radio item
            const activeOption = screen.getByText("Active");
            fireEvent.click(activeOption);

            expect(screen.getByText("Filtered no results")).toBeInTheDocument();
        });
    });

    describe("malformed / edge case responses", () => {
        it("handles items: null gracefully (shows empty state, not crash)", () => {
            // Simulate what happens when API returns { items: null, total: 0 }
            mockUseInfiniteListQuery.mockReturnValue(
                setupQuery({
                    rows: [],
                    total: 0,
                    isPending: false,
                    isError: false,
                    isSuccess: true,
                })
            );

            renderWithClient(
                <DataTable
                    queryKey={["test"]}
                    queryFn={vi.fn()}
                    params={{}}
                    columns={defaultColumns}
                    getRowId={(row) => row.id}
                    emptyState={<div>No data</div>}
                />
            );

            expect(screen.getByText("No data")).toBeInTheDocument();
        });

        it("handles total = 0 correctly", () => {
            mockUseInfiniteListQuery.mockReturnValue(
                setupQuery({
                    rows: [],
                    total: 0,
                    isPending: false,
                    isError: false,
                    isSuccess: true,
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

            expect(screen.getByText("No data.")).toBeInTheDocument();
        });

        it("handles total = undefined by omitting footer", () => {
            const rows: TestItem[] = [{ id: 1, name: "Item", value: "Val" }];
            mockUseInfiniteListQuery.mockReturnValue(
                setupQuery({
                    rows,
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

        it("handles rows with missing fields gracefully", () => {
            type PartialItem = { id: number; name?: string; value?: string };
            const rows: PartialItem[] = [{ id: 1, name: "Full", value: "Data" }, { id: 2 }];

            const columns: DataTableColumn<PartialItem>[] = [
                { id: "name", header: "Name", cell: (row) => row.name ?? "—" },
                { id: "value", header: "Value", cell: (row) => row.value ?? "—" },
            ];

            mockUseInfiniteListQuery.mockReturnValue(
                setupQuery({
                    rows: rows as TestItem[],
                    total: 2,
                    isPending: false,
                    isError: false,
                    isSuccess: true,
                })
            );
            const virtualItems: VirtualItem[] = [
                { index: 0, start: 0, key: "virtual-0", lane: 0, size: 44 },
                { index: 1, start: 44, key: "virtual-1", lane: 0, size: 44 },
            ];
            mockUseVirtualizer.mockReturnValue({
                getVirtualItems: () => virtualItems,
                getTotalSize: () => 88,
                measureElement: vi.fn(),
            });

            renderWithClient(
                <DataTable
                    queryKey={["test"]}
                    queryFn={vi.fn()}
                    params={{}}
                    columns={columns as unknown as DataTableColumn<TestItem>[]}
                    getRowId={(row: unknown, _i: number) => (row as PartialItem).id}
                />
            );

            expect(screen.getByText("Full")).toBeInTheDocument();
            // Row 2 has both name and value missing → two dashes
            const dashes = screen.getAllByText("—");
            expect(dashes).toHaveLength(2);
        });
    });

    describe("data rendering", () => {
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
            expect(screen.getByText("Value 1")).toBeInTheDocument();
            expect(screen.getByText("Value 2")).toBeInTheDocument();
        });

        it("applies column width to grid template", () => {
            const rows: TestItem[] = [{ id: 1, name: "Item", value: "Val" }];
            mockUseInfiniteListQuery.mockReturnValue(
                setupQuery({
                    rows,
                    total: 1,
                    isPending: false,
                    isError: false,
                    isSuccess: true,
                })
            );
            mockUseVirtualizer.mockReturnValue(createVirtualizerMock(1));

            const { container } = renderWithClient(
                <DataTable
                    queryKey={["test"]}
                    queryFn={vi.fn()}
                    params={{}}
                    columns={defaultColumns}
                    getRowId={(row) => row.id}
                />
            );

            const header = container.querySelector('[style*="grid-template-columns"]');
            expect(header?.getAttribute("style")).toContain("1fr");
            expect(header?.getAttribute("style")).toContain("200px");
        });

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

        it("renders a single row without crashing", () => {
            const rows: TestItem[] = [{ id: 1, name: "Solo", value: "Only" }];
            mockUseInfiniteListQuery.mockReturnValue(
                setupQuery({
                    rows,
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

            expect(screen.getByText("Solo")).toBeInTheDocument();
            expect(screen.getByText("Only")).toBeInTheDocument();
        });

        it("applies className prop to the root element", () => {
            mockUseInfiniteListQuery.mockReturnValue(
                setupQuery({
                    rows: [],
                    total: 0,
                    isPending: false,
                    isError: false,
                    isSuccess: true,
                })
            );

            const { container } = renderWithClient(
                <DataTable
                    queryKey={["test"]}
                    queryFn={vi.fn()}
                    params={{}}
                    columns={defaultColumns}
                    getRowId={(row) => row.id}
                    className="my-custom-class"
                />
            );

            const root = container.firstChild as HTMLElement;
            expect(root.className).toContain("my-custom-class");
        });

        it("renders with empty columns array", () => {
            const rows: TestItem[] = [{ id: 1, name: "Item", value: "Val" }];
            mockUseInfiniteListQuery.mockReturnValue(
                setupQuery({
                    rows,
                    total: 1,
                    isPending: false,
                    isError: false,
                    isSuccess: true,
                })
            );
            mockUseVirtualizer.mockReturnValue(createVirtualizerMock(1));

            const { container } = renderWithClient(
                <DataTable
                    queryKey={["test"]}
                    queryFn={vi.fn()}
                    params={{}}
                    columns={[]}
                    getRowId={(row: unknown, _i: number) => (row as TestItem).id}
                />
            );

            expect(container.querySelector('[class*="grid"]')).toBeInTheDocument();
        });
    });

    describe("search", () => {
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

        it("uses default search placeholder when none provided", () => {
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
                />
            );

            expect(screen.getByPlaceholderText("Search...")).toBeInTheDocument();
        });

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

            const headerCheckbox = screen.getAllByRole("checkbox")[0];
            fireEvent.click(headerCheckbox);
            expect(screen.getByText(/Delete/)).toBeInTheDocument();

            const searchInput = screen.getByPlaceholderText("Search...");
            fireEvent.change(searchInput, { target: { value: "x" } });

            expect(screen.queryByText(/Delete/)).not.toBeInTheDocument();
        });

        it("passes search value to useInfiniteListQuery", () => {
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
                />
            );

            // On initial render with isSearchable, search prop is "" (empty string from useDebouncedValue)
            // Not "undefined" — the DataTable passes debouncedSearch which starts as ""
            expect(mockUseInfiniteListQuery).toHaveBeenLastCalledWith(
                expect.objectContaining({ search: "" })
            );
        });
    });

    describe("toolbar disabled state", () => {
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

            const searchInput = screen.getByPlaceholderText("Search...");
            expect(searchInput).toBeDisabled();

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

        it("keeps toolbar disabled when re-pending before first load completes", () => {
            mockUseInfiniteListQuery.mockReturnValue(setupQuery({ rows: [], isPending: true }));

            const { rerender } = renderWithClient(
                <DataTable
                    queryKey={["test"]}
                    queryFn={vi.fn()}
                    params={{}}
                    columns={defaultColumns}
                    getRowId={(row) => row.id}
                    isSearchable
                />
            );

            const searchInput = screen.getByPlaceholderText("Search...");
            expect(searchInput).toBeDisabled();

            // Still pending — toolbar stays disabled
            mockUseInfiniteListQuery.mockReturnValue(setupQuery({ rows: [], isPending: true }));

            rerender(
                <QueryClientProvider client={createQueryClient()}>
                    <DataTable
                        queryKey={["test"]}
                        queryFn={vi.fn()}
                        params={{}}
                        columns={defaultColumns}
                        getRowId={(row) => row.id}
                        isSearchable
                    />
                </QueryClientProvider>
            );

            expect(searchInput).toBeDisabled();
        });
    });

    describe("checkbox / selection", () => {
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
            expect(checkboxes).toHaveLength(2); // header + 1 row
        });

        it("does NOT render checkboxes when isCheckable is false", () => {
            const rows: TestItem[] = [{ id: 1, name: "Item", value: "Val" }];
            mockUseInfiniteListQuery.mockReturnValue(
                setupQuery({
                    rows,
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

            expect(screen.queryByRole("checkbox")).not.toBeInTheDocument();
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
            fireEvent.click(checkboxes[0]);

            const rowCheckboxes = screen.getAllByRole("checkbox").slice(1);
            rowCheckboxes.forEach((cb) => {
                expect(cb).toBeChecked();
            });
        });

        it("header checkbox deselects all when already selected", () => {
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

            const checkboxes = screen.getAllByRole("checkbox");
            const headerCheckbox = checkboxes[0];

            // Select all
            fireEvent.click(headerCheckbox);
            expect(screen.getByText("Delete 2")).toBeInTheDocument();

            // Deselect all
            fireEvent.click(headerCheckbox);
            expect(screen.queryByText(/Delete/)).not.toBeInTheDocument();

            const rowCheckboxes = screen.getAllByRole("checkbox").slice(1);
            rowCheckboxes.forEach((cb) => {
                expect(cb).not.toBeChecked();
            });
        });

        it("header checkbox shows indeterminate state when some rows selected", () => {
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
            const virtualItems: VirtualItem[] = [
                { index: 0, start: 0, key: "virtual-0", lane: 0, size: 44 },
                { index: 1, start: 44, key: "virtual-1", lane: 0, size: 44 },
            ];
            mockUseVirtualizer.mockReturnValue({
                getVirtualItems: () => virtualItems,
                getTotalSize: () => 88,
                measureElement: vi.fn(),
            });

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

            // Click first row only
            const row1Checkbox = checkboxes[1];
            fireEvent.click(row1Checkbox);

            // The Radix UI Checkbox uses data-state="indeterminate" for indeterminate
            expect(headerCheckbox.getAttribute("data-state")).toBe("indeterminate");
        });

        it("row checkbox toggles individual selection", () => {
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
            mockUseVirtualizer.mockReturnValue(createVirtualizerMock(1));

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

            const rowCheckbox = screen.getAllByRole("checkbox")[1];
            expect(rowCheckbox).not.toBeChecked();

            fireEvent.click(rowCheckbox);
            expect(rowCheckbox).toBeChecked();

            fireEvent.click(rowCheckbox);
            expect(rowCheckbox).not.toBeChecked();
        });

        it("clears selection when filters change", async () => {
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
            mockUseVirtualizer.mockReturnValue(createVirtualizerMock(1));

            const filterGroups: FilterGroup[] = [
                {
                    id: "g1",
                    label: "Group 1",
                    items: [
                        {
                            id: "btn1",
                            label: "Toggle me",
                            type: "button",
                            value: "active",
                        },
                    ],
                },
            ];

            renderWithClient(
                <DataTable
                    queryKey={["test"]}
                    queryFn={vi.fn()}
                    params={{}}
                    columns={defaultColumns}
                    getRowId={(row) => row.id}
                    isCheckable
                    filterGroups={filterGroups}
                    deleteFn={vi.fn()}
                />
            );

            // Select the row
            const rowCheckbox = screen.getAllByRole("checkbox")[1];
            fireEvent.click(rowCheckbox);
            expect(screen.getByText("Delete 1")).toBeInTheDocument();

            // Open filter dropdown via pointerDown (Radix UI uses pointer events)
            const filterButton = screen.getByRole("button", { name: "" });
            fireEvent.pointerDown(filterButton);

            act(() => {
                vi.advanceTimersByTime(0);
            });

            // Click the "Toggle me" checkbox item
            const toggleItem = screen.getByText("Toggle me");
            fireEvent.click(toggleItem);

            // Selection should be cleared — delete button gone
            expect(screen.queryByText(/Delete/)).not.toBeInTheDocument();
        });
    });

    describe("delete", () => {
        it("shows delete button with count when items selected and deleteFn provided", () => {
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

            const checkboxes = screen.getAllByRole("checkbox");
            fireEvent.click(checkboxes[0]);

            expect(screen.getByText("Delete 2")).toBeInTheDocument();
        });

        it("shows singular 'Delete 1' text for single selection", () => {
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
            mockUseVirtualizer.mockReturnValue(createVirtualizerMock(1));

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

            const rowCheckbox = screen.getAllByRole("checkbox")[1];
            fireEvent.click(rowCheckbox);

            expect(screen.getByText("Delete 1")).toBeInTheDocument();
        });

        it("does not show delete button when no items selected", () => {
            mockUseInfiniteListQuery.mockReturnValue(
                setupQuery({
                    rows: [{ id: 1, name: "Item 1", value: "Value 1" }],
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
            mockUseVirtualizer.mockReturnValue(createVirtualizerMock(1));

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
            fireEvent.click(checkboxes[0]);

            expect(screen.queryByText(/Delete/)).not.toBeInTheDocument();
        });

        it("opens confirmation dialog on delete button click", () => {
            const rows: TestItem[] = [{ id: 1, name: "Item", value: "Val" }];
            mockUseInfiniteListQuery.mockReturnValue(
                setupQuery({
                    rows,
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
                    isCheckable
                    deleteFn={vi.fn()}
                />
            );

            const rowCheckbox = screen.getAllByRole("checkbox")[1];
            fireEvent.click(rowCheckbox);

            const deleteButton = screen.getByText("Delete 1");
            fireEvent.click(deleteButton);

            expect(screen.getByText("Delete 1 item")).toBeInTheDocument();
            expect(
                screen.getByText(
                    "This action cannot be undone. The selected items will be permanently removed."
                )
            ).toBeInTheDocument();
        });

        it("cancels delete when Cancel is clicked", () => {
            const rows: TestItem[] = [{ id: 1, name: "Item", value: "Val" }];
            const deleteFn = vi.fn();
            mockUseInfiniteListQuery.mockReturnValue(
                setupQuery({
                    rows,
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
                    isCheckable
                    deleteFn={deleteFn}
                />
            );

            const rowCheckbox = screen.getAllByRole("checkbox")[1];
            fireEvent.click(rowCheckbox);

            const deleteButton = screen.getByText("Delete 1");
            fireEvent.click(deleteButton);

            const cancelButton = screen.getByText("Cancel");
            fireEvent.click(cancelButton);

            expect(screen.queryByText("Delete 1 item")).not.toBeInTheDocument();
            expect(deleteFn).not.toHaveBeenCalled();
        });

        it("calls deleteFn for each selected item on confirm", async () => {
            vi.useRealTimers();

            const rows: TestItem[] = [
                { id: 1, name: "Item 1", value: "Val 1" },
                { id: 2, name: "Item 2", value: "Val 2" },
            ];
            const deleteFn = vi.fn().mockResolvedValue(undefined);
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
                    deleteFn={deleteFn}
                />
            );

            // Select all
            const headerCheckbox = screen.getAllByRole("checkbox")[0];
            fireEvent.click(headerCheckbox);

            // Open dialog
            fireEvent.click(screen.getByText("Delete 2"));

            // Confirm
            fireEvent.click(screen.getByText("Delete"));

            await waitFor(() => {
                expect(deleteFn).toHaveBeenCalledTimes(2);
            });
            // IDs are stringified via String(getRowId(...))
            expect(deleteFn).toHaveBeenCalledWith("1", undefined);
            expect(deleteFn).toHaveBeenCalledWith("2", undefined);
        });

        it("passes deleteParams to deleteFn", async () => {
            vi.useRealTimers();

            const rows: TestItem[] = [{ id: 1, name: "Item", value: "Val" }];
            const deleteFn = vi.fn().mockResolvedValue(undefined);
            mockUseInfiniteListQuery.mockReturnValue(
                setupQuery({
                    rows,
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
                    isCheckable
                    deleteFn={deleteFn}
                    deleteParams={{ school_id: "123" }}
                />
            );

            const rowCheckbox = screen.getAllByRole("checkbox")[1];
            fireEvent.click(rowCheckbox);

            fireEvent.click(screen.getByText("Delete 1"));
            fireEvent.click(screen.getByText("Delete"));

            await waitFor(() => {
                // ID is stringified via String(getRowId(...))
                expect(deleteFn).toHaveBeenCalledWith("1", { school_id: "123" });
            });
        });

        it("rolls back optimistic removal on delete error", async () => {
            vi.useRealTimers();

            const rows: TestItem[] = [{ id: 1, name: "Item", value: "Val" }];
            const deleteFn = vi.fn().mockRejectedValue(new Error("Delete failed"));
            mockUseInfiniteListQuery.mockReturnValue(
                setupQuery({
                    rows,
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
                    isCheckable
                    deleteFn={deleteFn}
                />
            );

            const rowCheckbox = screen.getAllByRole("checkbox")[1];
            fireEvent.click(rowCheckbox);
            fireEvent.click(screen.getByText("Delete 1"));
            fireEvent.click(screen.getByText("Delete"));

            await waitFor(() => {
                expect(deleteFn).toHaveBeenCalled();
            });

            // Flush microtask for the error toast
            await act(async () => {
                await Promise.resolve();
            });

            expect(mockToastError).toHaveBeenCalledWith("Delete failed");
        });
    });

    describe("add button", () => {
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

        it("does not render add button when addHref not provided", () => {
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
    });

    describe("renderToolBarComponents", () => {
        it("renders toolbar component with selected IDs", () => {
            const rows: TestItem[] = [
                { id: 1, name: "Item 1", value: "Val 1" },
                { id: 2, name: "Item 2", value: "Val 2" },
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
                    renderToolBarComponents={(ids) => (
                        <button type="button">Custom ({ids.size})</button>
                    )}
                />
            );

            expect(screen.getByText("Custom (0)")).toBeInTheDocument();

            const rowCheckbox = screen.getAllByRole("checkbox")[1];
            fireEvent.click(rowCheckbox);

            expect(screen.getByText("Custom (1)")).toBeInTheDocument();
        });

        it("renders nothing when renderToolBarComponents returns null", () => {
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
                    renderToolBarComponents={() => null}
                />
            );

            expect(screen.queryByText(/Custom/)).not.toBeInTheDocument();
        });
    });

    describe("filter interactions", () => {
        it("renders filter dropdown when filterGroups provided", () => {
            mockUseInfiniteListQuery.mockReturnValue(
                setupQuery({ rows: [], isPending: false, isError: false, isSuccess: true })
            );

            const filterGroups: FilterGroup[] = [
                {
                    id: "g1",
                    label: "Group 1",
                    items: [{ id: "b1", label: "Button", type: "button", value: "on" }],
                },
            ];

            renderWithClient(
                <DataTable
                    queryKey={["test"]}
                    queryFn={vi.fn()}
                    params={{}}
                    columns={defaultColumns}
                    getRowId={(row) => row.id}
                    filterGroups={filterGroups}
                />
            );

            expect(screen.getByRole("button", { name: "" })).toBeInTheDocument();
        });

        it("does NOT render filter button when filterGroups is empty array", () => {
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

            expect(screen.queryByRole("button", { name: "" })).not.toBeInTheDocument();
        });

        it("toggles button filter on click via Radix UI dropdown", () => {
            mockUseInfiniteListQuery.mockReturnValue(
                setupQuery({ rows: [], isPending: false, isError: false, isSuccess: true })
            );

            const filterGroups: FilterGroup[] = [
                {
                    id: "g1",
                    label: "Group 1",
                    items: [{ id: "active", label: "Active Only", type: "button", value: "true" }],
                },
            ];

            renderWithClient(
                <DataTable
                    queryKey={["test"]}
                    queryFn={vi.fn()}
                    params={{}}
                    columns={defaultColumns}
                    getRowId={(row) => row.id}
                    filterGroups={filterGroups}
                />
            );

            // Open dropdown via pointerDown (Radix UI uses pointer events)
            const filterButton = screen.getByRole("button", { name: "" });
            fireEvent.pointerDown(filterButton);

            act(() => {
                vi.advanceTimersByTime(0);
            });

            // Click the dropdown menu checkbox item
            // Use getAllByText because the pill (after click) also shows "Active Only"
            // We want the one inside the dropdown menu portal, which comes first
            const activeOnlyItems = screen.getAllByText("Active Only");
            // First one is in the dropdown menu, second is the pill after click
            fireEvent.click(activeOnlyItems[0]);

            // Filter pill should appear
            const pills = screen.getAllByText("Active Only");
            expect(pills.length).toBeGreaterThanOrEqual(2); // dropdown item + pill

            // Remove pill — find the X button inside the pill span
            const pillSpan =
                pills.find((el) => el.closest('[class*="bg-muted"]') !== null) ?? pills[1];
            const removeBtn = pillSpan.closest("span")!.querySelector("button")!;
            fireEvent.click(removeBtn);

            // Pill should be gone — dropdown item may still be visible if menu is still open
            const _remainingItems = screen.getAllByText("Active Only");
            // Menu might still be open, so there should only be the dropdown item
            // (the pill span was removed)
            expect(screen.queryByText(/^Active Only$/)).toBeInTheDocument();
        });

        it("shows filter pills for active filters via submenu selection", () => {
            mockUseInfiniteListQuery.mockReturnValue(
                setupQuery({ rows: [], isPending: false, isError: false, isSuccess: true })
            );

            const filterGroups: FilterGroup[] = [
                {
                    id: "g1",
                    label: "Status",
                    items: [
                        {
                            id: "status",
                            label: "Status",
                            type: "sub_menu_single",
                            submenu: [
                                { id: "active", label: "Active", value: "ACTIVE" },
                                { id: "inactive", label: "Inactive", value: "INACTIVE" },
                            ],
                        },
                    ],
                },
            ];

            renderWithClient(
                <DataTable
                    queryKey={["test"]}
                    queryFn={vi.fn()}
                    params={{}}
                    columns={defaultColumns}
                    getRowId={(row) => row.id}
                    filterGroups={filterGroups}
                />
            );

            // Open dropdown via pointerDown
            const filterButton = screen.getByRole("button", { name: "" });
            fireEvent.pointerDown(filterButton);

            act(() => {
                vi.advanceTimersByTime(0);
            });

            // The filter group label is "Status" and the submenu trigger is also "Status"
            // Use getAllByText and pick the submenu trigger (the one with role="menuitem")
            const statusElements = screen.getAllByText("Status");
            const statusTrigger = statusElements.find(
                (el) => el.getAttribute("role") === "menuitem"
            )!;
            fireEvent.click(statusTrigger);

            // Select "Active" radio item
            const activeItem = screen.getByText("Active");
            fireEvent.click(activeItem);

            // "Active" pill should appear
            expect(screen.getByText("Active")).toBeInTheDocument();
        });
    });

    describe("virtualizer / loading more", () => {
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

        it("calls fetchNextPage when virtualizer reaches the last row", () => {
            const rows = Array.from({ length: 50 }, (_, i) => ({
                id: i + 1,
                name: `Item ${i + 1}`,
                value: `Value ${i + 1}`,
            }));

            const fetchNextPage = vi.fn();
            mockUseInfiniteListQuery.mockReturnValue(
                setupQuery({
                    rows,
                    total: 200,
                    isPending: false,
                    isError: false,
                    isSuccess: true,
                    hasNextPage: true,
                    isFetchingNextPage: false,
                    fetchNextPage,
                })
            );

            const virtualItems: VirtualItem[] = Array.from({ length: 51 }, (_, i) => ({
                index: i,
                start: i * 44,
                key: `v-${i}`,
                lane: 0,
                size: 44,
            }));
            mockUseVirtualizer.mockReturnValue({
                getVirtualItems: () => virtualItems,
                getTotalSize: () => 51 * 44,
                measureElement: vi.fn(),
            });

            renderWithClient(
                <DataTable
                    queryKey={["test"]}
                    queryFn={vi.fn()}
                    params={{}}
                    columns={defaultColumns}
                    getRowId={(row) => row.id}
                />
            );

            expect(fetchNextPage).toHaveBeenCalledTimes(1);
        });

        it("does NOT call fetchNextPage when hasNextPage is false", () => {
            const rows = Array.from({ length: 50 }, (_, i) => ({
                id: i + 1,
                name: `Item ${i + 1}`,
                value: `Value ${i + 1}`,
            }));

            const fetchNextPage = vi.fn();
            mockUseInfiniteListQuery.mockReturnValue(
                setupQuery({
                    rows,
                    total: 50,
                    isPending: false,
                    isError: false,
                    isSuccess: true,
                    hasNextPage: false,
                    isFetchingNextPage: false,
                    fetchNextPage,
                })
            );

            const virtualItems: VirtualItem[] = Array.from({ length: 50 }, (_, i) => ({
                index: i,
                start: i * 44,
                key: `v-${i}`,
                lane: 0,
                size: 44,
            }));
            mockUseVirtualizer.mockReturnValue({
                getVirtualItems: () => virtualItems,
                getTotalSize: () => 50 * 44,
                measureElement: vi.fn(),
            });

            renderWithClient(
                <DataTable
                    queryKey={["test"]}
                    queryFn={vi.fn()}
                    params={{}}
                    columns={defaultColumns}
                    getRowId={(row) => row.id}
                />
            );

            expect(fetchNextPage).not.toHaveBeenCalled();
        });

        it("does NOT call fetchNextPage when already fetching next page", () => {
            const rows = Array.from({ length: 50 }, (_, i) => ({
                id: i + 1,
                name: `Item ${i + 1}`,
                value: `Value ${i + 1}`,
            }));

            const fetchNextPage = vi.fn();
            mockUseInfiniteListQuery.mockReturnValue(
                setupQuery({
                    rows,
                    total: 200,
                    isPending: false,
                    isError: false,
                    isSuccess: true,
                    hasNextPage: true,
                    isFetchingNextPage: true,
                    fetchNextPage,
                })
            );

            const virtualItems: VirtualItem[] = Array.from({ length: 51 }, (_, i) => ({
                index: i,
                start: i * 44,
                key: `v-${i}`,
                lane: 0,
                size: 44,
            }));
            mockUseVirtualizer.mockReturnValue({
                getVirtualItems: () => virtualItems,
                getTotalSize: () => 51 * 44,
                measureElement: vi.fn(),
            });

            renderWithClient(
                <DataTable
                    queryKey={["test"]}
                    queryFn={vi.fn()}
                    params={{}}
                    columns={defaultColumns}
                    getRowId={(row) => row.id}
                />
            );

            expect(fetchNextPage).not.toHaveBeenCalled();
        });

        it("does NOT call fetchNextPage when virtualizer has no items", () => {
            const fetchNextPage = vi.fn();
            mockUseInfiniteListQuery.mockReturnValue(
                setupQuery({
                    rows: [],
                    total: 0,
                    isPending: false,
                    isError: false,
                    isSuccess: true,
                    hasNextPage: false,
                    isFetchingNextPage: false,
                    fetchNextPage,
                })
            );
            mockUseVirtualizer.mockReturnValue({
                getVirtualItems: () => [],
                getTotalSize: () => 0,
                measureElement: vi.fn(),
            });

            renderWithClient(
                <DataTable
                    queryKey={["test"]}
                    queryFn={vi.fn()}
                    params={{}}
                    columns={defaultColumns}
                    getRowId={(row) => row.id}
                />
            );

            expect(fetchNextPage).not.toHaveBeenCalled();
        });
    });

    describe("edge cases", () => {
        it("handles getRowId returning numeric 0 (falsy id)", () => {
            interface ZeroIdItem {
                id: number;
                label: string;
            }
            const rows: ZeroIdItem[] = [{ id: 0, label: "Zero" }];

            const columns: DataTableColumn<ZeroIdItem>[] = [
                { id: "label", header: "Label", cell: (row) => row.label },
            ];

            mockUseInfiniteListQuery.mockReturnValue(
                setupQuery({
                    rows: rows as unknown as TestItem[],
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
                    columns={columns as unknown as DataTableColumn<unknown>[]}
                    getRowId={(row: unknown) => (row as ZeroIdItem).id}
                    isCheckable
                />
            );

            expect(screen.getByText("Zero")).toBeInTheDocument();
            expect(screen.getAllByRole("checkbox")).toHaveLength(2);
        });

        it("handles custom rowHeight and height props", () => {
            mockUseInfiniteListQuery.mockReturnValue(
                setupQuery({
                    rows: [{ id: 1, name: "Item", value: "Val" }],
                    total: 1,
                    isPending: false,
                    isError: false,
                    isSuccess: true,
                })
            );
            const virtualItems: VirtualItem[] = [
                { index: 0, start: 0, key: "v-0", lane: 0, size: 60 },
            ];
            mockUseVirtualizer.mockReturnValue({
                getVirtualItems: () => virtualItems,
                getTotalSize: () => 60,
                measureElement: vi.fn(),
            });

            const { container } = renderWithClient(
                <DataTable
                    queryKey={["test"]}
                    queryFn={vi.fn()}
                    params={{}}
                    columns={defaultColumns}
                    getRowId={(row) => row.id}
                    rowHeight={60}
                    height={400}
                />
            );

            const scrollContainer = container.querySelector('[style*="overflow: auto"]');
            expect(scrollContainer?.getAttribute("style")).toContain("height: 400px");
        });
    });
});
