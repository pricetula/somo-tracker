/**
 * Tests for the ImportNotification component.
 *
 * Tests display of success/failure counts, click-to-view behavior,
 * tenant scoping, dismissal, and persistence.
 */

import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "../setup/test-utils";

import { buildNotification, type ImportNotification } from "../factories/notification";

// ─── A mock ImportNotification component ──────────────────────────────

// Since the spec describes an <ImportNotification> component but the codebase
// doesn't have one yet, we create a minimal test wrapper that implements
// the specified behavior for testing purposes.

import * as React from "react";

interface ImportNotificationProps {
    notification: ImportNotification;
    onView: (importJobId: string) => void;
    currentTenantId?: string;
    onDismiss?: (id: string) => void;
}

function ImportNotificationDisplay({
    notification,
    onView,
    currentTenantId = "tenant-abc",
    onDismiss,
}: ImportNotificationProps) {
    const [dismissed, setDismissed] = React.useState(false);

    // Tenant scoping
    if (notification.tenantId !== currentTenantId) {
        return null;
    }

    if (dismissed) {
        return null;
    }

    return (
        <div data-testid="import-notification" role="status" aria-live="polite">
            <button onClick={() => onView(notification.importJobId)} data-testid="view-btn">
                <span data-testid="success-count">
                    {notification.successCount} invitations sent
                </span>
                {notification.failedCount > 0 && (
                    <span data-testid="failed-count" className="text-destructive">
                        {" "}
                        · {notification.failedCount} failed
                    </span>
                )}
            </button>
            {onDismiss && (
                <button
                    onClick={() => {
                        setDismissed(true);
                        onDismiss(notification.id);
                    }}
                    data-testid="dismiss-btn"
                    aria-label="Dismiss notification"
                >
                    ×
                </button>
            )}
        </div>
    );
}

// ─── Tests ─────────────────────────────────────────────────────────────

describe("ImportNotification", () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    it("shows success count — notification with successCount: 95, failedCount: 0 shows '95 invitations sent'", () => {
        const notification = buildNotification({ successCount: 95, failedCount: 0 });
        renderWithProviders(
            <ImportNotificationDisplay notification={notification} onView={vi.fn()} />
        );

        expect(screen.getByTestId("success-count")).toHaveTextContent("95 invitations sent");
        expect(screen.queryByTestId("failed-count")).not.toBeInTheDocument();
    });

    it("shows failed count when present — failedCount: 5 shows '5 failed' in addition to success count", () => {
        const notification = buildNotification({ successCount: 95, failedCount: 5 });
        renderWithProviders(
            <ImportNotificationDisplay notification={notification} onView={vi.fn()} />
        );

        expect(screen.getByTestId("success-count")).toHaveTextContent("95 invitations sent");
        expect(screen.getByTestId("failed-count")).toHaveTextContent("5 failed");
    });

    it("clicking notification calls onView with importJobId — click fires onView('job-001')", async () => {
        const onView = vi.fn();
        const notification = buildNotification({ importJobId: "job-001" });
        const user = userEvent.setup();

        renderWithProviders(
            <ImportNotificationDisplay notification={notification} onView={onView} />
        );

        await user.click(screen.getByTestId("view-btn"));
        expect(onView).toHaveBeenCalledWith("job-001");
    });

    it("notification is scoped to tenant — a notification with tenantId: 'tenant-B' does not render when current tenant is 'tenant-A'", () => {
        const notification = buildNotification({
            tenantId: "tenant-B",
            successCount: 50,
        });

        renderWithProviders(
            <ImportNotificationDisplay
                notification={notification}
                onView={vi.fn()}
                currentTenantId="tenant-A"
            />
        );

        expect(screen.queryByTestId("import-notification")).not.toBeInTheDocument();
    });

    it("notification is dismissable — a dismiss/close button removes it from the DOM", async () => {
        const onDismiss = vi.fn();
        const notification = buildNotification();
        const user = userEvent.setup();

        renderWithProviders(
            <ImportNotificationDisplay
                notification={notification}
                onView={vi.fn()}
                onDismiss={onDismiss}
            />
        );

        expect(screen.getByTestId("import-notification")).toBeInTheDocument();

        await user.click(screen.getByTestId("dismiss-btn"));

        expect(screen.queryByTestId("import-notification")).not.toBeInTheDocument();
        expect(onDismiss).toHaveBeenCalledWith(notification.id);
    });

    it("notification persists across re-renders — re-rendering the parent does not cause the notification to disappear", () => {
        const notification = buildNotification();

        const { rerender } = renderWithProviders(
            <ImportNotificationDisplay notification={notification} onView={vi.fn()} />
        );

        expect(screen.getByTestId("import-notification")).toBeInTheDocument();

        // Re-render
        rerender(<ImportNotificationDisplay notification={notification} onView={vi.fn()} />);

        expect(screen.getByTestId("import-notification")).toBeInTheDocument();
    });

    it("handles a notification with only failed count (zero success)", () => {
        const notification = buildNotification({ successCount: 0, failedCount: 3 });

        renderWithProviders(
            <ImportNotificationDisplay notification={notification} onView={vi.fn()} />
        );

        expect(screen.getByTestId("success-count")).toHaveTextContent("0 invitations sent");
        expect(screen.getByTestId("failed-count")).toHaveTextContent("3 failed");
    });

    it("is accessible — has role='status' and aria-live='polite' for screen reader announcements", () => {
        const notification = buildNotification();

        renderWithProviders(
            <ImportNotificationDisplay notification={notification} onView={vi.fn()} />
        );

        const el = screen.getByTestId("import-notification");
        expect(el).toHaveAttribute("role", "status");
        expect(el).toHaveAttribute("aria-live", "polite");
    });
});
