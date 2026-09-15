import { render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { vi } from "vitest";
import { ClassesTable } from "../classes-table";

// Mock hooks
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

// Mock DataTable
vi.mock("@/components/shared/data-table", () => ({
    DataTable: ({ filterGroups, isSearchable }: unknown) => (
        <div data-testid="data-table">
            <div data-testid="filter-groups">{JSON.stringify(filterGroups)}</div>
            <div data-testid="is-searchable">{String(isSearchable)}</div>
        </div>
    ),
}));

function createQueryClient() {
    return new QueryClient({
        defaultOptions: { queries: { retry: false } },
    });
}

function renderWithClient(ui: React.ReactElement) {
    const queryClient = createQueryClient();
    return render(<QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>);
}

describe("ClassesTable", () => {
    it("renders with dynamic grade and stream filters", async () => {
        renderWithClient(<ClassesTable />);

        await waitFor(() => {
            expect(screen.getByTestId("data-table")).toBeInTheDocument();
        });

        const filterGroups = JSON.parse(screen.getByTestId("filter-groups").textContent || "[]");
        expect(filterGroups).toHaveLength(2);

        const gradeGroup = filterGroups.find((g: unknown) => g.id === "grade");
        expect(gradeGroup).toBeDefined();
        expect(gradeGroup.items[0].submenu).toHaveLength(2);
        expect(gradeGroup.items[0].submenu[0].label).toBe("Grade 1");
        expect(gradeGroup.items[0].submenu[0].value).toBe("Grade 1");

        const streamGroup = filterGroups.find((g: unknown) => g.id === "stream");
        expect(streamGroup).toBeDefined();
        expect(streamGroup.items[0].submenu).toHaveLength(2);
        expect(streamGroup.items[0].submenu[0].label).toBe("Science");
        expect(streamGroup.items[0].submenu[0].value).toBe("Science");

        expect(screen.getByTestId("is-searchable").textContent).toBe("true");
    });

    it("uses grade-filter and stream-filter item IDs", async () => {
        renderWithClient(<ClassesTable />);

        await waitFor(() => {
            expect(screen.getByTestId("data-table")).toBeInTheDocument();
        });

        const filterGroups = JSON.parse(screen.getByTestId("filter-groups").textContent || "[]");
        const gradeItem = filterGroups[0].items[0];
        const streamItem = filterGroups[1].items[0];

        expect(gradeItem.id).toBe("grade-filter");
        expect(streamItem.id).toBe("stream-filter");
    });
});
