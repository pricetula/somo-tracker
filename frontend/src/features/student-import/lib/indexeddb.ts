/**
 * IndexedDB store for the student import engine.
 *
 * Database: somotracker_import_engine
 * Object Store: pending_student_imports
 *
 * Stores session metadata + staged student records for session recovery.
 */

import { get, set, del, createStore, type UseStore } from "idb-keyval";
import type { StagedStudentRecord, ImportSession, ImportStep, ParsedFileResult } from "../types";

// ─── Store setup ──────────────────────────────────────────────────────────

const store: UseStore = createStore("somotracker_import_engine", "pending_student_imports");

// ─── Key constants ────────────────────────────────────────────────────────

const SESSION_KEY = "session_meta";
const RECORDS_KEY = "staged_records";
const PARSED_FILE_KEY = "parsed_file_meta";

// ─── Session Operations ──────────────────────────────────────────────────

export async function saveSession(session: ImportSession): Promise<void> {
    await set(SESSION_KEY, session, store);
}

export async function loadSession(): Promise<ImportSession | null> {
    const session = await get<ImportSession | undefined>(SESSION_KEY, store);
    return session ?? null;
}

export async function updateSessionStep(step: ImportStep): Promise<void> {
    const session = await loadSession();
    if (!session) return;
    session.currentStep = step;
    session.lastUpdatedAt = new Date().toISOString();
    await saveSession(session);
}

export async function clearSession(): Promise<void> {
    await del(SESSION_KEY, store);
    await del(RECORDS_KEY, store);
    await del(PARSED_FILE_KEY, store);
}

// ─── Records Operations ─────────────────────────────────────────────────

export async function saveRecords(records: StagedStudentRecord[]): Promise<void> {
    await set(RECORDS_KEY, records, store);
}

export async function loadRecords(): Promise<StagedStudentRecord[]> {
    const records = await get<StagedStudentRecord[] | undefined>(RECORDS_KEY, store);
    return records ?? [];
}

export async function updateRecord(
    rowIndex: number,
    update: Partial<StagedStudentRecord>
): Promise<StagedStudentRecord[]> {
    const records = await loadRecords();
    const idx = records.findIndex((r) => r._rowIndex === rowIndex);
    if (idx === -1) return records;
    records[idx] = { ...records[idx], ...update };
    await set(RECORDS_KEY, records, store);
    return records;
}

export async function updateRecordBatch(
    updates: { rowIndex: number; update: Partial<StagedStudentRecord> }[]
): Promise<StagedStudentRecord[]> {
    const records = await loadRecords();
    for (const { rowIndex, update } of updates) {
        const idx = records.findIndex((r) => r._rowIndex === rowIndex);
        if (idx !== -1) {
            records[idx] = { ...records[idx], ...update };
        }
    }
    await set(RECORDS_KEY, records, store);
    return records;
}

// ─── Parsed File Meta (Pattern B) ────────────────────────────────────────

export async function saveParsedFileMeta(meta: ParsedFileResult): Promise<void> {
    await set(PARSED_FILE_KEY, meta, store);
}

export async function loadParsedFileMeta(): Promise<ParsedFileResult | null> {
    const meta = await get<ParsedFileResult | undefined>(PARSED_FILE_KEY, store);
    return meta ?? null;
}

export async function clearParsedFileMeta(): Promise<void> {
    await del(PARSED_FILE_KEY, store);
}

// ─── Existence Check ────────────────────────────────────────────────────

export async function hasStoredSession(): Promise<boolean> {
    const session = await loadSession();
    return session !== null;
}
