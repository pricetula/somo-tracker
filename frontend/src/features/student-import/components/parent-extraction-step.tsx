/**
 * Wizard Step 6: Parent Name Extraction.
 *
 * Multi-column concatenation similar to Name step.
 * On confirm, normalizes and looks up in Parents Map.
 */

"use client";

import * as React from "react";
import { ColumnWizardStep, type SelectSlotConfig } from "./column-wizard-step";

interface ParentExtractionStepProps {
    headers: string[];
    previewRow: Record<string, string>;
    selectedColumns: string[];
    onColumnsChange: (columns: string[]) => void;
    claimedColumns: Set<string>;
    onConfirm: () => void;
    onSkip: () => void;
    onBack?: () => void;
}

export function ParentExtractionStep({
    headers,
    previewRow,
    selectedColumns,
    onColumnsChange,
    claimedColumns,
    onConfirm,
    onSkip,
    onBack,
}: ParentExtractionStepProps) {
    const options = React.useMemo(() => headers.map((h) => ({ value: h, label: h })), [headers]);

    const previewValue = React.useMemo(() => {
        const parts = selectedColumns
            .map((col) => previewRow[col] ?? "")
            .filter(Boolean)
            .map((s) => s.trim());
        return parts.join(" ");
    }, [selectedColumns, previewRow]);

    const selectSlots: SelectSlotConfig[] = selectedColumns.map((col, index) => ({
        id: `parent-col-${index}`,
        label: index === 0 ? "Parent name column" : `Additional column ${index}`,
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
            explainer="Select the column(s) containing the parent/guardian name. The system will attempt to match the name against existing parent records using normalized lookup. If no match is found, parent linking will be skipped for that student (non-blocking)."
            previewValue={previewValue}
            placeholder="Jane Doe"
            selectSlots={selectSlots}
            claimedColumns={claimedColumns}
            onConfirm={onConfirm}
            onBack={onBack}
            onSkip={onSkip}
            showSkip
        />
    );
}
