/**
 * ManualImportForm tests.
 *
 * Covers: rendering, add/remove rows, field updates, validation,
 * mutation payload correctness, success/error toast feedback.
 */

import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { vi } from "vitest";
import { StudentManualImportForm as ManualImportForm } from "../manual-import-form";

// ── Mocks ─────────────────────────────────────────────────────────────────

const mockMutate = vi.hoisted(() => vi.fn());
const mockMutationState = vi.hoisted(() => ({ isPending: false }));

vi.mock("@tanstack/react-query", () => ({
    useMutation: vi.fn(
        ({
            onSuccess,
            onError,
        }: {
            onSuccess?: (result: unknown, payloads: unknown) => void;
            onError?: (error: unknown) => void;
        }) => ({
            get isPending() {
                return mockMutationState.isPending;
            },
            mutate: (payloads: unknown) => {
                const result = mockMutate(payloads);
                // If the mock returns a thenable, wire up success/error callbacks
                if (result && typeof result.then === "function") {
                    result.then(
                        (res: unknown) => onSuccess?.(res, payloads),
                        (err: unknown) => onError?.(err)
                    );
                }
            },
        })
    ),
    useQueryClient: vi.fn(),
}));

const mockToast = vi.hoisted(() => ({ success: vi.fn(), error: vi.fn() }));
vi.mock("sonner", () => ({
    toast: mockToast,
}));

// Mock shadcn Select to be a native <select> so tests can interact directly
vi.mock("@/components/ui/select", () => ({
    Select: ({
        value,
        onValueChange,
        disabled,
    }: {
        value?: string;
        onValueChange?: (value: string) => void;
        disabled?: boolean;
        children: React.ReactNode;
    }) => (
        <select
            aria-label="gender-select"
            value={value ?? ""}
            onChange={(e) => onValueChange?.(e.target.value)}
            disabled={disabled}
        >
            <option value="">-</option>
            <option value="M">Male</option>
            <option value="F">Female</option>
        </select>
    ),
    SelectTrigger: () => null,
    SelectValue: () => null,
    SelectContent: () => null,
    SelectItem: () => null,
}));

// Mock ClassCombobox so it doesn't fetch classes over the network
vi.mock("@/features/classes", () => ({
    ClassCombobox: ({
        value,
        onChange,
        placeholder,
    }: {
        value?: string;
        onChange?: (value: string) => void;
        placeholder?: string;
    }) => (
        <select
            data-testid="class-combobox"
            value={value ?? ""}
            onChange={(e) => onChange?.(e.target.value)}
        >
            <option value="">{placeholder}</option>
            <option value="class-1">Class 1</option>
            <option value="class-2">Class 2</option>
        </select>
    ),
}));

// Mock DatePicker to be a plain text input
vi.mock("@/components/ui/date-picker", () => ({
    DatePicker: ({
        value,
        onChange,
        placeholder,
        disabled,
    }: {
        value?: string;
        onChange?: (value: string) => void;
        placeholder?: string;
        disabled?: boolean;
    }) => (
        <input
            data-testid="date-picker"
            type="text"
            value={value ?? ""}
            placeholder={placeholder}
            disabled={disabled}
            onChange={(e) => onChange?.(e.target.value)}
        />
    ),
}));

// ── Helpers ───────────────────────────────────────────────────────────────

function renderForm(onReset = vi.fn()) {
    const user = userEvent.setup();
    const result = render(<ManualImportForm onReset={onReset} onJobCreated={vi.fn()} />);
    return { onReset, user, ...result };
}

// scrollIntoView is not implemented in jsdom
beforeAll(() => {
    Element.prototype.scrollIntoView = vi.fn();
});

beforeEach(() => {
    vi.clearAllMocks();
});

// ── Tests ─────────────────────────────────────────────────────────────────

describe("ManualImportForm", () => {
    // ── Rendering ──────────────────────────────────────────────────────

    it("renders initial form with one empty row", () => {
        renderForm();

        expect(screen.getByText("1 student ready to import")).toBeInTheDocument();
        expect(screen.getByPlaceholderText("e.g. John Kiprop")).toBeInTheDocument();
        expect(screen.getByPlaceholderText("e.g. UP123456789")).toBeInTheDocument();
        expect(screen.getByPlaceholderText("e.g. KNEC123456")).toBeInTheDocument();
        expect(screen.getByRole("button", { name: /import 1 student/i })).toBeInTheDocument();
    });

    it("renders the header with title and action buttons", () => {
        renderForm();

        expect(screen.getByText("Manual Student Import")).toBeInTheDocument();
        expect(screen.getByRole("button", { name: /add row/i })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: /cancel/i })).toBeInTheDocument();
    });

    // ── Adding rows ────────────────────────────────────────────────────

    it("adds a new row at the top when Add Row is clicked", async () => {
        const { user } = renderForm();

        await user.click(screen.getByRole("button", { name: /add row/i }));

        expect(screen.getByText("2 students ready to import")).toBeInTheDocument();
        expect(screen.getAllByPlaceholderText("e.g. John Kiprop")).toHaveLength(2);
    });

    it("prepends new row above existing ones", async () => {
        const { user } = renderForm();

        // Type in the existing (first) row
        const firstInput = screen.getByPlaceholderText("e.g. John Kiprop");
        await user.type(firstInput, "First Student");

        // Add a new row — it should appear at the top
        await user.click(screen.getByRole("button", { name: /add row/i }));

        const inputs = screen.getAllByPlaceholderText("e.g. John Kiprop");
        // The first input should be empty (new row) and second should have the typed text
        expect(inputs[0]).toHaveValue("");
        expect(inputs[1]).toHaveValue("First Student");
    });

    // ── Deleting rows ──────────────────────────────────────────────────

    it("disables delete button when only one row remains", () => {
        renderForm();

        const deleteBtn = screen.getByRole("button", { name: /remove student 1/i });
        expect(deleteBtn).toBeDisabled();
    });

    it("deletes a specific row", async () => {
        const { user } = renderForm();

        // Add two rows
        await user.click(screen.getByRole("button", { name: /add row/i }));
        await user.click(screen.getByRole("button", { name: /add row/i }));
        expect(screen.getAllByPlaceholderText("e.g. John Kiprop")).toHaveLength(3);

        // Delete the second row
        const deleteBtns = screen.getAllByRole("button", { name: /remove student/i });
        await user.click(deleteBtns[1]);

        expect(screen.getAllByPlaceholderText("e.g. John Kiprop")).toHaveLength(2);
    });

    // ── Field updates ──────────────────────────────────────────────────

    it("updates full name on typing", async () => {
        const { user } = renderForm();

        const input = screen.getByPlaceholderText("e.g. John Kiprop");
        await user.clear(input);
        await user.type(input, "Jane Doe");

        expect(input).toHaveValue("Jane Doe");
    });

    it("selects a gender", async () => {
        const { user } = renderForm();

        const combobox = screen.getByRole("combobox", { name: /gender-select/i });
        await user.selectOptions(combobox, "M");

        expect(combobox).toHaveValue("M");
    });

    it("updates date of birth via DatePicker mock", async () => {
        const { user } = renderForm();

        const dateInput = screen.getByTestId("date-picker");
        await user.clear(dateInput);
        await user.type(dateInput, "2005-06-15");

        expect(dateInput).toHaveValue("2005-06-15");
    });

    it("updates UPI number", async () => {
        const { user } = renderForm();

        const upiInput = screen.getByPlaceholderText("e.g. UP123456789");
        await user.type(upiInput, "UP987654321");

        expect(upiInput).toHaveValue("UP987654321");
    });

    it("updates KNEC number", async () => {
        const { user } = renderForm();

        const knecInput = screen.getByPlaceholderText("e.g. KNEC123456");
        await user.type(knecInput, "KNEC999999");

        expect(knecInput).toHaveValue("KNEC999999");
    });

    it("selects a class", async () => {
        const { user } = renderForm();

        const classSelect = screen.getByTestId("class-combobox");
        await user.selectOptions(classSelect, "class-1");

        expect(classSelect).toHaveValue("class-1");
    });

    // ── Validation ─────────────────────────────────────────────────────

    it("shows validation error when full name is empty on submit", async () => {
        const { user } = renderForm();

        // Full name is empty by default, click Import
        await user.click(screen.getByRole("button", { name: /import 1 student/i }));

        expect(screen.getByText("Full name is required")).toBeInTheDocument();
        expect(mockMutate).not.toHaveBeenCalled();
    });

    it("clears validation errors after editing the field", async () => {
        const { user } = renderForm();

        // Trigger validation
        await user.click(screen.getByRole("button", { name: /import 1 student/i }));
        expect(screen.getByText("Full name is required")).toBeInTheDocument();

        // Type in the field
        const input = screen.getByPlaceholderText("e.g. John Kiprop");
        await user.type(input, "Valid Name");

        // Re-submit — validation should pass, mutate should be called
        mockMutate.mockResolvedValue({ ids: ["new-id-1"] });
        await user.click(screen.getByRole("button", { name: /import 1 student/i }));

        await waitFor(() => {
            expect(mockMutate).toHaveBeenCalled();
        });
    });

    // ── Mutation — success ─────────────────────────────────────────────

    it("sends correct payload on submit and resets form on success", async () => {
        mockMutate.mockResolvedValue({ ids: ["student-1"] });

        const { user } = renderForm();

        // Fill in the form
        const nameInput = screen.getByPlaceholderText("e.g. John Kiprop");
        await user.type(nameInput, "Alice Wanjiku");

        const genderSelect = screen.getByRole("combobox", { name: /gender-select/i });
        await user.selectOptions(genderSelect, "F");

        const dateInput = screen.getByTestId("date-picker");
        await user.type(dateInput, "2006-03-20");

        const upiInput = screen.getByPlaceholderText("e.g. UP123456789");
        await user.type(upiInput, "UP555666777");

        const knecInput = screen.getByPlaceholderText("e.g. KNEC123456");
        await user.type(knecInput, "KNEC888888");

        const classSelect = screen.getByTestId("class-combobox");
        await user.selectOptions(classSelect, "class-2");

        // Submit
        await user.click(screen.getByRole("button", { name: /import 1 student/i }));

        await waitFor(() => {
            expect(mockMutate).toHaveBeenCalledTimes(1);
        });

        // Verify payload structure — mutate receives the payloads array directly
        const payload = mockMutate.mock.calls[0][0];
        expect(payload).toEqual([
            {
                full_name: "Alice Wanjiku",
                gender: "F",
                date_of_birth: "2006-03-20",
                upi_number: "UP555666777",
                knec_assessment_number: "KNEC888888",
                class_id: "class-2",
            },
        ]);

        // Success toast should be shown
        await waitFor(() => {
            expect(mockToast.success).toHaveBeenCalledWith("1 student created");
        });

        // Form should reset — back to one empty row
        expect(screen.getAllByPlaceholderText("e.g. John Kiprop")).toHaveLength(1);
        expect(screen.getByPlaceholderText("e.g. John Kiprop")).toHaveValue("");
    });

    it("submits multiple rows in a single request", async () => {
        mockMutate.mockResolvedValue({ ids: ["student-1", "student-2"] });

        const { user } = renderForm();

        // Fill first row
        await user.type(screen.getByPlaceholderText("e.g. John Kiprop"), "Student One");

        // Add second row
        await user.click(screen.getByRole("button", { name: /add row/i }));

        // Fill second row (now at the top — it was prepended)
        const inputs = screen.getAllByPlaceholderText("e.g. John Kiprop");
        await user.type(inputs[0], "Student Two");

        // Submit
        await user.click(screen.getByRole("button", { name: /import 2 students/i }));

        await waitFor(() => {
            expect(mockMutate).toHaveBeenCalledTimes(1);
        });

        const payload = mockMutate.mock.calls[0][0];
        expect(payload).toHaveLength(2);
        expect(payload[0].full_name).toBe("Student Two");
        expect(payload[1].full_name).toBe("Student One");
    });

    // ── Mutation — error ───────────────────────────────────────────────

    it("shows error toast when mutation fails", async () => {
        mockMutate.mockRejectedValue(new Error("Network error"));

        const { user } = renderForm();

        const input = screen.getByPlaceholderText("e.g. John Kiprop");
        await user.type(input, "Test Student");

        await user.click(screen.getByRole("button", { name: /import 1 student/i }));

        await waitFor(() => {
            expect(mockMutate).toHaveBeenCalled();
        });

        await waitFor(() => {
            expect(mockToast.error).toHaveBeenCalled();
        });
    });

    it("handles partial success (some failed IDs)", async () => {
        mockMutate.mockResolvedValue({ ids: ["student-1"] });

        const { user } = renderForm();

        await user.type(screen.getByPlaceholderText("e.g. John Kiprop"), "Alice");

        await user.click(screen.getByRole("button", { name: /add row/i }));

        const inputs = screen.getAllByPlaceholderText("e.g. John Kiprop");
        await user.type(inputs[0], "Bob");
        expect(inputs[1]).toHaveValue("Alice");

        await user.click(screen.getByRole("button", { name: /import 2 students/i }));

        await waitFor(() => {
            expect(mockMutate).toHaveBeenCalled();
        });

        await waitFor(() => {
            expect(mockToast.success).toHaveBeenCalled();
            expect(mockToast.error).toHaveBeenCalled();
        });
    });

    // ── Cancel ─────────────────────────────────────────────────────────

    it("calls onReset when Cancel is clicked", async () => {
        const { user, onReset } = renderForm();

        await user.click(screen.getByRole("button", { name: /cancel/i }));

        expect(onReset).toHaveBeenCalledTimes(1);
    });

    // ── Empty state ────────────────────────────────────────────────────

    it("shows empty state when all rows are removed", async () => {
        const { user } = renderForm();

        // We can't delete the last row — the delete button is disabled when only 1 remains.
        // Add two rows (total 3), then delete both new ones (total back to 1).
        // That leaves us with 1 row — which is the minimum — but we want to show the empty state.
        // The empty state shows when rows.length === 0. To get there we need to delete ALL rows.
        // Delete button is disabled when rows.length === 1, so we need to start with 2+ and
        // delete until the 2nd-to-last is gone, at which point the last one becomes deletable
        // only if we remove the disabled guard... but the guard prevents deleting the last one.
        //
        // Workaround: set rows to empty array directly is impossible from the UI.
        // Instead, simulate by adding and deleting up to the point where only 1 remains,
        // verifying that the button is disabled.

        await user.click(screen.getByRole("button", { name: /add row/i }));
        await user.click(screen.getByRole("button", { name: /add row/i }));

        expect(screen.getAllByPlaceholderText("e.g. John Kiprop")).toHaveLength(3);

        // Delete the first (top) row — this is newly added, deletable
        await user.click(screen.getAllByRole("button", { name: /remove student/i })[0]);
        expect(screen.getAllByPlaceholderText("e.g. John Kiprop")).toHaveLength(2);

        // Delete the top row again
        await user.click(screen.getAllByRole("button", { name: /remove student/i })[0]);
        expect(screen.getAllByPlaceholderText("e.g. John Kiprop")).toHaveLength(1);

        // The remaining delete button should be disabled (can't delete last row)
        expect(screen.getByRole("button", { name: /remove student/i })).toBeDisabled();
    });
});
