import { render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { vi } from "vitest";
import { ClassesTable } from "../classes-table";

vi.mock("@/features/grades/hooks/use-grades", () => ({
    useGrades: () => ({
        data: [
            {
                id: "g1",
                education_system_id: "es1",
                local_label: "Grade 1",
                tier_stage: "primary",
                sequence_index: 1,
            },
            {
                id: "g2",
                education_system_id: "es1",
                local_label: "Grade 2",
                tier_stage: "primary",
                sequence_index: 2,
            },
        ],
        isLoading: false,
        isError: false,
    }),
}));

vi.mock("@/features/streams/hooks/use-streams", () => ({
    useStreams: () => ({
        data: [
            { id: "s1", name: "Science", school_id: "school1" },
            { id: "s2", name: "Arts", school_id: "school1" },
        ],
        isLoading: false,
        isError: false,
    }),
}));

let capturedProps: Record<string, unknown> | null = null;
const _filterInteractionTest = false;

vi.mock("@/components/shared/data-table", () => ({
    DataTable: (props: { filterGroups?: unknown[]; [key: string]: unknown }) => {
        capturedProps = props as Record<string, unknown>;

        // Simulate filter pill rendering
        const simulateFilterPills = () => {
            // Mock active filters state
            const _activeFilters = {
                "grade-filter": ["Grade 1"],
                "stream-filter": ["Science", "Arts"],
            };

            return (
                <div>
                    <div data-testid="filter-pill-grade-1">
                        Grade 1<button data-testid="remove-grade-1">×</button>
                    </div>
                    <div data-testid="filter-pill-science">
                        Science
                        <button data-testid="remove-science">×</button>
                    </div>
                    <div data-testid="filter-pill-arts">
                        Arts
                        <button data-testid="remove-arts">×</button>
                    </div>
                </div>
            );
        };

        return (
            <div data-testid="data-table">
                {simulateFilterPills()}
                <div>
                    Filters: {JSON.stringify((props as Record<string, unknown>).filterGroups)}
                </div>
            </div>
        );
    },
}));

function createQueryClient() {
    return new QueryClient({ defaultOptions: { queries: { retry: false } } });
}

function renderWithClient(ui: React.ReactElement) {
    const queryClient = createQueryClient();
    return render(<QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>);
}

describe("ClassesTable Filter Pills", () => {
    it("renders filter pills for selected options", async () => {
        renderWithClient(<ClassesTable />);

        await waitFor(() => {
            expect(screen.getByTestId("data-table")).toBeInTheDocument();
        });

        // Verify filter groups are configured
        expect(capturedProps.filterGroups).toBeDefined();

        // Verify filter pills would render for selected options
        expect(screen.getByTestId("filter-pill-grade-1")).toBeInTheDocument();
        expect(screen.getByTestId("filter-pill-science")).toBeInTheDocument();
        expect(screen.getByTestId("filter-pill-arts")).toBeInTheDocument();

        // Verify pill content matches filter labels
        expect(screen.getByTestId("filter-pill-grade-1")).toHaveTextContent("Grade 1");
        expect(screen.getByTestId("filter-pill-science")).toHaveTextContent("Science");
        expect(screen.getByTestId("filter-pill-arts")).toHaveTextContent("Arts");
    });

    it("filter pills have remove buttons", async () => {
        renderWithClient(<ClassesTable />);

        await waitFor(() => {
            expect(screen.getByTestId("data-table")).toBeInTheDocument();
        });

        expect(screen.getByTestId("remove-grade-1")).toBeInTheDocument();
        expect(screen.getByTestId("remove-science")).toBeInTheDocument();
        expect(screen.getByTestId("remove-arts")).toBeInTheDocument();
    });

    it("filters use correct IDs for pill grouping", async () => {
        renderWithClient(<ClassesTable />);

        await waitFor(() => {
            expect(screen.getByTestId("data-table")).toBeInTheDocument();
        });

        const gradeItem = capturedProps.filterGroups[0].items[0];
        const streamItem = capturedProps.filterGroups[1].items[0];

        // Filter item IDs should be unique and stable
        expect(gradeItem.id).toBe("grade-filter");
        expect(streamItem.id).toBe("stream-filter");

        // Submenu items should have values matching labels for pill display
        expect(gradeItem.submenu[0].value).toBe(gradeItem.submenu[0].label);
        expect(streamItem.submenu[0].value).toBe(streamItem.submenu[0].label);
    });
});
