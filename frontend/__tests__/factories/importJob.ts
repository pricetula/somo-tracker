/**
 * Factory for building import job objects in tests.
 */

import type {
    ImportJob,
    StartImportResponse,
    TrackImportResponse,
    ImportProgressEvent,
} from "@/lib/api/imports";

let jobCounter = 0;

export function buildImportJob(overrides?: Partial<ImportJob>): ImportJob {
    jobCounter++;
    return {
        id: `job-${String(jobCounter).padStart(3, "0")}`,
        tenant_id: "tenant-abc",
        school_id: "school-xyz",
        role: "NURSE",
        created_by: "user-xyz",
        status: "pending",
        total_records: 100,
        processed_records: 0,
        success_count: 0,
        failed_count: 0,
        created_at: "2025-01-15T10:00:00Z",
        ...overrides,
    };
}

export function buildStartImportResponse(
    overrides?: Partial<StartImportResponse>
): StartImportResponse {
    return {
        import_job_id: `job-${String(jobCounter + 1).padStart(3, "0")}`,
        status: "accepted",
        total: 100,
        ...overrides,
    };
}

export function buildTrackImportResponse(
    overrides?: Partial<TrackImportResponse>
): TrackImportResponse {
    return {
        job: buildImportJob(overrides?.job),
        failed_records: overrides?.failed_records ?? 0,
    };
}

export function buildImportProgressEvent(
    overrides?: Partial<ImportProgressEvent>
): ImportProgressEvent {
    return {
        type: "import_progress",
        import_job_id: "job-001",
        status: "processing",
        processed_records: 50,
        success_count: 45,
        failed_count: 5,
        total_records: 100,
        ...overrides,
    } as ImportProgressEvent;
}

export function resetJobCounter() {
    jobCounter = 0;
}
