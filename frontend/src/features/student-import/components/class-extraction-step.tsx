/**
 * Wizard Step 7: Class Name Extraction.
 *
 * Multi-column concatenation (e.g., Grade + Stream → "Grade 4 West").
 * On confirm, normalizes with prefix-safe algorithm and looks up in Classes Map.
 */

"use client";

import * as React from "react";
import { ColumnWizardStep, type SelectSlotConfig } from "./column-wizard-step";

interface ClassExtractionStepProps {
    headers: string[];
    previewRow: Record<string, string>;
    selectedColumns: string[];
    onColumnsChange: (columns: string[]) => void;
    claimedColumns: Set<string>;
    onConfirm: () => void;
    onSkip: () => void;
    onBack?: () => void;
}

export function ClassExtractionStep({
    headers,
    previewRow,
    selectedColumns,
    onColumnsChange,
    claimedColumns,
    onConfirm,
    onSkip,
    onBack,
}: ClassExtractionStepProps) {
    const options = React.useMemo(() => headers.map((h) => ({ value: h, label: h })), [headers]);

    const previewValue = React.useMemo(() => {
        const parts = selectedColumns
            .map((col) => previewRow[col] ?? "")
            .filter(Boolean)
            .map((s) => s.trim());
        return parts.join(" ");
    }, [selectedColumns, previewRow]);

    const selectSlots: SelectSlotConfig[] = selectedColumns.map((col, index) => ({
        id: `class-col-${index}`,
        label: index === 0 ? "Class column" : `Additional column ${index}`,
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

    return (
        <ColumnWizardStep
            explainer="Select the column(s) that identify the student's class. For split grade/stream columns, use (+) to merge them (e.g., Grade column + Stream column). The system will normalize class names for matching against existing classes (e.g., 'Class 4 West' → '4west'). If no match is found, class assignment will be skipped (non-blocking)."
            previewValue={previewValue}
            placeholder="Class 1 North"
            selectSlots={selectSlots}
            claimedColumns={claimedColumns}
            onConfirm={onConfirm}
            onBack={onBack}
            onSkip={onSkip}
            showSkip
        />
    );
}
