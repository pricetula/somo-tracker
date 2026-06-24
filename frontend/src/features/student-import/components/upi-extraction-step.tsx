/**
 * Wizard Step 4: UPI Number Extraction.
 *
 * Single column select with Skip option.
 */

"use client";

import * as React from "react";
import { ColumnWizardStep, type SelectSlotConfig } from "./column-wizard-step";

interface UpiExtractionStepProps {
    headers: string[];
    previewRow: Record<string, string>;
    selectedColumn: string | null;
    onColumnChange: (column: string | null) => void;
    claimedColumns: Set<string>;
    onConfirm: () => void;
    onSkip: () => void;
    onBack?: () => void;
}

export function UpiExtractionStep({
    headers,
    previewRow,
    selectedColumn,
    onColumnChange,
    claimedColumns,
    onConfirm,
    onSkip,
    onBack,
}: UpiExtractionStepProps) {
    const options = React.useMemo(() => headers.map((h) => ({ value: h, label: h })), [headers]);

    const previewValue = selectedColumn ? (previewRow[selectedColumn] ?? "") : "";

    const selectSlots: SelectSlotConfig[] = [
        {
            id: "upi-col",
            label: "UPI Number column",
            value: selectedColumn ?? "",
            options,
            onChange: (v) => onColumnChange(v),
        },
    ];

    return (
        <ColumnWizardStep
            explainer="Select the column containing the student's UPI (Unique Personal Identifier) number as per Kenya NEMIS specification."
            previewValue={previewValue}
            placeholder="KP1234567X"
            selectSlots={selectSlots}
            claimedColumns={claimedColumns}
            onConfirm={onConfirm}
            onBack={onBack}
            onSkip={onSkip}
            showSkip
        />
    );
}
