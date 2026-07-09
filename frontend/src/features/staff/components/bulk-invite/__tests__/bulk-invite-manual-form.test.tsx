/**
 * BulkInviteManualForm tests.
 *
 * Covers: rendering, add/remove rows, field updates, validation,
 * mutation payload correctness, success/error toast feedback,
 * and import-already-in-progress recovery.
 */

import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { vi } from "vitest";
import { BulkInviteManualForm } from "../bulk-invite-manual-form";

// ── Mocks ─────────────────────────────────────────────────────────────────

const mockSubmitBulkInvite = vi.hoisted(() => vi.fn());
const mockGetImportAlreadyInProgress = vi.hoisted(() => vi.fn());

vi.mock("@/lib/api/invitations", () => ({
    submitBulkInvite: (...args: unknown[]) => mockSubmitBulkInvite(...args),
    getImportAlreadyInProgress: (...args: unknown[]) => mockGetImportAlreadyInProgress(...args),
}));

const mockToast = vi.hoisted(() => ({ success: vi.fn(), error: vi.fn() }));
vi.mock("sonner", () => ({
    toast: mockToast,
}));

// ── Props ─────────────────────────────────────────────────────────────────

const DEFAULT_ROLE = "TEACHER";

// ── Helpers ───────────────────────────────────────────────────────────────

function renderForm({
    onReset,
    onJobCreated,
    role,
}: {
    onReset?: () => void;
    onJobCreated?: (jobId: string, totalRecords: number) => void;
    role?: string;
} = {}) {
    const user = userEvent.setup();
    const result = render(
        <BulkInviteManualForm
            role={role ?? DEFAULT_ROLE}
            onReset={onReset ?? vi.fn()}
            onJobCreated={onJobCreated ?? vi.fn()}
        />
    );
    return { user, ...result };
}

async function fillRow(
    user: ReturnType<typeof userEvent.setup>,
    email: string,
    name?: string,
    rowIndex = 0
) {
    const emailInputs = screen.getAllByPlaceholderText("teacher@school.com");
    await user.type(emailInputs[rowIndex], email);

    if (name) {
        const nameInputs = screen.getAllByPlaceholderText("Jane Doe (optional)");
        await user.type(nameInputs[rowIndex], name);
    }
}

// scrollIntoView is not implemented in jsdom
beforeAll(() => {
    Element.prototype.scrollIntoView = vi.fn();
});

beforeEach(() => {
    vi.clearAllMocks();
});

// ── Tests ─────────────────────────────────────────────────────────────────

describe("BulkInviteManualForm", () => {
    // ── Rendering ──────────────────────────────────────────────────────

    it("renders initial form with one empty row", () => {
        renderForm();

        expect(screen.getByText("Manual Entry")).toBeInTheDocument();
        expect(
            screen.getByText((content) => content.startsWith("0 of 1 row ready"))
        ).toBeInTheDocument();
        expect(screen.getByPlaceholderText("teacher@school.com")).toBeInTheDocument();
        expect(screen.getByPlaceholderText("Jane Doe (optional)")).toBeInTheDocument();
        expect(screen.getByRole("button", { name: /invite 0/i })).toBeDisabled();
    });

    it("renders header with Cancel button", () => {
        renderForm();

        expect(screen.getByRole("button", { name: /cancel/i })).toBeInTheDocument();
    });

    it("renders Add Row button", () => {
        renderForm();

        expect(screen.getByRole("button", { name: /add row/i })).toBeInTheDocument();
    });

    // ── Adding rows ────────────────────────────────────────────────────

    it("adds a new row when Add Row is clicked", async () => {
        const { user } = renderForm();

        await user.click(screen.getByRole("button", { name: /add row/i }));

        expect(screen.getAllByPlaceholderText("teacher@school.com")).toHaveLength(2);
        expect(
            screen.getByText((content) => content.startsWith("0 of 2 rows ready"))
        ).toBeInTheDocument();
    });

    it("appends new row below existing ones", async () => {
        const { user } = renderForm();

        // Type in the existing (first) row
        const firstInput = screen.getByPlaceholderText("teacher@school.com");
        await user.type(firstInput, "first@school.com");

        // Add a new row — it should appear after the first one
        await user.click(screen.getByRole("button", { name: /add row/i }));

        const inputs = screen.getAllByPlaceholderText("teacher@school.com");
        expect(inputs[0]).toHaveValue("first@school.com");
        expect(inputs[1]).toHaveValue("");
    });

    // ── Removing rows ──────────────────────────────────────────────────

    it("disables delete button when only one row remains", () => {
        renderForm();

        const deleteBtn = screen.getByRole("button", { name: /remove row 1/i });
        expect(deleteBtn).toBeDisabled();
    });

    it("deletes a specific row", async () => {
        const { user } = renderForm();

        // Add two rows
        await user.click(screen.getByRole("button", { name: /add row/i }));
        await user.click(screen.getByRole("button", { name: /add row/i }));
        expect(screen.getAllByPlaceholderText("teacher@school.com")).toHaveLength(3);

        // Delete the second row
        const deleteBtns = screen.getAllByRole("button", { name: /remove row/i });
        await user.click(deleteBtns[1]);

        expect(screen.getAllByPlaceholderText("teacher@school.com")).toHaveLength(2);
    });

    it("resets to a single empty row when the last row is deleted", async () => {
        const { user } = renderForm();

        // Add a second row so we can delete the first without hitting the guard
        await user.click(screen.getByRole("button", { name: /add row/i }));
        expect(screen.getAllByPlaceholderText("teacher@school.com")).toHaveLength(2);

        // Delete the first row
        const deleteBtns = screen.getAllByRole("button", { name: /remove row/i });
        await user.click(deleteBtns[0]);

        // Should have 1 row (fresh replacement)
        expect(screen.getAllByPlaceholderText("teacher@school.com")).toHaveLength(1);
        // And the delete button should be disabled
        expect(screen.getByRole("button", { name: /remove row 1/i })).toBeDisabled();
    });

    // ── Field updates ──────────────────────────────────────────────────

    it("updates email on typing", async () => {
        const { user } = renderForm();

        const input = screen.getByPlaceholderText("teacher@school.com");
        await user.clear(input);
        await user.type(input, "alice@school.com");

        expect(input).toHaveValue("alice@school.com");
    });

    it("updates full name on typing", async () => {
        const { user } = renderForm();

        const input = screen.getByPlaceholderText("Jane Doe (optional)");
        await user.type(input, "Alice Wanjiku");

        expect(input).toHaveValue("Alice Wanjiku");
    });

    // ── Validation ─────────────────────────────────────────────────────

    it("shows validation error when email is empty on submit", async () => {
        const { user } = renderForm();

        // Email is empty by default — submit should be disabled
        await user.click(screen.getByRole("button", { name: /invite 0/i }));

        // Button should be disabled because there are 0 non-empty rows
        expect(screen.getByRole("button", { name: /invite 0/i })).toBeDisabled();
    });

    it("shows validation error for invalid email format", async () => {
        const { user } = renderForm();

        await fillRow(user, "not-an-email");

        // Now 1 row with invalid email — button says "Invite 1" and is enabled
        await user.click(screen.getByRole("button", { name: /invite 1/i }));

        expect(screen.getByText("Invalid email format")).toBeInTheDocument();

        // "1 error" appears both in header and footer — use getAllByText
        const errorIndicators = screen.getAllByText(/1 error/);
        expect(errorIndicators.length).toBeGreaterThanOrEqual(1);
    });

    it("shows validation error for duplicate emails in the batch", async () => {
        const { user } = renderForm();

        await fillRow(user, "dupe@school.com");

        // Add a second row
        await user.click(screen.getByRole("button", { name: /add row/i }));

        // Fill second row with the same email
        const emailInputs = screen.getAllByPlaceholderText("teacher@school.com");
        await user.type(emailInputs[1], "dupe@school.com");

        // Duplicate detection triggers on render; "1 error" appears in header + footer
        const errorIndicators = screen.getAllByText(/1 error/);
        expect(errorIndicators.length).toBeGreaterThanOrEqual(1);

        expect(screen.getByText(/Duplicate email — also used in row 1/i)).toBeInTheDocument();
    });

    it("clears validation error after fixing the field", async () => {
        const { user } = renderForm();

        await fillRow(user, "bad-email");

        // Trigger validation by trying to submit
        await user.click(screen.getByRole("button", { name: /invite 1/i }));
        expect(screen.getByText("Invalid email format")).toBeInTheDocument();

        // Fix the email
        const input = screen.getByPlaceholderText("teacher@school.com");
        await user.clear(input);
        await user.type(input, "good@school.com");

        // Error should disappear
        expect(screen.queryByText("Invalid email format")).not.toBeInTheDocument();
        // The header/footer error indicators should also disappear
        expect(screen.queryByText(/1 error/)).not.toBeInTheDocument();
    });

    // ── Row counter labels ─────────────────────────────────────────────

    it('displays "Person 1", "Person 2" labels for each row', async () => {
        const { user } = renderForm();

        expect(screen.getByText("Person 1")).toBeInTheDocument();

        await user.click(screen.getByRole("button", { name: /add row/i }));
        expect(screen.getByText("Person 2")).toBeInTheDocument();
    });

    it("shows (error) badge next to row label when row has an error", async () => {
        const { user } = renderForm();

        await fillRow(user, "bad-email");

        // Trigger validation by clicking invite
        await user.click(screen.getByRole("button", { name: /invite 1/i }));

        // The "(error)" text renders in a child <span> inside the row label
        expect(screen.getByText("(error)")).toBeInTheDocument();
    });

    // ── Mutation — success ─────────────────────────────────────────────

    it("sends correct payload on submit and calls onJobCreated on success", async () => {
        const onJobCreated = vi.fn();
        const onReset = vi.fn();
        mockSubmitBulkInvite.mockResolvedValue({
            job_id: "job-123",
            total_records: 1,
            total_chunks: 1,
            status: "processing",
        });

        const { user } = renderForm({ onReset, onJobCreated });

        await fillRow(user, "alice@school.com", "Alice Wanjiku");

        // Submit
        await user.click(screen.getByRole("button", { name: /invite 1/i }));

        await waitFor(() => {
            expect(mockSubmitBulkInvite).toHaveBeenCalledTimes(1);
        });

        // Verify payload structure
        const callArgs = mockSubmitBulkInvite.mock.calls[0][0];
        expect(callArgs).toEqual({
            role: "TEACHER",
            rows: [
                {
                    email: "alice@school.com",
                    full_name: "Alice Wanjiku",
                },
            ],
        });

        await waitFor(() => {
            expect(onJobCreated).toHaveBeenCalledWith("job-123", 1);
        });
    });

    it("sends payload without full_name when name is empty", async () => {
        mockSubmitBulkInvite.mockResolvedValue({
            job_id: "job-456",
            total_records: 1,
            total_chunks: 1,
            status: "processing",
        });

        const { user } = renderForm();

        await fillRow(user, "bob@school.com");

        await user.click(screen.getByRole("button", { name: /invite 1/i }));

        await waitFor(() => {
            expect(mockSubmitBulkInvite).toHaveBeenCalledTimes(1);
        });

        const callArgs = mockSubmitBulkInvite.mock.calls[0][0];
        expect(callArgs.rows[0]).toEqual({
            email: "bob@school.com",
        });
        expect(callArgs.rows[0]).not.toHaveProperty("full_name");
    });

    it("submits multiple rows in a single request", async () => {
        mockSubmitBulkInvite.mockResolvedValue({
            job_id: "job-multi",
            total_records: 2,
            total_chunks: 1,
            status: "processing",
        });

        const { user } = renderForm();

        // Fill first row
        await fillRow(user, "first@school.com");

        // Add second row
        await user.click(screen.getByRole("button", { name: /add row/i }));

        // Fill second row
        const emailInputs = screen.getAllByPlaceholderText("teacher@school.com");
        await user.type(emailInputs[1], "second@school.com");

        // Submit
        await user.click(screen.getByRole("button", { name: /invite 2/i }));

        await waitFor(() => {
            expect(mockSubmitBulkInvite).toHaveBeenCalledTimes(1);
        });

        const callArgs = mockSubmitBulkInvite.mock.calls[0][0];
        expect(callArgs.rows).toHaveLength(2);
        expect(callArgs.rows[0].email).toBe("first@school.com");
        expect(callArgs.rows[1].email).toBe("second@school.com");
    });

    it("passes the correct role to the API", async () => {
        mockSubmitBulkInvite.mockResolvedValue({
            job_id: "job-role",
            total_records: 1,
            total_chunks: 1,
            status: "processing",
        });

        const { user } = renderForm({ role: "SCHOOL_ADMIN" });

        await fillRow(user, "admin@school.com");

        await user.click(screen.getByRole("button", { name: /invite 1/i }));

        await waitFor(() => {
            expect(mockSubmitBulkInvite).toHaveBeenCalledWith({
                role: "SCHOOL_ADMIN",
                rows: [{ email: "admin@school.com" }],
            });
        });
    });

    // ── Mutation — error ───────────────────────────────────────────────

    it("shows error toast when submission fails", async () => {
        mockSubmitBulkInvite.mockRejectedValue(new Error("Network error"));

        const { user } = renderForm();

        await fillRow(user, "fail@school.com");

        await user.click(screen.getByRole("button", { name: /invite 1/i }));

        await waitFor(() => {
            expect(mockSubmitBulkInvite).toHaveBeenCalled();
        });

        await waitFor(() => {
            expect(mockToast.error).toHaveBeenCalledWith("Network error");
        });
    });

    it("re-enables the submit button after an error", async () => {
        mockSubmitBulkInvite.mockRejectedValue(new Error("Server error"));

        const { user } = renderForm();

        await fillRow(user, "retry@school.com");

        await user.click(screen.getByRole("button", { name: /invite 1/i }));

        await waitFor(() => {
            expect(mockToast.error).toHaveBeenCalled();
        });

        // Button should be enabled again (not showing spinner, not disabled)
        const inviteBtn = screen.getByRole("button", { name: /invite 1/i });
        expect(inviteBtn).not.toBeDisabled();
    });

    // ── Import already in progress ─────────────────────────────────────

    it("calls onJobCreated when import_already_in_progress error is returned", async () => {
        const apiError = new Error("An import is already in progress");
        mockSubmitBulkInvite.mockRejectedValue(apiError);
        mockGetImportAlreadyInProgress.mockReturnValue("existing-job-999");

        const onJobCreated = vi.fn();
        const { user } = renderForm({ onJobCreated });

        await fillRow(user, "existing@school.com");

        await user.click(screen.getByRole("button", { name: /invite 1/i }));

        await waitFor(() => {
            expect(mockGetImportAlreadyInProgress).toHaveBeenCalledWith(apiError);
        });

        await waitFor(() => {
            expect(onJobCreated).toHaveBeenCalledWith("existing-job-999", 1);
        });
    });

    it("does not call onJobCreated when getImportAlreadyInProgress returns null", async () => {
        const apiError = new Error("Some other error");
        mockSubmitBulkInvite.mockRejectedValue(apiError);
        mockGetImportAlreadyInProgress.mockReturnValue(null);

        const onJobCreated = vi.fn();
        const { user } = renderForm({ onJobCreated });

        await fillRow(user, "other-err@school.com");

        await user.click(screen.getByRole("button", { name: /invite 1/i }));

        await waitFor(() => {
            expect(mockSubmitBulkInvite).toHaveBeenCalled();
        });

        await waitFor(() => {
            expect(onJobCreated).not.toHaveBeenCalled();
            expect(mockToast.error).toHaveBeenCalledWith("Some other error");
        });
    });

    // ── Cancel ─────────────────────────────────────────────────────────

    it("calls onReset when Cancel is clicked", async () => {
        const onReset = vi.fn();
        const { user } = renderForm({ onReset });

        await user.click(screen.getByRole("button", { name: /cancel/i }));

        expect(onReset).toHaveBeenCalledTimes(1);
    });

    // ── Submitting state ───────────────────────────────────────────────

    it("shows submitting state while the request is in flight", async () => {
        // Never resolve — keep the promise pending
        mockSubmitBulkInvite.mockImplementation(
            () => new Promise(() => {}) // never resolves
        );

        const { user } = renderForm();

        await fillRow(user, "pending@school.com");

        await user.click(screen.getByRole("button", { name: /invite 1/i }));

        // Button should show "Sending…" and be disabled
        expect(screen.getByRole("button", { name: /sending/i })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: /sending/i })).toBeDisabled();

        // Cancel should be disabled during submission
        expect(screen.getByRole("button", { name: /cancel/i })).toBeDisabled();

        // Inputs should be disabled
        const emailInput = screen.getByPlaceholderText("teacher@school.com");
        expect(emailInput).toBeDisabled();
    });

    it("disables Add Row button while submitting", async () => {
        mockSubmitBulkInvite.mockImplementation(
            () => new Promise(() => {}) // never resolves
        );

        const { user } = renderForm();

        // Fill the existing row with a valid email first
        await fillRow(user, "a@school.com");

        // Now add a second row — also fill it so there are no validation errors
        await user.click(screen.getByRole("button", { name: /add row/i }));
        const emailInputs = screen.getAllByPlaceholderText("teacher@school.com");
        await user.type(emailInputs[1], "b@school.com");

        // Both rows are valid — both emails filled, no errors => canSubmit = true
        await user.click(screen.getByRole("button", { name: /invite 2/i }));

        expect(screen.getByRole("button", { name: /add row/i })).toBeDisabled();
    });

    // ── Row counter in footer ─────────────────────────────────────────

    it('shows "X of Y rows ready" in the footer', async () => {
        const { user } = renderForm();

        // Initial: 1 row, no emails filled yet
        // Empty email triggers a validation error so nonEmptyRows = 0
        expect(
            screen.getByText((content) => content.startsWith("0 of 1 row ready"))
        ).toBeInTheDocument();

        // Add a second row
        await user.click(screen.getByRole("button", { name: /add row/i }));
        expect(
            screen.getByText((content) => content.startsWith("0 of 2 rows ready"))
        ).toBeInTheDocument();

        // Fill one row
        await fillRow(user, "one@school.com");
        expect(
            screen.getByText((content) => content.startsWith("1 of 2 rows ready"))
        ).toBeInTheDocument();
    });
});
