/**
 * Wizard Step 5: KNEC Assessment Number Extraction.
 *
 * Single column select with Skip option.
 */

"use client";

import * as React from "react";
import { ColumnWizardStep, type SelectSlotConfig } from "./column-wizard-step";

interface KnecExtractionStepProps {
    headers: string[];
    previewRow: Record<string, string>;
    selectedColumn: string | null;
    onColumnChange: (column: string | null) => void;
    claimedColumns: Set<string>;
    onConfirm: () => void;
    onSkip: () => void;
    onBack?: () => void;
}

export function KnecExtractionStep({
    headers,
    previewRow,
    selectedColumn,
    onColumnChange,
    claimedColumns,
    onConfirm,
    onSkip,
    onBack,
}: KnecExtractionStepProps) {
    const options = React.useMemo(() => headers.map((h) => ({ value: h, label: h })), [headers]);

    const previewValue = selectedColumn ? (previewRow[selectedColumn] ?? "") : "";

    const selectSlots: SelectSlotConfig[] = [
        {
            id: "knec-col",
            label: "KNEC Assessment Number column",
            value: selectedColumn ?? "",
            options,
            onChange: (v) => onColumnChange(v),
        },
    ];

    return (
        <ColumnWizardStep
            explainer="Select the column containing the student's KNEC assessment number (6-14 alphanumeric characters)."
            previewValue={previewValue}
            placeholder="KNEC-123456"
            selectSlots={selectSlots}
            claimedColumns={claimedColumns}
            onConfirm={onConfirm}
            onBack={onBack}
            onSkip={onSkip}
            showSkip
        />
    );
}
