/**
 * Wizard Step 1: Student Name Extraction.
 *
 * Allows selecting one or more columns that assemble the full student name.
 * Multi-column selection concatenated with spaces.
 */

"use client";

import * as React from "react";
import { ColumnWizardStep, type SelectSlotConfig } from "./column-wizard-step";

interface NameExtractionStepProps {
    headers: string[];
    previewRow: Record<string, string>;
    selectedColumns: string[];
    onColumnsChange: (columns: string[]) => void;
    claimedColumns: Set<string>;
    onConfirm: () => void;
    onBack?: () => void;
}

export function NameExtractionStep({
    headers,
    previewRow,
    selectedColumns,
    onColumnsChange,
    claimedColumns,
    onConfirm,
    onBack,
}: NameExtractionStepProps) {
    const options = React.useMemo(() => headers.map((h) => ({ value: h, label: h })), [headers]);

    // Build preview value
    const previewValue = React.useMemo(() => {
        const parts = selectedColumns
            .map((col) => previewRow[col] ?? "")
            .filter(Boolean)
            .map((s) => s.trim());
        return parts.join(" ");
    }, [selectedColumns, previewRow]);

    // Auto-detect if a column looks like "full_name" or "name"
    React.useEffect(() => {
        if (selectedColumns.length === 0) {
            const nameCol = headers.find(
                (h) =>
                    h.toLowerCase().replace(/\s+/g, "") === "fullname" ||
                    h.toLowerCase().replace(/\s+/g, "") === "name"
            );
            if (nameCol) {
                onColumnsChange([nameCol]);
            }
        }
    }, [headers, selectedColumns, onColumnsChange]);

    const selectSlots: SelectSlotConfig[] = selectedColumns.map((col, index) => ({
        id: `name-col-${index}`,
        label: index === 0 ? "Name column" : `Additional column ${index}`,
        value: col,
        options,
        onChange: (v) => {
            const updated = [...selectedColumns];
            updated[index] = v;
            onColumnsChange(updated);
        },
        canRemove: selectedColumns.length > 1,
        onRemove: () => {
            const updated = selectedColumns.filter((_, i) => i !== index);
            onColumnsChange(updated);
        },
        showAdd: index === selectedColumns.length - 1,
        onAdd: () => {
            if (selectedColumns.length < headers.length) {
                onColumnsChange([...selectedColumns, ""]);
            }
        },
    }));

    const hasSelection = selectedColumns.some((c) => c && c.trim());

    return (
        <ColumnWizardStep
            explainer="Select the column(s) that contain the student's full name. Select one column if the full name is in a single field, or combine multiple columns (e.g., First Name + Last Name) using (+)."
            previewValue={previewValue}
            placeholder="John Doe"
            selectSlots={selectSlots}
            claimedColumns={claimedColumns}
            onConfirm={onConfirm}
            onBack={onBack}
            confirmDisabled={!hasSelection}
        />
    );
}
