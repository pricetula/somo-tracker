/**
 * Tests for the ValidationMotor component (Phase 4).
 *
 * Covers: view toggle (Errors & Warnings / All Records),
 * virtualized row rendering, error/warning/duplicate indicators,
 * editable cells, import-anyway checkbox, stats display, submit/back actions.
 *
 * To run: pnpm vitest run src/features/student-import/__tests__/validation-motor.test.tsx
 */

import { describe, it, expect, vi } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithQuery } from "@/__tests__/test-utils";
import { ValidationMotor } from "../components/validation-motor";
import type { StagedStudentRecord } from "../types";

// Mock TanStack Virtual to render all items in jsdom (no computed layout)
vi.mock("@tanstack/react-virtual", () => ({
    useVirtualizer: (opts: {
        count: number;
        getScrollElement: () => HTMLDivElement | null;
        estimateSize: () => number;
        overscan: number;
    }) => ({
        getVirtualItems: () =>
            Array.from({ length: opts.count }, (_, index) => ({
                index,
                key: index,
                start: index * opts.estimateSize(),
                size: opts.estimateSize(),
                end: (index + 1) * opts.estimateSize(),
                lane: 0,
            })),
        getTotalSize: () => opts.count * opts.estimateSize(),
        measureElement: vi.fn(),
    }),
}));

// ─── Helpers ──────────────────────────────────────────────────────────────

function createRecord(
    index: number,
    overrides: Partial<StagedStudentRecord> = {}
): StagedStudentRecord {
    return {
        _rowIndex: index,
        full_name: `Student ${index}`,
        gender: "M" as const,
        date_of_birth: "2010-03-15",
        upi_number: "KP1234567A",
        knec_assessment_number: "ABC12345",
        cbc_student_parents_id: null,
        class_id: null,
        parent_name_normalized: undefined,
        class_name_normalized: undefined,
        isValid: true,
        isDuplicate: false,
        importAnyway: false,
        errors: {},
        advisories: {},
        ...overrides,
    };
}

function validRecords(count = 3): StagedStudentRecord[] {
    return Array.from({ length: count }, (_, i) => createRecord(i));
}

function mixedRecords(): StagedStudentRecord[] {
    return [
        createRecord(0, { isValid: true }),
        createRecord(1, {
            isValid: false,
            errors: { full_name: "Full name is required" },
        }),
        createRecord(2, {
            isValid: true,
            isDuplicate: true,
            importAnyway: false,
        }),
        createRecord(3, {
            isValid: true,
            isDuplicate: true,
            importAnyway: true, // overridden
        }),
        createRecord(4, {
            isValid: false,
            errors: { gender: "Unrecognized gender value: 'x'" },
            advisories: { date_of_birth: "Ambiguous date — verify" },
        }),
    ];
}

// ─── Tests ────────────────────────────────────────────────────────────────

describe("ValidationMotor", () => {
    // ── View Toggle ─────────────────────────────────────────────────────────

    it("renders Errors & Warnings and All Records toggle buttons", () => {
        renderWithQuery(
            <ValidationMotor
                records={validRecords(3)}
                viewFilter="errors"
                onViewFilterChange={vi.fn()}
                onUpdateRecord={vi.fn()}
                onToggleImportAnyway={vi.fn()}
                errorCount={0}
                duplicateWarningCount={0}
                totalErrorCount={0}
                submitting={false}
                onSubmit={vi.fn()}
                onBack={vi.fn()}
            />
        );

        expect(screen.getByRole("button", { name: /errors.*warnings/i })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: /all records/i })).toBeInTheDocument();
    });

    it("calls onViewFilterChange when toggle is clicked", async () => {
        const onViewFilterChange = vi.fn();
        const user = userEvent.setup();

        renderWithQuery(
            <ValidationMotor
                records={validRecords(3)}
                viewFilter="errors"
                onViewFilterChange={onViewFilterChange}
                onUpdateRecord={vi.fn()}
                onToggleImportAnyway={vi.fn()}
                errorCount={0}
                duplicateWarningCount={0}
                totalErrorCount={0}
                submitting={false}
                onSubmit={vi.fn()}
                onBack={vi.fn()}
            />
        );

        await user.click(screen.getByRole("button", { name: /all records/i }));
        expect(onViewFilterChange).toHaveBeenCalledWith("all");
    });

    it("highlights the active view toggle", () => {
        renderWithQuery(
            <ValidationMotor
                records={validRecords(3)}
                viewFilter="errors"
                onViewFilterChange={vi.fn()}
                onUpdateRecord={vi.fn()}
                onToggleImportAnyway={vi.fn()}
                errorCount={0}
                duplicateWarningCount={0}
                totalErrorCount={0}
                submitting={false}
                onSubmit={vi.fn()}
                onBack={vi.fn()}
            />
        );

        // Errors & Warnings should be active (has bg-background class)
        const errorsTab = screen.getByRole("button", { name: /errors.*warnings/i });
        expect(errorsTab.className).toContain("bg-background");
    });

    // ── Stats Display ───────────────────────────────────────────────────────

    it("displays error count badge when errors exist", () => {
        renderWithQuery(
            <ValidationMotor
                records={mixedRecords()}
                viewFilter="errors"
                onViewFilterChange={vi.fn()}
                onUpdateRecord={vi.fn()}
                onToggleImportAnyway={vi.fn()}
                errorCount={2}
                duplicateWarningCount={1}
                totalErrorCount={3}
                submitting={false}
                onSubmit={vi.fn()}
                onBack={vi.fn()}
            />
        );

        expect(screen.getByText(/2 errors?/i)).toBeInTheDocument();
    });

    it("displays duplicate warning count when duplicates exist", () => {
        renderWithQuery(
            <ValidationMotor
                records={mixedRecords()}
                viewFilter="errors"
                onViewFilterChange={vi.fn()}
                onUpdateRecord={vi.fn()}
                onToggleImportAnyway={vi.fn()}
                errorCount={2}
                duplicateWarningCount={1}
                totalErrorCount={3}
                submitting={false}
                onSubmit={vi.fn()}
                onBack={vi.fn()}
            />
        );

        expect(screen.getByText(/1 duplicate/i)).toBeInTheDocument();
    });

    it("displays total record count", () => {
        renderWithQuery(
            <ValidationMotor
                records={validRecords(5)}
                viewFilter="errors"
                onViewFilterChange={vi.fn()}
                onUpdateRecord={vi.fn()}
                onToggleImportAnyway={vi.fn()}
                errorCount={0}
                duplicateWarningCount={0}
                totalErrorCount={0}
                submitting={false}
                onSubmit={vi.fn()}
                onBack={vi.fn()}
            />
        );

        expect(screen.getByText(/5 records?/i)).toBeInTheDocument();
    });

    // ── Submit Button ──────────────────────────────────────────────────────

    it("renders Submit Import button with record count", () => {
        renderWithQuery(
            <ValidationMotor
                records={validRecords(3)}
                viewFilter="errors"
                onViewFilterChange={vi.fn()}
                onUpdateRecord={vi.fn()}
                onToggleImportAnyway={vi.fn()}
                errorCount={0}
                duplicateWarningCount={0}
                totalErrorCount={0}
                submitting={false}
                onSubmit={vi.fn()}
                onBack={vi.fn()}
            />
        );

        expect(screen.getByRole("button", { name: /submit import/i })).toBeInTheDocument();
        expect(screen.getAllByText(/3/).length).toBeGreaterThan(0);
    });

    it("disables Submit Import when there are errors", () => {
        renderWithQuery(
            <ValidationMotor
                records={mixedRecords()}
                viewFilter="errors"
                onViewFilterChange={vi.fn()}
                onUpdateRecord={vi.fn()}
                onToggleImportAnyway={vi.fn()}
                errorCount={2}
                duplicateWarningCount={1}
                totalErrorCount={3}
                submitting={false}
                onSubmit={vi.fn()}
                onBack={vi.fn()}
            />
        );

        const submitButton = screen.getByRole("button", { name: /submit import/i });
        expect(submitButton).toBeDisabled();
    });

    it("enables Submit Import when error count is zero", () => {
        renderWithQuery(
            <ValidationMotor
                records={validRecords(3)}
                viewFilter="errors"
                onViewFilterChange={vi.fn()}
                onUpdateRecord={vi.fn()}
                onToggleImportAnyway={vi.fn()}
                errorCount={0}
                duplicateWarningCount={0}
                totalErrorCount={0}
                submitting={false}
                onSubmit={vi.fn()}
                onBack={vi.fn()}
            />
        );

        const submitButton = screen.getByRole("button", { name: /submit import/i });
        expect(submitButton).not.toBeDisabled();
    });

    it("disables Submit Import while submitting", () => {
        renderWithQuery(
            <ValidationMotor
                records={validRecords(3)}
                viewFilter="errors"
                onViewFilterChange={vi.fn()}
                onUpdateRecord={vi.fn()}
                onToggleImportAnyway={vi.fn()}
                errorCount={0}
                duplicateWarningCount={0}
                totalErrorCount={0}
                submitting={true}
                onSubmit={vi.fn()}
                onBack={vi.fn()}
            />
        );

        const submitButton = screen.getByRole("button", {
            name: /submitting/i,
        });
        expect(submitButton).toBeDisabled();
    });

    it("calls onSubmit when Submit Import is clicked", async () => {
        const onSubmit = vi.fn();
        const user = userEvent.setup();

        renderWithQuery(
            <ValidationMotor
                records={validRecords(3)}
                viewFilter="errors"
                onViewFilterChange={vi.fn()}
                onUpdateRecord={vi.fn()}
                onToggleImportAnyway={vi.fn()}
                errorCount={0}
                duplicateWarningCount={0}
                totalErrorCount={0}
                submitting={false}
                onSubmit={onSubmit}
                onBack={vi.fn()}
            />
        );

        await user.click(screen.getByRole("button", { name: /submit import/i }));
        expect(onSubmit).toHaveBeenCalledTimes(1);
    });

    // ── Back Button ────────────────────────────────────────────────────────

    it("renders a Back button", () => {
        renderWithQuery(
            <ValidationMotor
                records={validRecords(3)}
                viewFilter="errors"
                onViewFilterChange={vi.fn()}
                onUpdateRecord={vi.fn()}
                onToggleImportAnyway={vi.fn()}
                errorCount={0}
                duplicateWarningCount={0}
                totalErrorCount={0}
                submitting={false}
                onSubmit={vi.fn()}
                onBack={vi.fn()}
            />
        );

        expect(screen.getByRole("button", { name: /back/i })).toBeInTheDocument();
    });

    it("calls onBack when Back is clicked", async () => {
        const onBack = vi.fn();
        const user = userEvent.setup();

        renderWithQuery(
            <ValidationMotor
                records={validRecords(3)}
                viewFilter="errors"
                onViewFilterChange={vi.fn()}
                onUpdateRecord={vi.fn()}
                onToggleImportAnyway={vi.fn()}
                errorCount={0}
                duplicateWarningCount={0}
                totalErrorCount={0}
                submitting={false}
                onSubmit={vi.fn()}
                onBack={onBack}
            />
        );

        await user.click(screen.getByRole("button", { name: /back/i }));
        expect(onBack).toHaveBeenCalledTimes(1);
    });

    // ── Duplicate Handling ─────────────────────────────────────────────────

    it("renders import-anyway checkbox for duplicate rows", () => {
        renderWithQuery(
            <ValidationMotor
                records={[createRecord(0, { isDuplicate: true, importAnyway: false })]}
                viewFilter="all"
                onViewFilterChange={vi.fn()}
                onUpdateRecord={vi.fn()}
                onToggleImportAnyway={vi.fn()}
                errorCount={0}
                duplicateWarningCount={1}
                totalErrorCount={1}
                submitting={false}
                onSubmit={vi.fn()}
                onBack={vi.fn()}
            />
        );

        expect(screen.getByText(/possible duplicate/i)).toBeInTheDocument();
        // "import anyway" is split by <br> in the label
        expect(
            screen.getByText((content) => content.includes("Import") && content.includes("anyway"))
        ).toBeInTheDocument();
    });

    it("calls onToggleImportAnyway when checkbox is clicked", async () => {
        const onToggleImportAnyway = vi.fn();
        const user = userEvent.setup();

        renderWithQuery(
            <ValidationMotor
                records={[createRecord(0, { isDuplicate: true, importAnyway: false })]}
                viewFilter="all"
                onViewFilterChange={vi.fn()}
                onUpdateRecord={vi.fn()}
                onToggleImportAnyway={onToggleImportAnyway}
                errorCount={0}
                duplicateWarningCount={1}
                totalErrorCount={1}
                submitting={false}
                onSubmit={vi.fn()}
                onBack={vi.fn()}
            />
        );

        // Find and click the checkbox
        const checkbox = screen.getByRole("checkbox");
        await user.click(checkbox);
        expect(onToggleImportAnyway).toHaveBeenCalledWith(0);
    });

    it("does NOT render import-anyway checkbox for non-duplicate records", () => {
        renderWithQuery(
            <ValidationMotor
                records={[createRecord(0)]}
                viewFilter="all"
                onViewFilterChange={vi.fn()}
                onUpdateRecord={vi.fn()}
                onToggleImportAnyway={vi.fn()}
                errorCount={0}
                duplicateWarningCount={0}
                totalErrorCount={0}
                submitting={false}
                onSubmit={vi.fn()}
                onBack={vi.fn()}
            />
        );

        expect(screen.queryByText(/possible duplicate/i)).not.toBeInTheDocument();
        expect(screen.queryByText(/import anyway/i)).not.toBeInTheDocument();
    });

    // ── Inline Advisories ──────────────────────────────────────────────────

    it("shows inline advisory text for parent not found", () => {
        renderWithQuery(
            <ValidationMotor
                records={[
                    createRecord(0, {
                        advisories: { parent: "Parent not found in system: 'Unknown'" },
                    }),
                ]}
                viewFilter="all"
                onViewFilterChange={vi.fn()}
                onUpdateRecord={vi.fn()}
                onToggleImportAnyway={vi.fn()}
                errorCount={0}
                duplicateWarningCount={0}
                totalErrorCount={0}
                submitting={false}
                onSubmit={vi.fn()}
                onBack={vi.fn()}
            />
        );

        expect(screen.getByText(/Parent not found in system/i)).toBeInTheDocument();
    });

    it("shows inline advisory text for class not found", () => {
        renderWithQuery(
            <ValidationMotor
                records={[
                    createRecord(0, {
                        advisories: { class: "Class not found in system: 'Unknown'" },
                    }),
                ]}
                viewFilter="all"
                onViewFilterChange={vi.fn()}
                onUpdateRecord={vi.fn()}
                onToggleImportAnyway={vi.fn()}
                errorCount={0}
                duplicateWarningCount={0}
                totalErrorCount={0}
                submitting={false}
                onSubmit={vi.fn()}
                onBack={vi.fn()}
            />
        );

        expect(screen.getByText(/Class not found in system/i)).toBeInTheDocument();
    });

    it("shows inline advisory for ambiguous date", () => {
        renderWithQuery(
            <ValidationMotor
                records={[
                    createRecord(0, {
                        advisories: {
                            date_of_birth: "Ambiguous date — assumed DD/MM/YYYY",
                        },
                    }),
                ]}
                viewFilter="all"
                onViewFilterChange={vi.fn()}
                onUpdateRecord={vi.fn()}
                onToggleImportAnyway={vi.fn()}
                errorCount={0}
                duplicateWarningCount={0}
                totalErrorCount={0}
                submitting={false}
                onSubmit={vi.fn()}
                onBack={vi.fn()}
            />
        );

        expect(screen.getByText(/ambiguous date/i)).toBeInTheDocument();
    });

    // ── Column Headers ─────────────────────────────────────────────────────

    it("renders all column headers in validation view", () => {
        renderWithQuery(
            <ValidationMotor
                records={validRecords(1)}
                viewFilter="all"
                onViewFilterChange={vi.fn()}
                onUpdateRecord={vi.fn()}
                onToggleImportAnyway={vi.fn()}
                errorCount={0}
                duplicateWarningCount={0}
                totalErrorCount={0}
                submitting={false}
                onSubmit={vi.fn()}
                onBack={vi.fn()}
            />
        );

        expect(screen.getByText(/^name$/i)).toBeInTheDocument();
        expect(screen.getByText(/^gender$/i)).toBeInTheDocument();
        expect(screen.getByText(/^dob$/i)).toBeInTheDocument();
        expect(screen.getByText(/^upi$/i)).toBeInTheDocument();
        expect(screen.getByText(/^knec$/i)).toBeInTheDocument();
        expect(screen.getByText(/^parent$/i)).toBeInTheDocument();
        expect(screen.getByText(/^class$/i)).toBeInTheDocument();
    });
});
