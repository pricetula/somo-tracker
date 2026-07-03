/**
 * IndexedDB storage layer for the bulk student import pipeline.
 *
 * Database: student_imports_db
 * Object stores:
 *   - import_meta  (keyPath: "school_id")
 *   - staged_rows  (keyPath: "row_number")
 *
 * All IndexedDB reads/writes happen exclusively on the main thread
 * to avoid cross-context connection races.
 */

import { openDB, type IDBPDatabase } from "idb";
import type { ImportMeta, StagedRow } from "./types";
import { CURRENT_SCHEMA_VERSION } from "./types";

// ─── Constants ───────────────────────────────────────────────────────────

const DB_NAME = "student_imports_db";
const DB_VERSION = 2;

// ─── Database singleton ──────────────────────────────────────────────────

let dbPromise: Promise<IDBPDatabase> | null = null;

function getDb(): Promise<IDBPDatabase> {
    if (!dbPromise) {
        dbPromise = openDB(DB_NAME, DB_VERSION, {
            upgrade(db, oldVersion) {
                // Create stores on first install
                if (oldVersion < 1) {
                    db.createObjectStore("import_meta", { keyPath: "school_id" });
                    const stagedStore = db.createObjectStore("staged_rows", {
                        keyPath: "row_number",
                    });
                    stagedStore.createIndex("has_error", "ui_meta.has_error", {
                        unique: false,
                    });
                    stagedStore.createIndex("skipped", "ui_meta.skipped", {
                        unique: false,
                    });
                }

                if (oldVersion < 2) {
                    // v2: no structural store changes, just new fields on existing stores
                    // The schema_version check handles migration on read
                }
            },
        });
    }
    return dbPromise;
}

// ─── Import Meta ─────────────────────────────────────────────────────────

export async function getImportMeta(schoolId: string): Promise<ImportMeta | undefined> {
    const db = await getDb();
    return db.get("import_meta", schoolId);
}

export async function setImportMeta(meta: ImportMeta): Promise<void> {
    const db = await getDb();
    meta.schema_version = CURRENT_SCHEMA_VERSION;
    await db.put("import_meta", meta);
}

export async function deleteImportMeta(schoolId: string): Promise<void> {
    const db = await getDb();
    await db.delete("import_meta", schoolId);
}

export async function updateImportMeta(
    schoolId: string,
    updates: Partial<ImportMeta>
): Promise<void> {
    const db = await getDb();
    const existing = await db.get("import_meta", schoolId);
    if (!existing) return;
    const updated = { ...existing, ...updates, schema_version: CURRENT_SCHEMA_VERSION };
    await db.put("import_meta", updated);
}

// ─── Staged Rows ─────────────────────────────────────────────────────────

export async function getStagedRow(rowNumber: number): Promise<StagedRow | undefined> {
    const db = await getDb();
    return db.get("staged_rows", rowNumber);
}

export async function getAllStagedRows(): Promise<StagedRow[]> {
    const db = await getDb();
    return db.getAll("staged_rows");
}

export async function getStagedRowsByPage(
    page: number,
    pageSize: number,
    filter?: { hasError?: boolean }
): Promise<{ rows: StagedRow[]; total: number }> {
    const db = await getDb();
    const all = await db.getAll("staged_rows");

    let filtered = all;

    if (filter?.hasError) {
        filtered = all.filter((r) => r.ui_meta.has_error);
    }

    const total = filtered.length;
    const start = (page - 1) * pageSize;
    const end = start + pageSize;
    const rows = filtered.slice(start, end).sort((a, b) => a.row_number - b.row_number);

    return { rows, total };
}

export async function putStagedRow(row: StagedRow): Promise<void> {
    const db = await getDb();
    await db.put("staged_rows", row);
}

export async function putStagedRows(rows: StagedRow[]): Promise<void> {
    const db = await getDb();
    const tx = db.transaction("staged_rows", "readwrite");
    for (const row of rows) {
        await tx.store.put(row);
    }
    await tx.done;
}

export async function updateStagedRow(
    rowNumber: number,
    updates: Partial<StagedRow>
): Promise<void> {
    const db = await getDb();
    const existing = await db.get("staged_rows", rowNumber);
    if (!existing) return;
    await db.put("staged_rows", { ...existing, ...updates });
}

export async function deleteAllStagedRows(): Promise<void> {
    const db = await getDb();
    await db.clear("staged_rows");
}

export async function getStagedRowCount(): Promise<number> {
    const db = await getDb();
    return (await db.getAllKeys("staged_rows")).length;
}

export async function getErrorRowCount(): Promise<number> {
    const db = await getDb();
    const all = await db.getAll("staged_rows");
    return all.filter((r) => r.ui_meta.has_error && !r.ui_meta.skipped).length;
}

export async function getSkippedRowCount(): Promise<number> {
    const db = await getDb();
    const all = await db.getAll("staged_rows");
    return all.filter((r) => r.ui_meta.skipped).length;
}

// ─── Clearing (full reset) ──────────────────────────────────────────────

export async function clearAll(schoolId: string): Promise<void> {
    const db = await getDb();
    const tx = db.transaction(["import_meta", "staged_rows"], "readwrite");
    await tx.objectStore("import_meta").delete(schoolId);
    await tx.objectStore("staged_rows").clear();
    await tx.done;
}

// ─── Schema version check ───────────────────────────────────────────────

export async function checkSchemaVersion(
    schoolId: string
): Promise<{ meta: ImportMeta | undefined; isStale: boolean }> {
    const meta = await getImportMeta(schoolId);
    if (!meta) return { meta: undefined, isStale: false };

    const isStale = (meta.schema_version ?? 0) < CURRENT_SCHEMA_VERSION;
    return { meta, isStale };
}
