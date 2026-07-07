/**
 * IndexedDB database operations for the student import wizard.
 *
 * Uses the `idb` library for a promise-based IndexedDB wrapper.
 * Two object stores: student_import_staging and import_meta.
 */

import { openDB, type IDBPDatabase } from "idb";
import type { StagedStudentRecord, ImportSessionMeta, WizardStep } from "./types";

const DB_NAME = "somo_student_import";
const DB_VERSION = 2;

// ─── Database Initialization ──────────────────────────────────────────────

let _dbPromise: Promise<IDBPDatabase<unknown>> | null = null;

function getDb(): Promise<IDBPDatabase<unknown>> {
    if (!_dbPromise) {
        _dbPromise = openDB(DB_NAME, DB_VERSION, {
            upgrade(db, oldVersion) {
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
            },
        });
    }
    return _dbPromise;
}

// ─── Session Meta Operations ──────────────────────────────────────────────

export async function getSessionMeta(): Promise<ImportSessionMeta | undefined> {
    const db = await getDb();
    return (await db.get("import_meta", "current_session")) as ImportSessionMeta | undefined;
}

export async function saveSessionMeta(meta: Partial<ImportSessionMeta>): Promise<void> {
    const db = await getDb();
    const existing = await getSessionMeta();
    const now = new Date().toISOString();
    const defaults: ImportSessionMeta = {
        id: "current_session",
        current_step: "upload",
        file_name: "",
        total_rows: 0,
        column_mappings: {},
        class_mappings: {},
        completed_batch_ids: [],
        updated_at: now,
    };
    const updated: ImportSessionMeta = {
        ...defaults,
        ...existing,
        ...meta,
        id: "current_session" as const,
        updated_at: now,
    };
    await db.put("import_meta", updated);
}

export async function updateSessionStep(step: WizardStep): Promise<void> {
    await saveSessionMeta({ current_step: step } as Partial<ImportSessionMeta>);
}

export async function clearAllSessions(): Promise<void> {
    const db = await getDb();
    const tx = db.transaction(["student_import_staging", "import_meta"], "readwrite");
    await Promise.all([
        tx.objectStore("student_import_staging").clear(),
        tx.objectStore("import_meta").clear(),
        tx.done,
    ]);
}

// ─── Staging Record Operations ────────────────────────────────────────────

export async function getStagedRecords(): Promise<StagedStudentRecord[]> {
    const db = await getDb();
    return (await db.getAll("student_import_staging")) as StagedStudentRecord[];
}

export async function getStagedRecordsByStatus(
    status: "valid" | "error" | "duplicate" | "submitted"
): Promise<StagedStudentRecord[]> {
    const db = await getDb();
    const all = (await db.getAll("student_import_staging")) as StagedStudentRecord[];
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
        // eslint-disable-next-line @typescript-eslint/no-unused-vars
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
 * Get total count of records in staging.
 */
export async function getStagedCount(): Promise<number> {
    const db = await getDb();
    return db.count("student_import_staging");
}

/**
 * Count records by status.
 */
export async function getStagedCountByStatus(): Promise<{
    total: number;
    valid: number;
    error: number;
    duplicate: number;
    submitted: number;
}> {
    const all = await getStagedRecords();
    const total = all.length;
    const valid = all.filter((r) => r.status === "valid").length;
    const error = all.filter((r) => r.status === "error").length;
    const duplicate = all.filter((r) => r.status === "duplicate").length;
    const submitted = all.filter((r) => r.status === "submitted").length;
    return { total, valid, error, duplicate, submitted };
}

/**
 * Paginated read from staging.
 */
export async function getStagedRecordsPaginated(
    page: number,
    pageSize: number,
    filter?: "all" | "valid" | "error"
): Promise<{ records: StagedStudentRecord[]; total: number }> {
    const all = await getStagedRecords();
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
 * Update batch_id on records after assigning to a streaming batch.
 */
export async function assignBatchToRecords(recordIds: number[], batchId: string): Promise<void> {
    const db = await getDb();
    const tx = db.transaction("student_import_staging", "readwrite");
    const store = tx.objectStore("student_import_staging");
    for (const id of recordIds) {
        const record = (await store.get(id)) as StagedStudentRecord | undefined;
        if (record) {
            record.batch_id = batchId;
            record.status = "submitted";
            await store.put(record);
        }
    }
    await tx.done;
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
