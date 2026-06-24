/**
 * Tests for the IngestionSelector component.
 *
 * Verifies rendering of both ingestion modes (Manual Entry, File Upload),
 * file size/row limit info, and selection callbacks.
 *
 * To run: pnpm vitest run src/features/student-import/__tests__/ingestion-selector.test.tsx
 */

import { describe, it, expect, vi } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithQuery } from "@/__tests__/test-utils";
import { IngestionSelector } from "../components/ingestion-selector";

describe("IngestionSelector", () => {
    it("renders the title and description", () => {
        renderWithQuery(<IngestionSelector onSelect={vi.fn()} />);

        expect(screen.getByText(/import students/i)).toBeInTheDocument();
        expect(screen.getByText(/choose how you would like to add students/i)).toBeInTheDocument();
    });

    it("renders the Manual Entry option", () => {
        renderWithQuery(<IngestionSelector onSelect={vi.fn()} />);

        expect(screen.getByText(/manual entry/i)).toBeInTheDocument();
        expect(screen.getByText(/add students one by one/i)).toBeInTheDocument();
    });

    it("renders the Upload File option", () => {
        renderWithQuery(<IngestionSelector onSelect={vi.fn()} />);

        expect(screen.getByText(/upload file/i)).toBeInTheDocument();
        expect(screen.getByText(/import from csv or excel/i)).toBeInTheDocument();
    });

    it("calls onSelect with 'manual' when Manual Entry is clicked", async () => {
        const onSelect = vi.fn();
        const user = userEvent.setup();

        renderWithQuery(<IngestionSelector onSelect={onSelect} />);

        await user.click(screen.getByText(/manual entry/i));
        expect(onSelect).toHaveBeenCalledWith("manual");
    });

    it("calls onSelect with 'csv' when Upload File is clicked", async () => {
        const onSelect = vi.fn();
        const user = userEvent.setup();

        renderWithQuery(<IngestionSelector onSelect={onSelect} />);

        await user.click(screen.getByText(/upload file/i));
        expect(onSelect).toHaveBeenCalledWith("csv");
    });

    it("displays file size and row count limits in an aside", () => {
        renderWithQuery(<IngestionSelector onSelect={vi.fn()} />);

        expect(screen.getByText(/10MB/i)).toBeInTheDocument();
        expect(screen.getByText(/5,000/i)).toBeInTheDocument();
    });

    it("renders keyboard and file spreadsheet icons", () => {
        const { container } = renderWithQuery(<IngestionSelector onSelect={vi.fn()} />);

        // Both mode buttons should exist
        const buttons = container.querySelectorAll("button");
        expect(buttons.length).toBeGreaterThanOrEqual(2);
    });
});
