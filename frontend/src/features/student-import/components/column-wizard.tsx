/**
 * Column Configuration Wizard (Pattern B).
 *
 * Guides the user through 7 progressive steps to map spreadsheet columns
 * to student record fields. Enforces column conflict guard.
 */

"use client";

import * as React from "react";
import { NameExtractionStep } from "./name-extraction-step";
import { GenderExtractionStep } from "./gender-extraction-step";
import { DobExtractionStep } from "./dob-extraction-step";
import { UpiExtractionStep } from "./upi-extraction-step";
import { KnecExtractionStep } from "./knec-extraction-step";
import { ParentExtractionStep } from "./parent-extraction-step";
import { ClassExtractionStep } from "./class-extraction-step";
import type { ParsedFileResult, MappingConfig } from "../types";

type WizardStep = "name" | "gender" | "dob" | "upi" | "knec" | "parent" | "class";

interface ColumnWizardProps {
    parsedFile: ParsedFileResult;
    mappingConfig: MappingConfig;
    onMappingChange: (update: Partial<MappingConfig>) => void;
    onComplete: () => void;
}

const STEP_ORDER: WizardStep[] = ["name", "gender", "dob", "upi", "knec", "parent", "class"];

const STEP_LABELS: Record<WizardStep, string> = {
    name: "Student Name",
    gender: "Gender",
    dob: "Date of Birth",
    upi: "UPI Number",
    knec: "KNEC Number",
    parent: "Parent",
    class: "Class",
};

export function ColumnWizard({
    parsedFile,
    mappingConfig,
    onMappingChange,
    onComplete,
}: ColumnWizardProps) {
    const [currentStepIdx, setCurrentStepIdx] = React.useState(0);
    const currentStep = STEP_ORDER[currentStepIdx];

    const headers = parsedFile.headers;
    const previewRow = parsedFile.previewRows[0] ?? {};

    // Build claimed columns set for conflict guard
    const claimedColumns = React.useMemo(() => {
        const claimed = new Set<string>();
        const config = mappingConfig;

        if (config.genderColumn) claimed.add(config.genderColumn);
        if (config.dobColumn) claimed.add(config.dobColumn);
        if (config.upiColumn) claimed.add(config.upiColumn);
        if (config.knecColumn) claimed.add(config.knecColumn);

        // Multi-column fields
        for (const col of config.nameColumns) {
            if (col) claimed.add(col);
        }
        for (const col of config.parentColumns) {
            if (col) claimed.add(col);
        }
        for (const col of config.classColumns) {
            if (col) claimed.add(col);
        }

        return claimed;
    }, [mappingConfig]);

    function advance() {
        if (currentStepIdx < STEP_ORDER.length - 1) {
            setCurrentStepIdx((i) => i + 1);
        } else {
            onComplete();
        }
    }

    function goBack() {
        if (currentStepIdx > 0) {
            setCurrentStepIdx((i) => i - 1);
        }
    }

    return (
        <div className="space-y-4">
            {/* Step indicator */}
            <div className="flex items-center gap-2">
                {STEP_ORDER.map((step, idx) => (
                    <React.Fragment key={step}>
                        <div
                            className={`flex size-6 items-center justify-center rounded-full text-xs font-medium ${
                                idx <= currentStepIdx
                                    ? "bg-primary text-primary-foreground"
                                    : "bg-muted text-muted-foreground"
                            }`}
                        >
                            {idx + 1}
                        </div>
                        {idx < STEP_ORDER.length - 1 && (
                            <div
                                className={`h-px flex-1 ${
                                    idx < currentStepIdx ? "bg-primary" : "bg-muted"
                                }`}
                            />
                        )}
                    </React.Fragment>
                ))}
            </div>
            <p className="text-muted-foreground text-xs">
                Step {currentStepIdx + 1} of {STEP_ORDER.length}: {STEP_LABELS[currentStep]}
            </p>

            {/* Current step */}
            {currentStep === "name" && (
                <NameExtractionStep
                    headers={headers}
                    previewRow={previewRow}
                    selectedColumns={mappingConfig.nameColumns}
                    onColumnsChange={(cols) => onMappingChange({ nameColumns: cols })}
                    claimedColumns={claimedColumns}
                    onConfirm={advance}
                />
            )}

            {currentStep === "gender" && (
                <GenderExtractionStep
                    headers={headers}
                    previewRow={previewRow}
                    selectedColumn={mappingConfig.genderColumn}
                    onColumnChange={(col) => onMappingChange({ genderColumn: col })}
                    claimedColumns={claimedColumns}
                    onConfirm={advance}
                    onBack={goBack}
                />
            )}

            {currentStep === "dob" && (
                <DobExtractionStep
                    headers={headers}
                    previewRow={previewRow}
                    selectedColumn={mappingConfig.dobColumn}
                    onColumnChange={(col) => onMappingChange({ dobColumn: col })}
                    claimedColumns={claimedColumns}
                    onConfirm={advance}
                    onSkip={() => {
                        onMappingChange({ dobColumn: null });
                        advance();
                    }}
                    onBack={goBack}
                />
            )}

            {currentStep === "upi" && (
                <UpiExtractionStep
                    headers={headers}
                    previewRow={previewRow}
                    selectedColumn={mappingConfig.upiColumn}
                    onColumnChange={(col) => onMappingChange({ upiColumn: col })}
                    claimedColumns={claimedColumns}
                    onConfirm={advance}
                    onSkip={() => {
                        onMappingChange({ upiColumn: null });
                        advance();
                    }}
                    onBack={goBack}
                />
            )}

            {currentStep === "knec" && (
                <KnecExtractionStep
                    headers={headers}
                    previewRow={previewRow}
                    selectedColumn={mappingConfig.knecColumn}
                    onColumnChange={(col) => onMappingChange({ knecColumn: col })}
                    claimedColumns={claimedColumns}
                    onConfirm={advance}
                    onSkip={() => {
                        onMappingChange({ knecColumn: null });
                        advance();
                    }}
                    onBack={goBack}
                />
            )}

            {currentStep === "parent" && (
                <ParentExtractionStep
                    headers={headers}
                    previewRow={previewRow}
                    selectedColumns={mappingConfig.parentColumns}
                    onColumnsChange={(cols) => onMappingChange({ parentColumns: cols })}
                    claimedColumns={claimedColumns}
                    onConfirm={advance}
                    onSkip={() => {
                        onMappingChange({ parentColumns: [] });
                        advance();
                    }}
                    onBack={goBack}
                />
            )}

            {currentStep === "class" && (
                <ClassExtractionStep
                    headers={headers}
                    previewRow={previewRow}
                    selectedColumns={mappingConfig.classColumns}
                    onColumnsChange={(cols) => onMappingChange({ classColumns: cols })}
                    claimedColumns={claimedColumns}
                    onConfirm={advance}
                    onSkip={() => {
                        onMappingChange({ classColumns: [] });
                        advance();
                    }}
                    onBack={goBack}
                />
            )}
        </div>
    );
}
