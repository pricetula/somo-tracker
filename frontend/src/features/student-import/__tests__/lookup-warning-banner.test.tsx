/**
 * Tests for the LookupWarningBanner component.
 *
 * Verifies rendering for "parents" and "classes" types,
 * error message display, and retry interaction.
 *
 * To run: pnpm vitest run src/features/student-import/__tests__/lookup-warning-banner.test.tsx
 */

import { describe, it, expect, vi } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithQuery } from "@/__tests__/test-utils";
import { LookupWarningBanner } from "../components/lookup-warning-banner";

describe("LookupWarningBanner", () => {
    describe("parents type", () => {
        it('renders "Parent linking unavailable" label', () => {
            renderWithQuery(
                <LookupWarningBanner type="parents" message="Network error" onRetry={vi.fn()} />
            );

            expect(screen.getByText(/parent linking unavailable/i)).toBeInTheDocument();
        });

        it("displays the error message", () => {
            renderWithQuery(
                <LookupWarningBanner
                    type="parents"
                    message="Failed to fetch parent records"
                    onRetry={vi.fn()}
                />
            );

            expect(screen.getByText(/failed to fetch parent records/i)).toBeInTheDocument();
        });

        it("renders a Retry Lookup button", () => {
            renderWithQuery(
                <LookupWarningBanner type="parents" message="Network error" onRetry={vi.fn()} />
            );

            expect(screen.getByRole("button", { name: /retry lookup/i })).toBeInTheDocument();
        });
    });

    describe("classes type", () => {
        it('renders "Class linking unavailable" label', () => {
            renderWithQuery(
                <LookupWarningBanner type="classes" message="Server error" onRetry={vi.fn()} />
            );

            expect(screen.getByText(/class linking unavailable/i)).toBeInTheDocument();
        });

        it("displays the error message", () => {
            renderWithQuery(
                <LookupWarningBanner
                    type="classes"
                    message="Timeout fetching classes"
                    onRetry={vi.fn()}
                />
            );

            expect(screen.getByText(/timeout fetching classes/i)).toBeInTheDocument();
        });

        it("renders a Retry Lookup button", () => {
            renderWithQuery(
                <LookupWarningBanner type="classes" message="Server error" onRetry={vi.fn()} />
            );

            expect(screen.getByRole("button", { name: /retry lookup/i })).toBeInTheDocument();
        });
    });

    describe("interactions", () => {
        it("calls onRetry when Retry Lookup is clicked", async () => {
            const onRetry = vi.fn();
            const user = userEvent.setup();

            renderWithQuery(
                <LookupWarningBanner type="parents" message="Failed" onRetry={onRetry} />
            );

            await user.click(screen.getByRole("button", { name: /retry lookup/i }));
            expect(onRetry).toHaveBeenCalledTimes(1);
        });

        it("shows an alert icon in the banner", () => {
            const { container } = renderWithQuery(
                <LookupWarningBanner type="parents" message="Error" onRetry={vi.fn()} />
            );

            // lucide AlertCircle renders an SVG with size-4
            const alertIcon = container.querySelector(".size-4");
            expect(alertIcon).toBeInTheDocument();
        });
    });
});
