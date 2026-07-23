/**
 * IndexedDB database operations for the staff bulk-invite wizard.
 *
 * Mirrors the student import db.ts pattern.
 * Three object stores:
 *   - invite_staging       — staged invitation records (keyed by auto-increment id)
 *   - import_meta          — current session metadata (keyed by `session:<school_id>`)
 *   - parsed_file          — raw parsed file rows for crash-recovery of early steps
 */

import { openDB, type IDBPDatabase } from "idb";
import type { StagedInviteRecord, ImportSessionMeta, WizardStep, StoredParsedFile } from "./types";

const DB_NAME = "somo_staff_invite";
const DB_VERSION = 1;

// ─── Database Initialization ──────────────────────────────────────────────

let _dbPromise: Promise<IDBPDatabase<unknown>> | null = null;

function getDb(): Promise<IDBPDatabase<unknown>> {
    if (!_dbPromise) {
        _dbPromise = openDB(DB_NAME, DB_VERSION, {
            upgrade(db) {
                if (!db.objectStoreNames.contains("invite_staging")) {
                    db.createObjectStore("invite_staging", {
                        keyPath: "id",
                        autoIncrement: true,
                    });
                }
                if (!db.objectStoreNames.contains("import_meta")) {
                    db.createObjectStore("import_meta", { keyPath: "id" });
                }
                if (!db.objectStoreNames.contains("parsed_file")) {
                    db.createObjectStore("parsed_file", { keyPath: "id" });
                }
            },
        });
    }
    return _dbPromise;
}

// ─── Session Meta Operations ──────────────────────────────────────────────

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
        updated_at: now,
        school_id: schoolId,
        role: "",
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
    const tx = db.transaction(["invite_staging", "import_meta", "parsed_file"], "readwrite");
    await Promise.all([
        tx.objectStore("invite_staging").clear(),
        tx.objectStore("import_meta").clear(),
        tx.objectStore("parsed_file").clear(),
        tx.done,
    ]);
}

// ─── Parsed File Operations ───────────────────────────────────────────────

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

export async function getStagedRecords(schoolId: string): Promise<StagedInviteRecord[]> {
    const db = await getDb();
    const all = (await db.getAll("invite_staging")) as StagedInviteRecord[];
    return all.filter((r) => !schoolId || r.school_id === schoolId);
}

export async function getStagedRecordsByStatus(
    schoolId: string,
    status: "valid" | "error" | "duplicate" | "submitted"
): Promise<StagedInviteRecord[]> {
    const all = await getStagedRecords(schoolId);
    return all.filter((r) => r.status === status);
}

export async function getStagedRecord(id: number): Promise<StagedInviteRecord | undefined> {
    const db = await getDb();
    return (await db.get("invite_staging", id)) as StagedInviteRecord | undefined;
}

export async function bulkWriteStagedRecords(
    records: StagedInviteRecord[]
): Promise<StagedInviteRecord[]> {
    const db = await getDb();
    const tx = db.transaction("invite_staging", "readwrite");
    const store = tx.objectStore("invite_staging");
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

export async function updateStagedRecord(record: StagedInviteRecord): Promise<void> {
    const db = await getDb();
    await db.put("invite_staging", record);
}

export async function updateStagedRecords(records: StagedInviteRecord[]): Promise<void> {
    const db = await getDb();
    const tx = db.transaction("invite_staging", "readwrite");
    const store = tx.objectStore("invite_staging");
    for (const record of records) {
        await store.put(record);
    }
    await tx.done;
}

export async function getStagedCount(schoolId: string): Promise<number> {
    const all = await getStagedRecords(schoolId);
    return all.length;
}

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

export async function getStagedRecordsPaginated(
    schoolId: string,
    page: number,
    pageSize: number,
    filter?: "all" | "valid" | "error"
): Promise<{ records: StagedInviteRecord[]; total: number }> {
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

export async function checkStorageForBulkWrite(
    recordCount: number,
    bytesPerRecord: number = 2048
): Promise<string | null> {
    if (!navigator.storage?.estimate) return null;

    try {
        const estimate = await navigator.storage.estimate();
        const quota = estimate.quota ?? 0;
        const usage = estimate.usage ?? 0;

        if (quota === 0) return null;

        const needed = recordCount * bytesPerRecord;
        const available = quota - usage;

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
