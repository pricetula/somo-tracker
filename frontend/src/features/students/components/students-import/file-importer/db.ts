/**
 * IndexedDB database operations for the student import wizard.
 *
 * Uses the `idb` library for a promise-based IndexedDB wrapper.
 * Three object stores:
 *   - student_import_staging  — staged student records (keyed by auto-increment id)
 *   - import_meta             — current session metadata (keyed by `session:<school_id>`)
 *   - parsed_file             — raw parsed file rows for crash-recovery of early steps
 */

import { openDB, type IDBPDatabase } from "idb";
import type { StagedStudentRecord, ImportSessionMeta, WizardStep, StoredParsedFile } from "./types";

const DB_NAME = "somo_student_import";
const DB_VERSION = 3;

// ─── Database Initialization ──────────────────────────────────────────────

let _dbPromise: Promise<IDBPDatabase<unknown>> | null = null;

function getDb(): Promise<IDBPDatabase<unknown>> {
    if (!_dbPromise) {
        _dbPromise = openDB(DB_NAME, DB_VERSION, {
            upgrade(db, oldVersion) {
                // V1 stores
                if (!db.objectStoreNames.contains("student_import_staging")) {
                    db.createObjectStore("student_import_staging", {
                        keyPath: "id",
                        autoIncrement: true,
                    });
                }
                if (!db.objectStoreNames.contains("import_meta")) {
                    db.createObjectStore("import_meta", { keyPath: "id" });
                } else if (oldVersion < 2) {
                    // V1 had no keyPath on import_meta; delete and recreate with keyPath
                    db.deleteObjectStore("import_meta");
                    db.createObjectStore("import_meta", { keyPath: "id" });
                }

                // V3: parsed_file store + school_id index on staging
                if (oldVersion < 3) {
                    if (!db.objectStoreNames.contains("parsed_file")) {
                        db.createObjectStore("parsed_file", { keyPath: "id" });
                    }
                }
            },
        });
    }
    return _dbPromise;
}

// ─── Session Meta Operations ──────────────────────────────────────────────
// Keys are scoped by school: `session:<school_id>`

function sessionKey(schoolId: string): `session:${string}` {
    return `session:${schoolId}`;
}

export async function getSessionMeta(schoolId: string): Promise<ImportSessionMeta | undefined> {
    const db = await getDb();
    return (await db.get("import_meta", sessionKey(schoolId))) as ImportSessionMeta | undefined;
}

export async function saveSessionMeta(
    schoolId: string,
    meta: Partial<ImportSessionMeta>
): Promise<void> {
    const db = await getDb();
    const existing = await getSessionMeta(schoolId);
    const now = new Date().toISOString();
    const defaults: ImportSessionMeta = {
        id: sessionKey(schoolId),
        current_step: "upload",
        file_name: "",
        total_rows: 0,
        column_mappings: {},
        class_mappings: {},
        updated_at: now,
        school_id: schoolId,
    };
    const updated: ImportSessionMeta = {
        ...defaults,
        ...existing,
        ...meta,
        id: sessionKey(schoolId) as `session:${string}`,
        updated_at: now,
        school_id: schoolId,
    };
    await db.put("import_meta", updated);
}

export async function updateSessionStep(schoolId: string, step: WizardStep): Promise<void> {
    await saveSessionMeta(schoolId, { current_step: step });
}

export async function clearAllSessions(): Promise<void> {
    const db = await getDb();
    const tx = db.transaction(
        ["student_import_staging", "import_meta", "parsed_file"],
        "readwrite"
    );
    await Promise.all([
        tx.objectStore("student_import_staging").clear(),
        tx.objectStore("import_meta").clear(),
        tx.objectStore("parsed_file").clear(),
        tx.done,
    ]);
}

// ─── Parsed File Operations ───────────────────────────────────────────────
// Stores the raw parse result so the wizard can resume from column_mapping
// or class_resolving without the user having to re-upload.

export async function saveParsedFile(
    schoolId: string,
    data: Pick<StoredParsedFile, "file_name" | "sheet_name" | "headers" | "rows" | "total_rows">
): Promise<StoredParsedFile> {
    const db = await getDb();
    const id = `parsed_file:${schoolId}` as const;
    const record: StoredParsedFile = { id, ...data };
    await db.put("parsed_file", record);
    return record;
}

export async function getParsedFile(schoolId: string): Promise<StoredParsedFile | undefined> {
    const db = await getDb();
    return (await db.get("parsed_file", `parsed_file:${schoolId}`)) as StoredParsedFile | undefined;
}

export async function deleteParsedFile(schoolId: string): Promise<void> {
    const db = await getDb();
    await db.delete("parsed_file", `parsed_file:${schoolId}`);
}

// ─── Staging Record Operations ────────────────────────────────────────────

export async function getStagedRecords(schoolId: string): Promise<StagedStudentRecord[]> {
    const db = await getDb();
    const all = (await db.getAll("student_import_staging")) as StagedStudentRecord[];
    return all.filter((r) => !schoolId || r.school_id === schoolId);
}

export async function getStagedRecordsByStatus(
    schoolId: string,
    status: "valid" | "error" | "duplicate" | "submitted"
): Promise<StagedStudentRecord[]> {
    const all = await getStagedRecords(schoolId);
    return all.filter((r) => r.status === status);
}

export async function getStagedRecord(id: number): Promise<StagedStudentRecord | undefined> {
    const db = await getDb();
    return (await db.get("student_import_staging", id)) as StagedStudentRecord | undefined;
}

/**
 * Bulk-write staged records in a single IndexedDB transaction.
 * Removes the `id` field if present so autoIncrement assigns new keys.
 * Returns the array with assigned IDs.
 */
export async function bulkWriteStagedRecords(
    records: StagedStudentRecord[]
): Promise<StagedStudentRecord[]> {
    const db = await getDb();
    const tx = db.transaction("student_import_staging", "readwrite");
    const store = tx.objectStore("student_import_staging");
    const ids: number[] = [];

    for (const record of records) {
        const { id: _id, ...rest } = record;
        const newId = (await store.add(rest)) as number;
        ids.push(newId);
    }

    await tx.done;

    return records.map((r, i) => ({
        ...r,
        id: ids[i],
    }));
}

/**
 * Update a single staged record (atomic write).
 * Debounce-wrapped callers should use this directly.
 */
export async function updateStagedRecord(record: StagedStudentRecord): Promise<void> {
    const db = await getDb();
    await db.put("student_import_staging", record);
}

/**
 * Update multiple staged records in a single transaction.
 */
export async function updateStagedRecords(records: StagedStudentRecord[]): Promise<void> {
    const db = await getDb();
    const tx = db.transaction("student_import_staging", "readwrite");
    const store = tx.objectStore("student_import_staging");
    for (const record of records) {
        await store.put(record);
    }
    await tx.done;
}

/**
 * Get total count of records in staging for the given school.
 */
export async function getStagedCount(schoolId: string): Promise<number> {
    const all = await getStagedRecords(schoolId);
    return all.length;
}

/**
 * Count records by status for the given school.
 */
export async function getStagedCountByStatus(schoolId: string): Promise<{
    total: number;
    valid: number;
    error: number;
    duplicate: number;
    submitted: number;
}> {
    const all = await getStagedRecords(schoolId);
    const total = all.length;
    const valid = all.filter((r) => r.status === "valid").length;
    const error = all.filter((r) => r.status === "error").length;
    const duplicate = all.filter((r) => r.status === "duplicate").length;
    const submitted = all.filter((r) => r.status === "submitted").length;
    return { total, valid, error, duplicate, submitted };
}

/**
 * Paginated read from staging for the given school.
 */
export async function getStagedRecordsPaginated(
    schoolId: string,
    page: number,
    pageSize: number,
    filter?: "all" | "valid" | "error"
): Promise<{ records: StagedStudentRecord[]; total: number }> {
    const all = await getStagedRecords(schoolId);
    let filtered = all;
    if (filter === "valid") {
        filtered = all.filter((r) => r.status === "valid");
    } else if (filter === "error") {
        filtered = all.filter((r) => r.status === "error" || r.status === "duplicate");
    }
    const sorted = [...filtered].sort((a, b) => (a.id ?? 0) - (b.id ?? 0));
    const start = (page - 1) * pageSize;
    const records = sorted.slice(start, start + pageSize);
    return { records, total: filtered.length };
}

/**
 * Get the storage estimate from the browser.
 */
export async function getStorageEstimate(): Promise<{
    usage: number;
    quota: number;
    nearQuota: boolean;
}> {
    if (!navigator.storage?.estimate) {
        return { usage: 0, quota: 0, nearQuota: false };
    }
    const estimate = await navigator.storage.estimate();
    const usage = estimate.usage ?? 0;
    const quota = estimate.quota ?? 0;
    return {
        usage,
        quota,
        nearQuota: quota > 0 && usage / quota > 0.85,
    };
}

/**
 * Pre-flight storage check for a bulk write.
 * Returns a user-facing error message if storage appears insufficient,
 * or null if the write can proceed safely.
 */
export async function checkStorageForBulkWrite(
    recordCount: number,
    bytesPerRecord: number = 2048
): Promise<string | null> {
    if (!navigator.storage?.estimate) return null;

    try {
        const estimate = await navigator.storage.estimate();
        const quota = estimate.quota ?? 0;
        const usage = estimate.usage ?? 0;

        if (quota === 0) return null; // cannot determine

        const needed = recordCount * bytesPerRecord;
        const available = quota - usage;

        // Leave 512 KB of headroom for safety
        if (needed + 512 * 1024 > available) {
            return (
                `Not enough browser storage available to save ${recordCount.toLocaleString()} records. ` +
                `Need approximately ${(needed / 1024 / 1024).toFixed(1)} MB but only ` +
                `${(available / 1024 / 1024).toFixed(1)} MB free. ` +
                `Try a smaller file or free up browser storage.`
            );
        }
    } catch {
        // If the estimate call itself fails, allow the write to proceed
    }

    return null;
}
