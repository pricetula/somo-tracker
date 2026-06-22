/**
 * Tests for the RegisterPage component with mocked backend API requests.
 *
 * To run: pnpm vitest run src/features/auth/__tests__/register-page.test.tsx
 */

import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { TooltipProvider } from "@/components/ui/tooltip";
import { renderWithQuery } from "@/__tests__/test-utils";

// ─── Mocks ────────────────────────────────────────────────────────────────

vi.mock("sonner");

const mockPush = vi.fn();
const mockReplace = vi.fn();

/** Mutable ref so the mock's useSearchParams always reads the latest value. */
const searchParamsRef: { current: URLSearchParams } = {
    current: new URLSearchParams(),
};

vi.mock("next/navigation", () => ({
    useRouter: () => ({
        push: mockPush,
        replace: mockReplace,
        back: vi.fn(),
        forward: vi.fn(),
        refresh: vi.fn(),
        prefetch: vi.fn(),
    }),
    useSearchParams: () => searchParamsRef.current,
}));

import { ApiError } from "@/lib/api/client";

const mockRegisterFn = vi.fn();

function createApiError(status: number, body: { error: string; message?: string }) {
    return new ApiError(status, body.error, body.message ?? body.error);
}

vi.mock("@/lib/api/auth", () => ({
    discover: vi.fn(),
    register: (...args: unknown[]) => mockRegisterFn(...args),
    verifyToken: vi.fn(),
    getMe: vi.fn(),
    logout: vi.fn(),
    isApiError: (err: unknown) =>
        typeof err === "object" && err !== null && "status" in err && "body" in err,
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

import { RegisterPage } from "@/features/auth/components/register-page";

// ─── Helpers ──────────────────────────────────────────────────────────────

function renderRegisterPage() {
    return renderWithQuery(<RegisterPage />);
}

function setSessionRef(ref: string) {
    searchParamsRef.current = new URLSearchParams({ session_ref: ref });
}

function clearSessionRef() {
    searchParamsRef.current = new URLSearchParams();
}

function getSchoolNameInput() {
    return screen.getByPlaceholderText(/e\.g\. Lincoln High School/i);
}

function getFirstNameInput() {
    return screen.getByPlaceholderText(/jane/i);
}

function getLastNameInput() {
    return screen.getByPlaceholderText(/doe/i);
}

// ─── Tests ────────────────────────────────────────────────────────────────

describe("RegisterPage", () => {
    beforeEach(() => {
        mockRegisterFn.mockReset();
        mockPush.mockReset();
        mockReplace.mockReset();
        clearSessionRef();
    });

    describe("when no session_ref is present", () => {
        it("redirects to /login", async () => {
            renderRegisterPage();

            await waitFor(() => {
                expect(mockReplace).toHaveBeenCalledWith("/login");
            });
        });

        it("does not render the registration form", () => {
            renderRegisterPage();

            expect(screen.queryByText(/create your school account/i)).not.toBeInTheDocument();
        });
    });

    describe("when session_ref is present", () => {
        beforeEach(() => {
            setSessionRef("ref_abc123");
        });

        it("renders the registration form with all required fields", () => {
            renderRegisterPage();

            expect(screen.getByText(/create your school account/i)).toBeInTheDocument();
            expect(
                screen.getByText(/set up your school or educational organization/i)
            ).toBeInTheDocument();
            expect(getSchoolNameInput()).toBeInTheDocument();
            expect(getFirstNameInput()).toBeInTheDocument();
            expect(getLastNameInput()).toBeInTheDocument();
            expect(screen.getByRole("button", { name: /create account/i })).toBeInTheDocument();
        });

        it("does not redirect to /login when session_ref is present", () => {
            renderRegisterPage();

            expect(mockReplace).not.toHaveBeenCalled();
        });

        it("shows validation errors when submitting an empty form", async () => {
            const user = userEvent.setup();
            renderRegisterPage();

            await user.click(screen.getByRole("button", { name: /create account/i }));

            expect(
                await screen.findByText(/school name must be at least 2 characters/i)
            ).toBeInTheDocument();
            expect(screen.getByText(/first name is required/i)).toBeInTheDocument();
            expect(screen.getByText(/last name is required/i)).toBeInTheDocument();
            expect(mockRegisterFn).not.toHaveBeenCalled();
        });

        it("shows a validation error when school name is too short", async () => {
            const user = userEvent.setup();
            renderRegisterPage();

            await user.type(getSchoolNameInput(), "A");
            await user.type(getFirstNameInput(), "Jane");
            await user.type(getLastNameInput(), "Doe");

            await user.click(screen.getByRole("button", { name: /create account/i }));

            expect(
                await screen.findByText(/school name must be at least 2 characters/i)
            ).toBeInTheDocument();
            expect(mockRegisterFn).not.toHaveBeenCalled();
        });

        it("calls the register API with valid form data", async () => {
            mockRegisterFn.mockResolvedValueOnce(undefined);

            const user = userEvent.setup();
            renderRegisterPage();

            await user.type(getSchoolNameInput(), "Lincoln High School");
            await user.type(getFirstNameInput(), "Jane");
            await user.type(getLastNameInput(), "Doe");

            await user.click(screen.getByRole("button", { name: /create account/i }));

            await waitFor(() => {
                expect(mockRegisterFn).toHaveBeenCalledTimes(1);
                expect(mockRegisterFn).toHaveBeenCalledWith({
                    school_name: "Lincoln High School",
                    session_ref: "ref_abc123",
                    first_name: "Jane",
                    last_name: "Doe",
                });
            });
        });

        it("disables the submit button and shows a spinner while submitting", async () => {
            mockRegisterFn.mockReturnValueOnce(new Promise<never>(() => {}));

            const user = userEvent.setup();
            renderRegisterPage();

            await user.type(getSchoolNameInput(), "Lincoln High School");
            await user.type(getFirstNameInput(), "Jane");
            await user.type(getLastNameInput(), "Doe");

            const submitButton = screen.getByRole("button", { name: /create account/i });
            await user.click(submitButton);

            await waitFor(() => {
                expect(submitButton).toBeDisabled();
            });

            const spinner = document.querySelector(".animate-spin");
            expect(spinner).toBeInTheDocument();
        });

        it("navigates to / on successful registration", async () => {
            mockRegisterFn.mockResolvedValueOnce(undefined);

            const user = userEvent.setup();
            renderRegisterPage();

            await user.type(getSchoolNameInput(), "Lincoln High School");
            await user.type(getFirstNameInput(), "Jane");
            await user.type(getLastNameInput(), "Doe");

            await user.click(screen.getByRole("button", { name: /create account/i }));

            await waitFor(() => {
                expect(mockPush).toHaveBeenCalledWith("/");
            });
        });

        it("re-enables the button when registration fails with a generic error", async () => {
            mockRegisterFn.mockRejectedValueOnce(new Error("Email already in use"));

            const user = userEvent.setup();
            renderRegisterPage();

            await user.type(getSchoolNameInput(), "Lincoln High School");
            await user.type(getFirstNameInput(), "Jane");
            await user.type(getLastNameInput(), "Doe");

            const submitButton = screen.getByRole("button", { name: /create account/i });
            await user.click(submitButton);

            await waitFor(() => {
                expect(mockRegisterFn).toHaveBeenCalledTimes(1);
            });

            expect(submitButton).not.toBeDisabled();
        });

        it("redirects to /login on 401 (expired session_ref)", async () => {
            const expiredError = createApiError(401, {
                error: "session_expired",
                message: "Session has expired",
            });
            mockRegisterFn.mockRejectedValueOnce(expiredError);

            const user = userEvent.setup();
            renderRegisterPage();

            await user.type(getSchoolNameInput(), "Lincoln High School");
            await user.type(getFirstNameInput(), "Jane");
            await user.type(getLastNameInput(), "Doe");

            await user.click(screen.getByRole("button", { name: /create account/i }));

            await waitFor(() => {
                expect(mockReplace).toHaveBeenCalledWith("/login");
            });
        });
    });

    describe("when tooltipSummary is provided", () => {
        beforeEach(() => {
            setSessionRef("ref_abc123");
        });

        it("renders the card description text", () => {
            renderWithQuery(
                <TooltipProvider>
                    <RegisterPage tooltipSummary="Registration help" />
                </TooltipProvider>
            );

            expect(
                screen.getByText(/set up your school or educational organization/i)
            ).toBeInTheDocument();
        });
    });
});
