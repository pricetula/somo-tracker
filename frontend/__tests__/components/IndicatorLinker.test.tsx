/**
 * Tests for the IndicatorLinker component.
 *
 * Covers dialog open/close, learning area browsing, indicator selection,
 * already-linked filtering, and the link flow.
 */

import { describe, it, expect, beforeEach, vi } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "../setup/test-utils";
import { http, HttpResponse } from "msw";
import { server } from "../setup/msw-server";

import { IndicatorLinker } from "@/features/assessment";

// ─── Mock data ────────────────────────────────────────────────────────────

const LEARNING_AREAS_RESPONSE = {
    learning_areas: [
        { id: "la-1", name: "English", code: "ENG", education_level: "Upper_Primary" },
        { id: "la-2", name: "Mathematics", code: "MATH", education_level: "Upper_Primary" },
    ],
    total: 2,
};

const LA1_TREE_RESPONSE = {
    id: "la-1",
    name: "English",
    code: "ENG",
    education_level: "Upper_Primary",
    strands: [
        {
            id: "strand-1",
            learning_area_id: "la-1",
            name: "Listening and Speaking",
            sub_strands: [
                {
                    id: "ss-1",
                    strand_id: "strand-1",
                    name: "Oral Narratives",
                    performance_indicators: [
                        {
                            id: "pi-1",
                            sub_strand_id: "ss-1",
                            description: "Retell a simple oral narrative",
                            sequence_order: 1,
                        },
                        {
                            id: "pi-2",
                            sub_strand_id: "ss-1",
                            description: "Identify characters in a narrative",
                            sequence_order: 2,
                        },
                    ],
                },
            ],
        },
        {
            id: "strand-2",
            learning_area_id: "la-1",
            name: "Reading",
            sub_strands: [
                {
                    id: "ss-2",
                    strand_id: "strand-2",
                    name: "Comprehension",
                    performance_indicators: [
                        {
                            id: "pi-3",
                            sub_strand_id: "ss-2",
                            description: "Answer literal questions from a passage",
                            sequence_order: 1,
                        },
                    ],
                },
            ],
        },
    ],
};

// ─── Link endpoint mock ───────────────────────────────────────────────────

const LINK_ENDPOINT = "http://localhost:3000/api/v1/assessment/blueprints/blueprint-001/indicators";

function setupLinkMock(status = 200) {
    server.use(
        http.post(LINK_ENDPOINT, () => {
            if (status === 200) return new HttpResponse(null, { status: 200 });
            return HttpResponse.json(
                { code: "validation_error", message: "Invalid indicator IDs" },
                { status: 400 }
            );
        })
    );
}

describe("IndicatorLinker", () => {
    beforeEach(() => {
        vi.clearAllMocks();

        // Register learning areas mock
        server.use(
            http.get("http://localhost:3000/api/v1/curriculum/learning-areas", () => {
                return HttpResponse.json(LEARNING_AREAS_RESPONSE);
            })
        );
    });

    it("renders the dialog with learning areas when open", async () => {
        setupLinkMock();
        renderWithProviders(
            <IndicatorLinker
                open={true}
                onOpenChange={vi.fn()}
                blueprintId="blueprint-001"
                alreadyLinked={[]}
            />
        );

        // Title should be visible
        expect(screen.getByText("Link Indicators from Curriculum")).toBeInTheDocument();

        // Learning areas should load
        await waitFor(() => {
            expect(screen.getByText("English")).toBeInTheDocument();
        });
        expect(screen.getByText("Mathematics")).toBeInTheDocument();
    });

    it("does not render content when closed", () => {
        setupLinkMock();
        renderWithProviders(
            <IndicatorLinker
                open={false}
                onOpenChange={vi.fn()}
                blueprintId="blueprint-001"
                alreadyLinked={[]}
            />
        );

        expect(screen.queryByText("Link Indicators from Curriculum")).not.toBeInTheDocument();
    });

    it("shows already-linked count when indicators are already linked", async () => {
        setupLinkMock();
        renderWithProviders(
            <IndicatorLinker
                open={true}
                onOpenChange={vi.fn()}
                blueprintId="blueprint-001"
                alreadyLinked={["pi-1", "pi-2"]}
            />
        );

        await waitFor(() => {
            expect(screen.getByText(/2 indicators already linked/)).toBeInTheDocument();
        });
    });

    it("expands learning area and shows strands on click", async () => {
        const user = userEvent.setup();
        setupLinkMock();

        // Register tree endpoint
        server.use(
            http.get(
                "http://localhost:3000/api/v1/curriculum/learning-areas/la-1/tree",
                () => {
                    return HttpResponse.json(LA1_TREE_RESPONSE);
                },
                { once: true }
            )
        );

        renderWithProviders(
            <IndicatorLinker
                open={true}
                onOpenChange={vi.fn()}
                blueprintId="blueprint-001"
                alreadyLinked={[]}
            />
        );

        // Wait for learning areas to load
        await waitFor(() => {
            expect(screen.getByText("English")).toBeInTheDocument();
        });

        // Click on English to expand
        await user.click(screen.getByText("English"));

        // Strands should appear
        await waitFor(() => {
            expect(screen.getByText("Listening and Speaking")).toBeInTheDocument();
        });
        expect(screen.getByText("Reading")).toBeInTheDocument();
    });

    it("allows selecting indicators and shows count", async () => {
        const user = userEvent.setup();
        setupLinkMock();

        server.use(
            http.get(
                "http://localhost:3000/api/v1/curriculum/learning-areas/la-1/tree",
                () => {
                    return HttpResponse.json(LA1_TREE_RESPONSE);
                },
                { once: true }
            )
        );

        renderWithProviders(
            <IndicatorLinker
                open={true}
                onOpenChange={vi.fn()}
                blueprintId="blueprint-001"
                alreadyLinked={[]}
            />
        );

        // Expand English → Listening and Speaking
        await waitFor(() => expect(screen.getByText("English")).toBeInTheDocument());
        await user.click(screen.getByText("English"));
        await waitFor(() => expect(screen.getByText("Listening and Speaking")).toBeInTheDocument());

        // Expand Listening and Speaking → Oral Narratives
        await user.click(screen.getByText("Listening and Speaking"));
        await waitFor(() => expect(screen.getByText("Oral Narratives")).toBeInTheDocument());

        // Expand Oral Narratives to reveal indicators
        await user.click(screen.getByText("Oral Narratives"));

        // Indicators should now be visible as checkboxes
        await waitFor(() => {
            expect(screen.getByText("Retell a simple oral narrative")).toBeInTheDocument();
        });

        // Check one indicator
        const checkboxes = screen.getAllByRole("checkbox");
        await user.click(checkboxes[0]);

        // Selected count should update
        await waitFor(() => {
            expect(screen.getByText(/1 selected to link/)).toBeInTheDocument();
        });
    });

    it("filters learning areas by search", async () => {
        const user = userEvent.setup();
        setupLinkMock();

        renderWithProviders(
            <IndicatorLinker
                open={true}
                onOpenChange={vi.fn()}
                blueprintId="blueprint-001"
                alreadyLinked={[]}
            />
        );

        await waitFor(() => {
            expect(screen.getByText("English")).toBeInTheDocument();
        });

        // Type search
        const searchInput = screen.getByPlaceholderText("Search learning areas...");
        await user.type(searchInput, "Math");

        // English should disappear, Mathematics should remain
        expect(screen.queryByText("English")).not.toBeInTheDocument();
        expect(screen.getByText("Mathematics")).toBeInTheDocument();
    });

    it("calls linkIndicators on 'Link Selected' button click", async () => {
        const user = userEvent.setup();
        const onOpenChange = vi.fn();

        // Set up a mock that we can assert on
        let linkedIds: string[] | null = null;
        server.use(
            http.post(LINK_ENDPOINT, async ({ request }) => {
                const body = (await request.json()) as { indicator_ids: string[] };
                linkedIds = body.indicator_ids;
                return new HttpResponse(null, { status: 200 });
            })
        );

        server.use(
            http.get(
                "http://localhost:3000/api/v1/curriculum/learning-areas/la-1/tree",
                () => {
                    return HttpResponse.json(LA1_TREE_RESPONSE);
                },
                { once: true }
            )
        );

        renderWithProviders(
            <IndicatorLinker
                open={true}
                onOpenChange={onOpenChange}
                blueprintId="blueprint-001"
                alreadyLinked={[]}
            />
        );

        // Expand and select an indicator
        await waitFor(() => expect(screen.getByText("English")).toBeInTheDocument());
        await user.click(screen.getByText("English"));
        await waitFor(() => expect(screen.getByText("Listening and Speaking")).toBeInTheDocument());
        await user.click(screen.getByText("Listening and Speaking"));
        await waitFor(() => expect(screen.getByText("Oral Narratives")).toBeInTheDocument());
        await user.click(screen.getByText("Oral Narratives"));
        await waitFor(() =>
            expect(screen.getByText("Retell a simple oral narrative")).toBeInTheDocument()
        );

        // Check the checkbox
        const checkboxes = screen.getAllByRole("checkbox");
        await user.click(checkboxes[0]);

        // Click "Link Selected"
        const linkButton = screen.getByRole("button", { name: /Link Selected/ });
        await user.click(linkButton);

        // Assert the API was called with the right indicator IDs
        await waitFor(() => {
            expect(linkedIds).toEqual(["pi-1"]);
        });

        // Dialog should close on success
        expect(onOpenChange).toHaveBeenCalledWith(false);
    });

    it("disables Link button when no indicators are selected", async () => {
        setupLinkMock();

        renderWithProviders(
            <IndicatorLinker
                open={true}
                onOpenChange={vi.fn()}
                blueprintId="blueprint-001"
                alreadyLinked={[]}
            />
        );

        await waitFor(() => expect(screen.getByText("English")).toBeInTheDocument());

        const linkButton = screen.getByRole("button", { name: /Link Selected/ });
        expect(linkButton).toBeDisabled();
    });

    it("shows an error state when learning areas fail to load", async () => {
        server.use(
            http.get("http://localhost:3000/api/v1/curriculum/learning-areas", () => {
                return new HttpResponse(null, { status: 500 });
            })
        );

        renderWithProviders(
            <IndicatorLinker
                open={true}
                onOpenChange={vi.fn()}
                blueprintId="blueprint-001"
                alreadyLinked={[]}
            />
        );

        await waitFor(() => {
            expect(
                screen.getByText("Failed to load learning areas. Please try again.")
            ).toBeInTheDocument();
        });
    });
});
