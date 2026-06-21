/**
 * Tests for the LoginPage component with mocked backend API requests.
 *
 * To run: pnpm vitest run src/features/auth/__tests__/login-page.test.tsx
 */

import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { TooltipProvider } from "@/components/ui/tooltip";
import { renderWithQuery } from "@/__tests__/test-utils";

// ─── Mocks ────────────────────────────────────────────────────────────────

vi.mock("sonner");

const mockDiscover = vi.fn();

vi.mock("@/lib/api/auth", () => ({
    discover: (...args: unknown[]) => mockDiscover(...args),
    register: vi.fn(),
    verifyToken: vi.fn(),
    getMe: vi.fn(),
    logout: vi.fn(),
    isApiError: (err: unknown) => err instanceof Error && "status" in err && "body" in err,
    getApiErrorMessage: (err: unknown) => {
        if (err instanceof Error) return err.message;
        return "An unexpected error occurred";
    },
    ApiRequestError: class extends Error {
        status: number;
        body: { error: string; message?: string };
        constructor(status: number, body: { error: string; message?: string }) {
            super(body.message ?? body.error);
            this.name = "ApiRequestError";
            this.status = status;
            this.body = body;
        }
    },
}));

import { LoginPage } from "@/features/auth/components/login-page";

// ─── Helpers ──────────────────────────────────────────────────────────────

function renderLoginPage() {
    return renderWithQuery(<LoginPage />);
}

// ─── Tests ────────────────────────────────────────────────────────────────

describe("LoginPage", () => {
    beforeEach(() => {
        mockDiscover.mockReset();
    });

    it("renders the login form with all required elements", () => {
        renderLoginPage();

        expect(screen.getByText("Welcome")).toBeInTheDocument();
        expect(screen.getByText(/enter your email to sign in or register/i)).toBeInTheDocument();
        expect(screen.getByLabelText(/email/i)).toBeInTheDocument();
        expect(screen.getByRole("button", { name: /send magic link/i })).toBeInTheDocument();
    });

    it("shows a validation error when submitting with an empty email", async () => {
        const user = userEvent.setup();
        renderLoginPage();

        await user.click(screen.getByRole("button", { name: /send magic link/i }));

        await waitFor(() => {
            const messages = document.querySelectorAll('[data-slot="form-message"]');
            expect(messages.length).toBeGreaterThan(0);
        });
        expect(mockDiscover).not.toHaveBeenCalled();
    });

    it("calls the discover API when submitting a valid email", async () => {
        mockDiscover.mockResolvedValueOnce(undefined);

        const user = userEvent.setup();
        renderLoginPage();

        await user.type(screen.getByLabelText(/email/i), "teacher@school.edu");
        await user.click(screen.getByRole("button", { name: /send magic link/i }));

        await waitFor(() => {
            expect(mockDiscover).toHaveBeenCalledTimes(1);
            expect(mockDiscover).toHaveBeenCalledWith("teacher@school.edu");
        });
    });

    it("disables the submit button and shows a spinner while the request is in flight", async () => {
        mockDiscover.mockReturnValueOnce(new Promise<never>(() => {}));

        const user = userEvent.setup();
        renderLoginPage();

        await user.type(screen.getByLabelText(/email/i), "teacher@school.edu");
        const submitButton = screen.getByRole("button", { name: /send magic link/i });
        await user.click(submitButton);

        await waitFor(() => {
            expect(submitButton).toBeDisabled();
        });

        const spinner = document.querySelector(".animate-spin");
        expect(spinner).toBeInTheDocument();
    });

    it("re-enables the button when the API call fails", async () => {
        mockDiscover.mockRejectedValueOnce(new Error("Network error"));

        const user = userEvent.setup();
        renderLoginPage();

        await user.type(screen.getByLabelText(/email/i), "teacher@school.edu");
        const submitButton = screen.getByRole("button", { name: /send magic link/i });
        await user.click(submitButton);

        await waitFor(() => {
            expect(mockDiscover).toHaveBeenCalledTimes(1);
        });

        // Button should be re-enabled after error
        expect(submitButton).not.toBeDisabled();
    });

    it("renders the footer text when a tooltipSummary is provided", () => {
        renderWithQuery(
            <TooltipProvider>
                <LoginPage tooltipSummary="Auth help text" />
            </TooltipProvider>
        );

        expect(screen.getByText(/no password needed/i)).toBeInTheDocument();
    });
});
