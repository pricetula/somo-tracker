/**
 * Wizard Step 2: Gender Extraction.
 *
 * Single column select for gender.
 */

"use client";

import * as React from "react";
import { ColumnWizardStep, type SelectSlotConfig } from "./column-wizard-step";

interface GenderExtractionStepProps {
    headers: string[];
    previewRow: Record<string, string>;
    selectedColumn: string | null;
    onColumnChange: (column: string | null) => void;
    claimedColumns: Set<string>;
    onConfirm: () => void;
    onBack?: () => void;
}

export function GenderExtractionStep({
    headers,
    previewRow,
    selectedColumn,
    onColumnChange,
    claimedColumns,
    onConfirm,
    onBack,
}: GenderExtractionStepProps) {
    const options = React.useMemo(() => headers.map((h) => ({ value: h, label: h })), [headers]);

    // Auto-detect
    React.useEffect(() => {
        if (!selectedColumn) {
            const col = headers.find(
                (h) => h.toLowerCase().replace(/\s+/g, "") === "gender" || h.toLowerCase() === "sex"
            );
            if (col) onColumnChange(col);
        }
    }, [headers, selectedColumn, onColumnChange]);

    const previewValue = selectedColumn ? (previewRow[selectedColumn] ?? "") : "";

    const selectSlots: SelectSlotConfig[] = [
        {
            id: "gender-col",
            label: "Gender column",
            value: selectedColumn ?? "",
            options,
            onChange: (v) => onColumnChange(v),
        },
    ];

    return (
        <ColumnWizardStep
            explainer="Select the column that contains the student's gender. Values will be normalized to M or F automatically."
            previewValue={previewValue}
            placeholder="M / F"
            selectSlots={selectSlots}
            claimedColumns={claimedColumns}
            onConfirm={onConfirm}
            onBack={onBack}
            confirmDisabled={!selectedColumn}
        />
    );
}
