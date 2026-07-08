"use client";

import { useState } from "react";
import { StudentsImportSelector } from "./import-selector";
import { StudentManualImportForm } from "./manual-import-form";
import { FileImporter } from "./file-importer";
import { ImportProgress } from "./import-progress";

interface ActiveJob {
    jobId: string;
    totalRecords: number;
}

interface StudentsImportFormProps {
    isDialogVersion: boolean;
}

export function StudentsImportForm({ isDialogVersion }: StudentsImportFormProps) {
    const [selectedImportType, setSelectedImportType] = useState<"manual" | "file" | null>(null);
    const [activeJob, setActiveJob] = useState<ActiveJob | null>(null);

    function handleReset() {
        setSelectedImportType(null);
        setActiveJob(null);
    }

    function handleJobCreated(jobId: string, totalRecords: number) {
        setActiveJob({ jobId, totalRecords });
    }

    function handleRetry() {
        // Go back to the previous step to retry
        setActiveJob(null);
    }

    // Show shared progress when a job is active
    if (activeJob) {
        return (
            <ImportProgress
                jobId={activeJob.jobId}
                totalRecords={activeJob.totalRecords}
                onDone={handleReset}
                onRetry={handleRetry}
            />
        );
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
        return <StudentManualImportForm onReset={handleReset} onJobCreated={handleJobCreated} />;
    }

    if (selectedImportType === "file") {
        return <FileImporter onReset={handleReset} onJobCreated={handleJobCreated} />;
    }

    return <section></section>;
}
