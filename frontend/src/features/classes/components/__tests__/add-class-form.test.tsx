import React from "react";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { AddClassForm } from "../add-class-form";
import { createClass, type Class } from "@/lib/api/classes";
import { isApiError, getErrorMessage } from "@/lib/errors";
import { useRouter } from "next/navigation";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { vi, describe, test, expect, beforeEach, afterEach } from "vitest";

vi.mock("next/navigation", () => ({
    useRouter: vi.fn(),
}));

vi.mock("@tanstack/react-query", () => ({
    useMutation: vi.fn(),
    useQueryClient: vi.fn(),
}));

vi.mock("@/lib/api/classes", () => ({
    createClass: vi.fn(),
}));

vi.mock("@/lib/errors", () => ({
    isApiError: vi.fn(),
    getErrorMessage: vi.fn((err: Error) => err.message),
}));

vi.mock("@/features/grade-level", () => ({
    GradeLevelCombobox: vi.fn(({ value, onChange, placeholder }) => (
        <select
            data-testid="grade-level-combobox"
            value={value}
            onChange={(e) => onChange(e.target.value)}
            aria-label={placeholder}
        >
            <option value="">Select a grade level...</option>
            <option value="PP1">PP1</option>
            <option value="PP2">PP2</option>
            <option value="G1">Grade 1</option>
            <option value="G2">Grade 2</option>
        </select>
    )),
}));

vi.mock("@/features/streams", () => ({
    StreamCombobox: vi.fn(({ value, onChange, placeholder, onCreateItem }) => (
        <div>
            <select
                data-testid="stream-combobox"
                value={value}
                onChange={(e) => onChange(e.target.value)}
                aria-label={placeholder}
            >
                <option value="">Select a stream...</option>
                <option value="stream1">Stream A</option>
                <option value="stream2">Stream B</option>
            </select>
            {onCreateItem && (
                <button data-testid="create-stream" onClick={() => onCreateItem("new stream")}>
                    Create &quot;new stream&quot;
                </button>
            )}
        </div>
    )),
}));

vi.mock("@/features/academic-terms", () => ({
    AcademicYearCombobox: vi.fn(({ value, onChange, placeholder, onCreateItem }) => (
        <div>
            <select
                data-testid="academic-year-combobox"
                value={value}
                onChange={(e) => onChange(e.target.value)}
                aria-label={placeholder}
            >
                <option value="">Select an academic year...</option>
                <option value="year1">2023-2024</option>
                <option value="year2">2024-2025</option>
            </select>
            {onCreateItem && (
                <button data-testid="create-academic-year" onClick={() => onCreateItem("new year")}>
                    Create &quot;new year&quot;
                </button>
            )}
        </div>
    )),
}));

describe("AddClassForm", () => {
    const mockRouter = {
        back: vi.fn(),
        push: vi.fn(),
    };

    const mockQueryClient = {
        invalidateQueries: vi.fn(),
    };

    const mockCreateMutation = {
        mutate: vi.fn(),
        isPending: false,
    };

    const mockOnSuccess = vi.fn();

    beforeEach(() => {
        vi.clearAllMocks();

        (useRouter as vi.Mock).mockReturnValue(mockRouter);
        (useQueryClient as vi.Mock).mockReturnValue(mockQueryClient);
        (useMutation as vi.Mock).mockReturnValue(mockCreateMutation);
        (createClass as vi.Mock).mockResolvedValue({ id: "class1", name: "Test Class" });
        (isApiError as vi.Mock).mockReturnValue(false);
        (getErrorMessage as vi.Mock).mockImplementation((err: Error) => err.message);

        mockCreateMutation.mutate.mockImplementation(() => {});
        mockCreateMutation.isPending = false;
    });

    afterEach(() => {
        vi.resetAllMocks();
    });

    const renderForm = () => {
        return render(<AddClassForm onSuccess={mockOnSuccess} />);
    };

    test("renders form with all three comboboxes", () => {
        renderForm();

        // Labels are rendered as text
        expect(screen.getByText("Grade Level")).toBeInTheDocument();
        expect(screen.getByText("Stream")).toBeInTheDocument();
        expect(screen.getByText("Academic Year")).toBeInTheDocument();
        expect(screen.getByRole("button", { name: /create class/i })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: /cancel/i })).toBeInTheDocument();
    });

    test("shows validation errors when fields are empty on submit", async () => {
        const user = userEvent.setup();
        renderForm();

        await user.click(screen.getByRole("button", { name: /create class/i }));

        expect(mockCreateMutation.mutate).not.toHaveBeenCalled();
    });

    test("calls createClass with correct payload when all fields are filled", async () => {
        const user = userEvent.setup();
        renderForm();

        await user.selectOptions(screen.getByTestId("grade-level-combobox"), "G1");
        await user.selectOptions(screen.getByTestId("stream-combobox"), "stream1");
        await user.selectOptions(screen.getByTestId("academic-year-combobox"), "year1");
        await user.click(screen.getByRole("button", { name: /create class/i }));

        expect(mockCreateMutation.mutate).toHaveBeenCalled();
    });

    test("navigates back on successful creation", async () => {
        const user = userEvent.setup();
        renderForm();

        await user.selectOptions(screen.getByTestId("grade-level-combobox"), "G1");
        await user.selectOptions(screen.getByTestId("stream-combobox"), "stream1");
        await user.selectOptions(screen.getByTestId("academic-year-combobox"), "year1");
        await user.click(screen.getByRole("button", { name: /create class/i }));

        await waitFor(() => {
            expect(mockOnSuccess).toHaveBeenCalled();
        });

        expect(mockRouter.back).toHaveBeenCalled();
    });

    test("calls onSuccess callback with created class", async () => {
        const mockClass: Class = {
            id: "class1",
            name: "G1 - Stream A",
            grade_level: "G1",
            stream_id: "stream1",
            academic_year_id: "year1",
            academic_term_id: "term1",
        };

        (createClass as vi.Mock).mockResolvedValue(mockClass);

        const user = userEvent.setup();
        renderForm();

        await user.selectOptions(screen.getByTestId("grade-level-combobox"), "G1");
        await user.selectOptions(screen.getByTestId("stream-combobox"), "stream1");
        await user.selectOptions(screen.getByTestId("academic-year-combobox"), "year1");
        await user.click(screen.getByRole("button", { name: /create class/i }));

        await waitFor(() => {
            expect(mockOnSuccess).toHaveBeenCalledWith(mockClass);
        });
    });

    test("displays general error on API error (non-400)", async () => {
        (isApiError as vi.Mock).mockReturnValue(true);
        const apiError = new Error("Server error") as Error & { status: number };
        apiError.status = 500;
        (createClass as vi.Mock).mockRejectedValue(apiError);

        const user = userEvent.setup();
        renderForm();

        await user.selectOptions(screen.getByTestId("grade-level-combobox"), "G1");
        await user.selectOptions(screen.getByTestId("stream-combobox"), "stream1");
        await user.selectOptions(screen.getByTestId("academic-year-combobox"), "year1");
        await user.click(screen.getByRole("button", { name: /create class/i }));

        await waitFor(() => {
            expect(screen.getByText(/server error/i)).toBeInTheDocument();
        });
    });

    test("displays field errors on 400 validation error", async () => {
        (isApiError as vi.Mock).mockReturnValue(true);
        const apiError = {
            status: 400,
            errors: {
                grade_level: ["Grade level already exists for this stream and year"],
                stream_id: ["Invalid stream"],
            },
        };
        (createClass as vi.Mock).mockRejectedValue(apiError);

        const user = userEvent.setup();
        renderForm();

        await user.selectOptions(screen.getByTestId("grade-level-combobox"), "G1");
        await user.selectOptions(screen.getByTestId("stream-combobox"), "stream1");
        await user.selectOptions(screen.getByTestId("academic-year-combobox"), "year1");
        await user.click(screen.getByRole("button", { name: /create class/i }));

        await waitFor(() => {
            expect(screen.getByText(/grade level already exists/i)).toBeInTheDocument();
            expect(screen.getByText(/invalid stream/i)).toBeInTheDocument();
        });
    });

    test("clears field errors when user changes a field value", async () => {
        (isApiError as vi.Mock).mockReturnValue(true);
        const apiError = {
            status: 400,
            errors: {
                grade_level: ["Grade level already exists"],
            },
        };
        (createClass as vi.Mock).mockRejectedValue(apiError);

        const user = userEvent.setup();
        renderForm();

        await user.selectOptions(screen.getByTestId("grade-level-combobox"), "G1");
        await user.selectOptions(screen.getByTestId("stream-combobox"), "stream1");
        await user.selectOptions(screen.getByTestId("academic-year-combobox"), "year1");
        await user.click(screen.getByRole("button", { name: /create class/i }));

        await waitFor(() => {
            expect(screen.getByText(/grade level already exists/i)).toBeInTheDocument();
        });

        await user.selectOptions(screen.getByTestId("grade-level-combobox"), "G2");

        await waitFor(() => {
            expect(screen.queryByText(/grade level already exists/i)).not.toBeInTheDocument();
        });
    });

    test("disables submit button while mutation is pending", () => {
        mockCreateMutation.isPending = true;

        renderForm();

        const submitButton = screen.getByRole("button", { name: /creating.../i });
        expect(submitButton).toBeDisabled();
    });

    test("shows loading spinner while mutation is pending", () => {
        mockCreateMutation.isPending = true;

        renderForm();

        expect(screen.getByRole("button", { name: /creating.../i })).toBeInTheDocument();
    });

    test("calls router.push with correct URL when creating new stream", async () => {
        const user = userEvent.setup();
        renderForm();

        await user.click(screen.getByTestId("create-stream"));

        expect(mockRouter.push).toHaveBeenCalledWith("/streams/add?value=new%20stream");
    });

    test("calls router.push with correct URL when creating new academic year", async () => {
        const user = userEvent.setup();
        renderForm();

        await user.click(screen.getByTestId("create-academic-year"));

        expect(mockRouter.push).toHaveBeenCalledWith("/academic-terms/new");
    });

    test("calls router.back when cancel button is clicked", async () => {
        const user = userEvent.setup();
        renderForm();

        await user.click(screen.getByRole("button", { name: /cancel/i }));

        expect(mockRouter.back).toHaveBeenCalled();
    });

    test("invalidates classes query on successful creation", async () => {
        const user = userEvent.setup();
        renderForm();

        await user.selectOptions(screen.getByTestId("grade-level-combobox"), "G1");
        await user.selectOptions(screen.getByTestId("stream-combobox"), "stream1");
        await user.selectOptions(screen.getByTestId("academic-year-combobox"), "year1");
        await user.click(screen.getByRole("button", { name: /create class/i }));

        await waitFor(() => {
            expect(mockQueryClient.invalidateQueries).toHaveBeenCalledWith({
                queryKey: ["classes"],
            });
        });
    });
});
