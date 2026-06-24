/**
 * Tests for IndexedDB operations used by the student import engine.
 *
 * Covers: session save/load/clear, records save/load/update,
 * parsed file meta, existence check, and session recovery flow.
 *
 * These tests use fake-indexeddb (installed globally in vitest.setup.ts).
 *
 * To run: pnpm vitest run src/features/student-import/__tests__/indexeddb.test.ts
 */

import { describe, it, expect, beforeEach } from "vitest";
import { clear } from "idb-keyval";
import {
    saveSession,
    loadSession,
    clearSession,
    updateSessionStep,
    saveRecords,
    loadRecords,
    updateRecord,
    updateRecordBatch,
    saveParsedFileMeta,
    loadParsedFileMeta,
    clearParsedFileMeta,
    hasStoredSession,
} from "../lib/indexeddb";
import type { ImportSession, StagedStudentRecord, ParsedFileResult } from "../types";

// ⚠️ fake-indexeddb is installed globally in __tests__/setup/vitest.setup.ts.
// Each test starts with a clean IndexedDB via the global beforeEach(clear).

describe("IndexedDB: Session Operations", () => {
    beforeEach(async () => {
        await clear();
    });

    const sampleSession: ImportSession = {
        sessionId: "session-uuid-123",
        createdAt: "2026-06-24T10:00:00.000Z",
        lastUpdatedAt: "2026-06-24T10:05:00.000Z",
        currentStep: "validation",
        totalRecords: 50,
        ingestionPattern: "csv",
        mappingConfig: {
            nameColumns: ["full_name"],
            genderColumn: "gender",
            dobColumn: "date_of_birth",
            upiColumn: "upi_number",
            knecColumn: "knec_assessment_number",
            parentColumns: ["parent_name"],
            classColumns: ["class_name"],
        },
    };

    it("saves and loads a session", async () => {
        await saveSession(sampleSession);
        const loaded = await loadSession();
        expect(loaded).not.toBeNull();
        expect(loaded!.sessionId).toBe("session-uuid-123");
        expect(loaded!.currentStep).toBe("validation");
        expect(loaded!.ingestionPattern).toBe("csv");
        expect(loaded!.totalRecords).toBe(50);
    });

    it("returns null when no session exists", async () => {
        const loaded = await loadSession();
        expect(loaded).toBeNull();
    });

    it("clears session data", async () => {
        await saveSession(sampleSession);
        await clearSession();
        const loaded = await loadSession();
        expect(loaded).toBeNull();
    });

    it("updates session step and timestamp", async () => {
        await saveSession(sampleSession);
        await updateSessionStep("results");

        const loaded = await loadSession()!;
        expect(loaded!.currentStep).toBe("results");
        expect(loaded!.lastUpdatedAt).not.toBe(sampleSession.lastUpdatedAt);
    });

    it("updateSessionStep returns gracefully when no session exists", async () => {
        // Should not throw
        await updateSessionStep("manual-entry");
        const loaded = await loadSession();
        expect(loaded).toBeNull();
    });

    it("overwrites an existing session on save", async () => {
        await saveSession(sampleSession);
        const updated = { ...sampleSession, totalRecords: 100, currentStep: "results" as const };
        await saveSession(updated);
        const loaded = await loadSession();
        expect(loaded!.totalRecords).toBe(100);
        expect(loaded!.currentStep).toBe("results");
    });

    it("hasStoredSession returns true when session exists", async () => {
        await saveSession(sampleSession);
        const exists = await hasStoredSession();
        expect(exists).toBe(true);
    });

    it("hasStoredSession returns false when no session exists", async () => {
        const exists = await hasStoredSession();
        expect(exists).toBe(false);
    });
});

describe("IndexedDB: Records Operations", () => {
    beforeEach(async () => {
        await clear();
    });

    const sampleRecords: StagedStudentRecord[] = [
        {
            _rowIndex: 0,
            full_name: "John Kamau",
            gender: "M",
            date_of_birth: "2010-03-15",
            upi_number: "KP1234567A",
            knec_assessment_number: "ABC12345",
            cbc_student_parents_id: "parent-1",
            class_id: "class-1",
            isValid: true,
            isDuplicate: false,
            importAnyway: false,
            errors: {},
            advisories: {},
        },
        {
            _rowIndex: 1,
            full_name: "Jane Wanjiku",
            gender: "F",
            date_of_birth: "2011-07-22",
            upi_number: null,
            knec_assessment_number: null,
            cbc_student_parents_id: null,
            class_id: null,
            isValid: true,
            isDuplicate: true,
            importAnyway: false,
            errors: {},
            advisories: {},
        },
    ];

    it("saves and loads records", async () => {
        await saveRecords(sampleRecords);
        const loaded = await loadRecords();
        expect(loaded).toHaveLength(2);
        expect(loaded[0].full_name).toBe("John Kamau");
        expect(loaded[1].full_name).toBe("Jane Wanjiku");
    });

    it("returns empty array when no records exist", async () => {
        const loaded = await loadRecords();
        expect(loaded).toEqual([]);
    });

    it("updates a single record by rowIndex", async () => {
        await saveRecords(sampleRecords);
        const updated = await updateRecord(0, { full_name: "John Kamau Updated" });
        expect(updated[0].full_name).toBe("John Kamau Updated");
        expect(updated[1].full_name).toBe("Jane Wanjiku"); // unchanged
    });

    it("updateRecord returns same records when rowIndex not found", async () => {
        await saveRecords(sampleRecords);
        const updated = await updateRecord(99, { full_name: "Ghost" });
        expect(updated).toHaveLength(2);
        expect(updated[0].full_name).toBe("John Kamau");
    });

    it("updateRecord returns empty array when no records saved", async () => {
        const updated = await updateRecord(0, { full_name: "X" });
        expect(updated).toEqual([]);
    });

    it("updates multiple records in batch", async () => {
        await saveRecords(sampleRecords);
        const updated = await updateRecordBatch([
            { rowIndex: 0, update: { full_name: "John Updated" } },
            { rowIndex: 1, update: { importAnyway: true } },
        ]);
        expect(updated[0].full_name).toBe("John Updated");
        expect(updated[1].importAnyway).toBe(true);
    });

    it("batch update ignores non-existent row indices", async () => {
        await saveRecords(sampleRecords);
        const updated = await updateRecordBatch([
            { rowIndex: 0, update: { full_name: "John" } },
            { rowIndex: 99, update: { full_name: "Ghost" } }, // doesn't exist
        ]);
        expect(updated).toHaveLength(2);
        expect(updated[0].full_name).toBe("John");
        expect(updated[1].full_name).toBe("Jane Wanjiku"); // unchanged
    });

    it("overwrites all records on second saveRecords call", async () => {
        await saveRecords(sampleRecords);
        const newBatch: StagedStudentRecord[] = [
            {
                _rowIndex: 0,
                full_name: "New Student",
                gender: "M",
                date_of_birth: null,
                upi_number: null,
                knec_assessment_number: null,
                cbc_student_parents_id: null,
                class_id: null,
                isValid: true,
                isDuplicate: false,
                importAnyway: false,
                errors: {},
                advisories: {},
            },
        ];
        await saveRecords(newBatch);
        const loaded = await loadRecords();
        expect(loaded).toHaveLength(1);
        expect(loaded[0].full_name).toBe("New Student");
    });
});

describe("IndexedDB: Parsed File Meta", () => {
    beforeEach(async () => {
        await clear();
    });

    const sampleMeta: ParsedFileResult = {
        headers: ["full_name", "gender", "date_of_birth"],
        previewRows: [{ full_name: "John Kamau", gender: "male", date_of_birth: "15/03/2010" }],
        totalRows: 100,
        fullData: [],
        fileName: "students.csv",
    };

    it("saves and loads parsed file meta", async () => {
        await saveParsedFileMeta(sampleMeta);
        const loaded = await loadParsedFileMeta();
        expect(loaded).not.toBeNull();
        expect(loaded!.headers).toEqual(["full_name", "gender", "date_of_birth"]);
        expect(loaded!.totalRows).toBe(100);
        expect(loaded!.fileName).toBe("students.csv");
    });

    it("returns null when no meta exists", async () => {
        const loaded = await loadParsedFileMeta();
        expect(loaded).toBeNull();
    });

    it("clears parsed file meta", async () => {
        await saveParsedFileMeta(sampleMeta);
        await clearParsedFileMeta();
        const loaded = await loadParsedFileMeta();
        expect(loaded).toBeNull();
    });
});
