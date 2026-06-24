/**
 * Wizard Step 3: Date of Birth Extraction.
 *
 * Single column select with Skip option.
 */

"use client";

import * as React from "react";
import { ColumnWizardStep, type SelectSlotConfig } from "./column-wizard-step";

interface DobExtractionStepProps {
    headers: string[];
    previewRow: Record<string, string>;
    selectedColumn: string | null;
    onColumnChange: (column: string | null) => void;
    claimedColumns: Set<string>;
    onConfirm: () => void;
    onSkip: () => void;
    onBack?: () => void;
}

export function DobExtractionStep({
    headers,
    previewRow,
    selectedColumn,
    onColumnChange,
    claimedColumns,
    onConfirm,
    onSkip,
    onBack,
}: DobExtractionStepProps) {
    const options = React.useMemo(() => headers.map((h) => ({ value: h, label: h })), [headers]);

    const previewValue = selectedColumn ? (previewRow[selectedColumn] ?? "") : "";

    const selectSlots: SelectSlotConfig[] = [
        {
            id: "dob-col",
            label: "Date of Birth column",
            value: selectedColumn ?? "",
            options,
            onChange: (v) => onColumnChange(v),
        },
    ];

    return (
        <ColumnWizardStep
            explainer="Select the column containing the student's date of birth. Common formats (DD/MM/YYYY, MM/DD/YYYY, YYYY-MM-DD) are supported. Ambiguous dates will be flagged for review."
            previewValue={previewValue}
            placeholder="DD/MM/YYYY"
            selectSlots={selectSlots}
            claimedColumns={claimedColumns}
            onConfirm={onConfirm}
            onBack={onBack}
            onSkip={onSkip}
            showSkip
        />
    );
}
