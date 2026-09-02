/**
 * Shared helpers for resolving in-progress import job details.
 */

import { getActiveImportJob } from "@/lib/api/imports";

/**
 * Resolves the authoritative total record count for an in-progress import job.
 *
 * When a submit fails with `import_already_in_progress`, the error body only
 * carries the active job id — not its total record count. The bulk invite
 * forms call this so the shared ImportProgress shows the real total instead
 * of the (usually smaller) current batch count.
 *
 * Falls back to `fallback` if the lookup fails or the active job no longer
 * matches, so callers can always proceed.
 */
export async function resolveActiveJobTotalRecords(
    jobId: string,
    fallback: number
): Promise<number> {
    try {
        const { job } = await getActiveImportJob();
        if (job && job.id === jobId && job.total_records > 0) {
            return job.total_records;
        }
    } catch (err) {
        console.warn("Failed to resolve active job total records; using fallback.", err);
    }
    return fallback;
}
