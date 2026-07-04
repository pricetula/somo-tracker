// data-table.test.tsx
import { act, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { rest } from "msw";
import { setupServer } from "msw/node";
import { DataTable } from "./data-table";
import type { DataTableColumn } from "./types";
import { useDebouncedValue } from "./use-debounced-value";
import { vi } from "vitest";

// Mock useDebouncedValue to control debounce in tests
vi.mock("./use-debounced-value");
const useDebouncedValueMock = useDebouncedValue as unknown as ReturnType<typeof vi.fn>;

// Test item type
interface TestItem {
    id: number;
    name: string;
    value: string;
}

// Mock API server
const server = setupServer(
    rest.get("/api/test-items", (req, res, ctx) => {
        const { page = 1, limit = 50 } = req.url.searchParams;
        const items: TestItem[] = [];
        const start = (Number(page) - 1) * Number(limit);
        for (let i = start; i < start + Number(limit) && i < 200; i++) {
            items.push({ id: i + 1, name: `Item ${i + 1}`, value: `Value ${i + 1}` });
        }
        return res(
            ctx.status(200),
            ctx.json({
                items,
                total: 200,
                page: Number(page),
                limit: Number(limit),
            })
        );
    })
);

beforeAll(() => server.listen());
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

describe("DataTable", () => {
    const columns: DataTableColumn<TestItem>[] = [
        { id: "name", header: "Name", cell: (row) => row.name },
        { id: "value", header: "Value", cell: (row) => row.value, width: "200px" },
    ];

    const queryFn = async ({
        params,
        page,
        limit,
        signal,
    }: {
        params: Record<string, string | number>;
        page: number;
        limit: number;
        signal: AbortSignal;
    }) => {
        const response = await fetch(
            `/api/test-items?page=${page}&limit=${limit}&${new URLSearchParams(params as Record<string, string>)}`,
            { signal }
        );
        if (!response.ok) throw new Error("Network error");
        return response.json();
    };

    const renderTable = (props: {
        height?: number;
        rowHeight?: number;
        emptyState?: React.ReactNode;
        onSearch?: (term: string) => void;
        searchPlaceholder?: string;
        params?: Record<string, string | number>;
        columns?: DataTableColumn<TestItem>[];
    }) => {
        render(
            <DataTable
                queryKey={["test"]}
                queryFn={
                    queryFn as (
                        p: { page: number; limit: number } & Record<string, string | number>
                    ) => Promise<unknown>
                }
                params={props.params ?? {}}
                columns={props.columns ?? columns}
                height={props.height}
                rowHeight={props.rowHeight}
                emptyState={props.emptyState}
                onSearch={props.onSearch}
                searchPlaceholder={props.searchPlaceholder}
            />
        );
    };

    // Rendering states
    it("shows loading indicator initially", async () => {
        renderTable();
        expect(screen.getByText(/loading/i)).toBeInTheDocument();
        await waitFor(() => expect(screen.getByText(/loaded/i)).not.toBeInTheDocument());
    });

    it("shows error message on query failure", async () => {
        server.use(rest.get("/api/test-items", (req, res, ctx) => res(ctx.status(500))));

        renderTable();
        await waitFor(() => expect(screen.getByText(/failed/i)).toBeInTheDocument());
    });

    it("shows emptyState when no items", async () => {
        server.use(
            rest.get("/api/test-items", (req, res, ctx) => res(ctx.json({ items: [], total: 0 })))
        );

        renderTable({ emptyState: <div>Custom empty</div> });
        await waitFor(() => {
            expect(screen.getByText(/custom empty/i)).toBeInTheDocument();
            expect(screen.queryByText(/no results/i)).not.toBeInTheDocument();
        });
    });

    it("renders header and cells correctly", async () => {
        renderTable();
        await waitFor(() => {
            expect(screen.getByText(/name/i)).toBeInTheDocument();
            expect(screen.getByText(/value/i)).toBeInTheDocument();
            expect(screen.getByText(/item 1/i)).toBeInTheDocument();
            expect(screen.getByText(/value 1/i)).toBeInTheDocument();
        });
    });

    it("shows X of Y loaded footer", async () => {
        renderTable();
        await waitFor(() => {
            const footerText = screen.getByText(/\d+ of \d+/);
            expect(footerText).toHaveTextContent(/50 of 200/);
        });
    });

    // Search functionality
    it("does not render search input when onSearch not provided", () => {
        renderTable();
        expect(screen.queryByRole("textbox")).not.toBeInTheDocument();
    });

    it("renders search input with placeholder when onSearch provided", () => {
        const onSearch = vi.fn();
        renderTable({ onSearch, searchPlaceholder: "Find items" });
        expect(screen.getByPlaceholderText(/find items/i)).toBeInTheDocument();
    });

    it("debounces search input", async () => {
        const onSearch = vi.fn();
        useDebouncedValueMock.mockImplementation((value: string) => {
            return value;
        });
        renderTable({ onSearch });
        const input = screen.getByRole("textbox");
        await userEvent.type(input, "hello");
        expect(onSearch).not.toHaveBeenCalled();
        vi.advanceTimersByTime(350);
        expect(onSearch).toHaveBeenCalledWith("hello");
    });

    it("ignores intermediate search values", async () => {
        const onSearch = vi.fn();
        useDebouncedValueMock.mockImplementation((value: string) => value);
        renderTable({ onSearch });
        const input = screen.getByRole("textbox");
        await userEvent.type(input, "a");
        await userEvent.type(input, "ab");
        await userEvent.type(input, "abc");
        vi.advanceTimersByTime(350);
        expect(onSearch).toHaveBeenCalledTimes(1);
        expect(onSearch).toHaveBeenLastCalledWith("abc");
    });

    it("does not call stale onSearch reference", async () => {
        const onSearchV1 = vi.fn();
        const onSearchV2 = vi.fn();
        useDebouncedValueMock.mockImplementation((value: string) => value);
        renderTable({ onSearch: onSearchV1 });
        const input = screen.getByRole("textbox");
        await userEvent.type(input, "test");
        vi.advanceTimersByTime(350);
        expect(onSearchV2).toHaveBeenCalledWith("test");
    });

    // Virtualized scrolling
    it("renders subset of rows initially", async () => {
        renderTable({ height: 200, rowHeight: 44 });
        await waitFor(() => {
            const rows = screen.getAllByRole("row", { hidden: false });
            expect(rows.length).toBeLessThan(50);
            expect(rows.length).toBeGreaterThan(5);
        });
    });

    it("stops fetching when all items loaded", async () => {
        renderTable({ height: 200, rowHeight: 44 });
        await waitFor(() => {
            expect(screen.getByText(/item 1/i)).toBeInTheDocument();
        });
        const scrollContainer = screen.getByRole("region");
        userEvent.dispatchEvent(scrollContainer, new Event("scroll"));
        userEvent.dispatchEvent(scrollContainer, new Event("scroll"));
        userEvent.dispatchEvent(scrollContainer, new Event("scroll"));
        await waitFor(() => {
            expect(screen.getAllByText(/item \d+/i)).toHaveLength(200);
        });
    });

    // Request cancellation
    it("cancels previous request on param change", async () => {
        const fetchSpy = vi.spyOn(global, "fetch").mockImplementation(() =>
            Promise.resolve(
                new Response(JSON.stringify({ items: [], total: 0 }), {
                    status: 200,
                    headers: { "content-type": "application/json" },
                })
            )
        );

        renderTable({ params: { search: "a" } });
        act(() => {});
        expect(fetchSpy).toHaveBeenCalledWith(
            expect.any(String),
            expect.objectContaining({ signal: expect.any(AbortSignal) })
        );
        expect(fetchSpy).toHaveBeenCalledTimes(2);
    });

    // Column configuration
    it("applies column width", async () => {
        renderTable();
        await waitFor(() => {
            const header = screen.getByText(/value/i);
            expect(header).toHaveStyle("width: 200px");
        });
    });

    it("applies column text alignment", async () => {
        const alignedColumns = [...columns];
        alignedColumns[0].align = "right";
        renderTable({ columns: alignedColumns });
        await waitFor(() => {
            const cell = screen.getByText(/item 1/i);
            expect(cell).toHaveStyle("text-align: right");
        });
    });
});
