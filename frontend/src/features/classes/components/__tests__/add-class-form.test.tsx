import React from "react";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { vi, describe, test, expect, beforeEach, afterEach } from "vitest";

// Mock the useCreateClass hook BEFORE importing the component
const mockCreateClassMutation = {
    mutate: vi.fn(),
    isPending: false,
    isError: false,
    error: null,
};

vi.mock("../../hooks/use-classes", () => ({
    useCreateClass: vi.fn(() => mockCreateClassMutation),
}));

vi.mock("next/navigation", () => ({
    useRouter: vi.fn(),
}));

vi.mock("@tanstack/react-query", () => ({
    useQueryClient: vi.fn(),
    useMutation: vi.fn(),
}));

vi.mock("@/lib/errors", () => ({
    isApiError: vi.fn(),
    getErrorMessage: vi.fn((err: Error) => err.message),
}));

vi.mock("sonner", () => ({
    toast: {
        success: vi.fn(),
        error: vi.fn(),
    },
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
    StreamCombobox: vi.fn(({ value, onChange, placeholder }) => (
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
    )),
}));

// Import the component and utilities AFTER mocks are set up
import { AddClassForm } from "../add-class-form";
import { isApiError, getErrorMessage } from "@/lib/errors";
import { useRouter } from "next/navigation";
import { toast } from "sonner";

// Safely wrap imported mocks with vi.mocked() to retain correct TypeScript signatures
const mockedUseRouter = vi.mocked(useRouter);
const mockedIsApiError = vi.mocked(isApiError);
const mockedGetErrorMessage = vi.mocked(getErrorMessage);

describe("AddClassForm", () => {
    const mockRouter = {
        back: vi.fn(),
        push: vi.fn(),
        forward: vi.fn(),
        refresh: vi.fn(),
        replace: vi.fn(),
        prefetch: vi.fn(),
    };

    const mockOnSuccess = vi.fn();

    beforeEach(() => {
        vi.clearAllMocks();

        mockedUseRouter.mockReturnValue(mockRouter);
        mockedIsApiError.mockReturnValue(false);
        mockedGetErrorMessage.mockImplementation((err) => (err as { message: string }).message);

        mockCreateClassMutation.mutate.mockImplementation((_data, options) => {
            setTimeout(() => {
                options?.onSuccess?.();
            }, 0);
        });
        mockCreateClassMutation.isPending = false;
        mockCreateClassMutation.isError = false;
        mockCreateClassMutation.error = null;
    });

    afterEach(() => {
        vi.resetAllMocks();
    });

    const renderForm = () => {
        return render(<AddClassForm onSuccess={mockOnSuccess} />);
    };

    test("renders form with grade level and stream comboboxes", () => {
        renderForm();

        expect(screen.getByText("Grade Level")).toBeInTheDocument();
        expect(screen.getByText("Stream")).toBeInTheDocument();
        expect(screen.getByRole("button", { name: /create class/i })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: /cancel/i })).toBeInTheDocument();
        expect(screen.queryByText("Academic Year")).not.toBeInTheDocument();
    });

    test("shows validation errors when fields are empty on submit", async () => {
        const user = userEvent.setup();
        renderForm();

        await user.click(screen.getByRole("button", { name: /create class/i }));

        expect(mockCreateClassMutation.mutate).not.toHaveBeenCalled();
    });

    test("calls useCreateClass mutate with correct payload when all fields are filled", async () => {
        const user = userEvent.setup();
        renderForm();

        await user.selectOptions(screen.getByTestId("grade-level-combobox"), "G1");
        await user.selectOptions(screen.getByTestId("stream-combobox"), "stream1");
        await user.click(screen.getByRole("button", { name: /create class/i }));

        expect(mockCreateClassMutation.mutate).toHaveBeenCalledWith(
            expect.objectContaining({
                grade_level: "G1",
                stream_id: "stream1",
                student_ids: [],
            }),
            expect.any(Object)
        );
    });

    test("navigates back on successful creation", async () => {
        const user = userEvent.setup();
        renderForm();

        await user.selectOptions(screen.getByTestId("grade-level-combobox"), "G1");
        await user.selectOptions(screen.getByTestId("stream-combobox"), "stream1");
        await user.click(screen.getByRole("button", { name: /create class/i }));

        await waitFor(() => {
            expect(mockOnSuccess).toHaveBeenCalled();
        });

        expect(mockRouter.back).toHaveBeenCalled();
    });

    test("displays general error on API error (non-400)", async () => {
        mockedIsApiError.mockReturnValue(false);
        const apiError = new Error("Server error") as Error & { status: number };
        apiError.status = 500;

        mockCreateClassMutation.mutate.mockImplementation((_data, options) => {
            setTimeout(() => {
                options?.onError?.(apiError);
            }, 0);
        });

        const user = userEvent.setup();
        renderForm();

        await user.selectOptions(screen.getByTestId("grade-level-combobox"), "G1");
        await user.selectOptions(screen.getByTestId("stream-combobox"), "stream1");
        await user.click(screen.getByRole("button", { name: /create class/i }));

        await waitFor(() => {
            expect(toast.error).toHaveBeenCalledWith("Server error");
        });
    });

    test("displays field errors on 400 validation error", async () => {
        mockedIsApiError.mockReturnValue(true);
        const apiError = {
            status: 400,
            errors: {
                grade_level: ["Grade level already exists for this stream"],
                stream_id: ["Invalid stream"],
            },
        };

        mockCreateClassMutation.mutate.mockImplementation((_data, options) => {
            setTimeout(() => {
                options?.onError?.(apiError);
            }, 0);
        });

        const user = userEvent.setup();
        renderForm();

        await user.selectOptions(screen.getByTestId("grade-level-combobox"), "G1");
        await user.selectOptions(screen.getByTestId("stream-combobox"), "stream1");
        await user.click(screen.getByRole("button", { name: /create class/i }));

        await waitFor(() => {
            expect(screen.getByText(/grade level already exists/i)).toBeInTheDocument();
            expect(screen.getByText(/invalid stream/i)).toBeInTheDocument();
        });
    });

    test("clears field errors when user changes a field value", async () => {
        mockedIsApiError.mockReturnValue(true);
        const apiError = {
            status: 400,
            errors: {
                grade_level: ["Grade level already exists"],
            },
        };

        mockCreateClassMutation.mutate.mockImplementation((_data, options) => {
            setTimeout(() => {
                options?.onError?.(apiError);
            }, 0);
        });

        const user = userEvent.setup();
        renderForm();

        await user.selectOptions(screen.getByTestId("grade-level-combobox"), "G1");
        await user.selectOptions(screen.getByTestId("stream-combobox"), "stream1");
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
        mockCreateClassMutation.isPending = true;

        renderForm();

        const submitButton = screen.getByRole("button", { name: /creating.../i });
        expect(submitButton).toBeDisabled();
    });

    test("shows loading text while mutation is pending", () => {
        mockCreateClassMutation.isPending = true;

        renderForm();

        expect(screen.getByRole("button", { name: /creating.../i })).toBeInTheDocument();
    });

    test("calls router.push with correct URL when creating new stream", async () => {
        const user = userEvent.setup();
        renderForm();

        await user.click(screen.getByTestId("create-stream"));

        expect(mockRouter.push).toHaveBeenCalledWith("/streams/add?value=new%20stream");
    });

    test("calls router.back when cancel button is clicked", async () => {
        const user = userEvent.setup();
        renderForm();

        await user.click(screen.getByRole("button", { name: /cancel/i }));

        expect(mockRouter.back).toHaveBeenCalled();
    });
});
