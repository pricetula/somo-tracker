"use client";

import { useState } from "react";
import { StudentsImportSelector } from "./import-selector";
import { ManualImportForm } from "./manual-import-form";

interface StudentsImportFormProps {
    isDialogVersion: boolean;
}

export function StudentsImportForm({ isDialogVersion }: StudentsImportFormProps) {
    const [selectedImportType, setSelectedImportType] = useState<"manual" | "file" | null>(null);

    function handleReset() {
        setSelectedImportType(null);
    }

    if (!selectedImportType) {
        return (
            <StudentsImportSelector
                onSelect={setSelectedImportType}
                isDialogVersion={isDialogVersion}
            />
        );
    }

    if (selectedImportType === "manual") {
        return <ManualImportForm onReset={handleReset} />;
    }

    return <section></section>;
}
