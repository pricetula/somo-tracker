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

vi.mock("@/components/shared/data-table", () => ({
    DataTable: (props: { filterGroups?: unknown[]; [key: string]: unknown }) => {
        capturedProps = props as Record<string, unknown>;
        return <div data-testid="data-table">Mock DataTable</div>;
    },
}));

function createQueryClient() {
    return new QueryClient({ defaultOptions: { queries: { retry: false } } });
}

function renderWithClient(ui: React.ReactElement) {
    const queryClient = createQueryClient();
    return render(<QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>);
}

describe("ClassesTable Filters", () => {
    beforeEach(() => {
        capturedProps = null;
    });

    it("configures grade and stream filters with labels", async () => {
        renderWithClient(<ClassesTable />);
        await waitFor(() => expect(screen.getByTestId("data-table")).toBeInTheDocument());

        expect(capturedProps.filterGroups).toHaveLength(2);

        const gradeItem = capturedProps.filterGroups[0].items[0];
        expect(gradeItem.id).toBe("grade-filter");
        expect(gradeItem.submenu[0].value).toBe("Grade 1");
        expect(gradeItem.submenu[0].label).toBe("Grade 1");

        const streamItem = capturedProps.filterGroups[1].items[0];
        expect(streamItem.id).toBe("stream-filter");
        expect(streamItem.submenu[0].value).toBe("Science");
        expect(streamItem.submenu[0].label).toBe("Science");
    });
});
