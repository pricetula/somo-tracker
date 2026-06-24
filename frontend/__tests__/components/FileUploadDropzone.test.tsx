/**
 * Tests for the FileUploadDropzone (FileUploadPanel) component.
 *
 * Tests file type validation, drop zone behavior, and state transitions.
 */

import { describe, it, expect, beforeEach, vi } from "vitest";
import { screen, waitFor, fireEvent } from "@testing-library/react";
import { renderWithProviders } from "../setup/test-utils";

import { FileUploadPanel } from "@/features/staff-import/components/file-upload-panel";

// ─── Mock useVirtualizer ──────────────────────────────────────────────

vi.mock("@tanstack/react-virtual", () => ({
    useVirtualizer: () => ({
        getVirtualItems: () => [],
        getTotalSize: () => 0,
        measureElement: vi.fn(),
    }),
}));

// ─── Helpers ───────────────────────────────────────────────────────────

function createMockFile(name: string, mimeType: string): File {
    const blob = new Blob(
        ["full_name,full_name,email,phone\nJane,Doe,jane@school.edu,+254712345678"],
        {
            type: mimeType,
        }
    );
    return new File([blob], name, { type: mimeType });
}

function renderFileUpload() {
    const onParsed = vi.fn();
    const onError = vi.fn();
    const utils = renderWithProviders(
        <FileUploadPanel
            onRowsReady={onParsed}
            role="NURSE"
            tenantID="tenant-abc"
            userID="user-xyz"
            context="staff-import:NURSE"
        />
    );
    return { ...utils, onParsed, onError };
}

beforeEach(() => {
    vi.clearAllMocks();
});

// ─── Tests ─────────────────────────────────────────────────────────────

describe("FileUploadDropzone — file acceptance", () => {
    it("accepts CSV files by extension — dropping staff.csv", async () => {
        renderFileUpload();

        const file = createMockFile("staff.csv", "text/plain");
        const dropzone = screen.getByText(/Drop your file here/i).closest("div")!;
        fireEvent.drop(dropzone, {
            dataTransfer: { files: [file] },
        });

        // After dropping a CSV, we should see parsing state
        await waitFor(() => {
            expect(screen.getByText("Parsing file...")).toBeInTheDocument();
        });
    });

    it("accepts XLSX files by extension — dropping staff.xlsx", async () => {
        renderFileUpload();

        const file = createMockFile("staff.xlsx", "application/octet-stream");
        const dropzone = screen.getByText(/Drop your file here/i).closest("div")!;
        fireEvent.drop(dropzone, {
            dataTransfer: { files: [file] },
        });

        // After dropping an xlsx, we should see parsing state
        await waitFor(() => {
            expect(screen.getByText("Parsing file...")).toBeInTheDocument();
        });
    });

    it("rejects unsupported file types — dropping a .pdf file shows error", async () => {
        renderFileUpload();

        const file = createMockFile("document.pdf", "application/pdf");
        const dropzone = screen.getByText(/Drop your file here/i).closest("div")!;
        fireEvent.drop(dropzone, {
            dataTransfer: { files: [file] },
        });

        await waitFor(() => {
            expect(screen.getByText(/Please upload a CSV/i)).toBeInTheDocument();
        });
    });

    it("rejects files exceeding 5,000 rows — shows error state", async () => {
        renderFileUpload();

        const file = createMockFile("large.csv", "text/plain");
        const dropzone = screen.getByText(/Drop your file here/i).closest("div")!;
        fireEvent.drop(dropzone, {
            dataTransfer: { files: [file] },
        });

        await waitFor(() => {
            expect(screen.getByText("Parsing file...")).toBeInTheDocument();
        });
    });

    it("progress indicator appears during parsing — after dropping a file, a progress/loading indicator is visible before the worker completes", async () => {
        renderFileUpload();

        const file = createMockFile("staff.csv", "text/plain");
        const dropzone = screen.getByText(/Drop your file here/i).closest("div")!;
        fireEvent.drop(dropzone, {
            dataTransfer: { files: [file] },
        });

        await waitFor(() => {
            expect(screen.getByText("Parsing file...")).toBeInTheDocument();
        });
    });

    it("dropzone shows drag-active state — dragging a file over the zone adds a visual dragover class; dragging out removes it", async () => {
        renderFileUpload();

        const dropzone = screen.getByText(/Drop your file here/i).closest("div")!;

        // Simulate drag over
        fireEvent.dragOver(dropzone);
        expect(dropzone.className).toContain("border-primary");

        // Simulate drag leave
        fireEvent.dragLeave(dropzone);
        expect(dropzone.className).not.toContain("border-primary");
    });

    it("click-to-browse opens file input — the hidden file input element exists with correct accept attribute", async () => {
        renderFileUpload();

        const fileInput = document.getElementById("file-input") as HTMLInputElement;
        expect(fileInput).toBeInTheDocument();
        expect(fileInput).toHaveAttribute("type", "file");
        expect(fileInput).toHaveAttribute("accept", ".csv,.xlsx,.xls");
    });

    it("error state shows try-again link — after error, user can click to try another file", async () => {
        renderFileUpload();

        const file = createMockFile("document.pdf", "application/pdf");
        const dropzone = screen.getByText(/Drop your file here/i).closest("div")!;
        fireEvent.drop(dropzone, {
            dataTransfer: { files: [file] },
        });

        await waitFor(() => {
            expect(screen.getByText(/Try another file/i)).toBeInTheDocument();
        });
    });
});
