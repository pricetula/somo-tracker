/**
 * Tests for BulkStaffImport component behavior.
 *
 * Tests rendering in page mode, role labels, and the entry tabs.
 */

import { describe, it, expect, beforeEach, vi } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders, mockGetMe } from "../setup/test-utils";

import { BulkStaffImport } from "@/features/staff-import/components/bulk-staff-import-dialog";

// ─── Mock useVirtualizer ──────────────────────────────────────────────

vi.mock("@tanstack/react-virtual", () => ({
    useVirtualizer: () => ({
        getVirtualItems: () => [],
        getTotalSize: () => 0,
        measureElement: vi.fn(),
    }),
}));

// ─── Mock IndexedDB draft persistence ─────────────────────────────────
// These tests don't exercise the draft-resume flow, so prevent leaked
// IndexedDB state from flaking the tab assertions.

vi.mock("@/lib/db", () => ({
    loadDraft: vi.fn().mockResolvedValue(null),
    saveDraft: vi.fn().mockResolvedValue(undefined),
    clearDraft: vi.fn().mockResolvedValue(undefined),
}));

// ─── Tests ─────────────────────────────────────────────────────────────

describe("BulkStaffImport — component rendering", () => {
    beforeEach(() => {
        mockGetMe();
        vi.clearAllMocks();
    });

    it("renders in page mode with heading showing the role label (NURSE → Nurses)", async () => {
        renderWithProviders(<BulkStaffImport role="NURSE" mode="page" />);

        await waitFor(() => {
            expect(screen.queryByText("Loading...")).not.toBeInTheDocument();
        });

        expect(screen.getByText(/Invite Nurses/i)).toBeInTheDocument();
    });

    it("renders for FINANCE role with correct label", async () => {
        renderWithProviders(<BulkStaffImport role="FINANCE" mode="page" />);

        await waitFor(() => {
            expect(screen.queryByText("Loading...")).not.toBeInTheDocument();
        });

        expect(screen.getByText(/Invite Finance Staff/i)).toBeInTheDocument();
    });

    it("renders for SCHOOL_ADMIN role with correct label", async () => {
        renderWithProviders(<BulkStaffImport role="SCHOOL_ADMIN" mode="page" />);

        await waitFor(() => {
            expect(screen.queryByText("Loading...")).not.toBeInTheDocument();
        });

        expect(screen.getByText(/Invite School Admins/i)).toBeInTheDocument();
    });

    it("shows loading state initially before session resolves", () => {
        renderWithProviders(<BulkStaffImport role="NURSE" mode="page" />);

        expect(screen.getByText("Loading...")).toBeInTheDocument();
    });
});

describe("BulkStaffImport — entry branch tabs", () => {
    beforeEach(() => {
        mockGetMe();
        vi.clearAllMocks();
    });

    it("'Add Manually' and 'Upload File' tabs are both rendered after session loads", async () => {
        renderWithProviders(<BulkStaffImport role="NURSE" mode="page" />);

        await waitFor(() => {
            expect(screen.queryByText("Loading...")).not.toBeInTheDocument();
        });

        await waitFor(() => {
            // Use getAllByText since TabsTrigger may have nested elements
            const manualTab = screen.getByRole("tab", { name: /add manually/i });
            const uploadTab = screen.getByRole("tab", { name: /upload file/i });
            expect(manualTab).toBeInTheDocument();
            expect(uploadTab).toBeInTheDocument();
        });
    });

    it("default branch shows manual form after session loads", async () => {
        renderWithProviders(<BulkStaffImport role="NURSE" mode="page" />);

        await waitFor(() => {
            expect(screen.queryByText("Loading...")).not.toBeInTheDocument();
        });

        await waitFor(() => {
            expect(screen.getByRole("tab", { name: /add manually/i })).toBeInTheDocument();
        });
    });

    it("switching to Upload tab shows the dropzone", async () => {
        const user = userEvent.setup();
        renderWithProviders(<BulkStaffImport role="NURSE" mode="page" />);

        // Wait for session to resolve and tabs to appear
        await waitFor(() => {
            expect(screen.queryByText("Loading...")).not.toBeInTheDocument();
        });

        await waitFor(() => {
            const uploadTab = screen.queryByRole("tab", { name: /upload file/i });
            if (uploadTab) {
                user.click(uploadTab);
            }
        });

        await waitFor(() => {
            expect(screen.getByText(/Drop your file here/i)).toBeInTheDocument();
        });
    });

    it("switching back to Manual tab shows the form", async () => {
        const user = userEvent.setup();
        renderWithProviders(<BulkStaffImport role="NURSE" mode="page" />);

        await waitFor(() => {
            expect(screen.queryByText("Loading...")).not.toBeInTheDocument();
        });

        // Switch to Upload
        await waitFor(() => {
            const uploadTab = screen.queryByRole("tab", { name: /upload file/i });
            if (uploadTab) {
                user.click(uploadTab);
            }
        });

        await waitFor(() => {
            expect(screen.getByText(/Drop your file here/i)).toBeInTheDocument();
        });

        // Switch back to Manual
        await waitFor(() => {
            const manualTab = screen.queryByRole("tab", { name: /add manually/i });
            if (manualTab) {
                user.click(manualTab);
            }
        });

        // After switching back, the dropzone should be gone
        await waitFor(() => {
            expect(screen.queryByText(/Drop your file here/i)).not.toBeInTheDocument();
        });
    });
});
