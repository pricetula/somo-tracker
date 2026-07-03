/**
 * indexeddb.test.ts — tests for the IndexedDB persistence layer.
 *
 * Uses fake-indexeddb (already installed globally in vitest.setup.ts).
 * Tests: CRUD on import_meta and staged_rows, schema version check,
 * pagination, filtering, clearing, staleness detection.
 */

import { describe, it, expect, beforeEach } from "vitest";

import {
    getImportMeta,
    setImportMeta,
    deleteImportMeta,
    updateImportMeta,
    getAllStagedRows,
    getStagedRowsByPage,
    putStagedRows,
    putStagedRow,
    updateStagedRow,
    deleteAllStagedRows,
    clearAll,
    checkSchemaVersion,
    getErrorRowCount,
    getSkippedRowCount,
    getStagedRowCount,
} from "../db";
import type { ImportMeta, StagedRow } from "../types";

// ─── Fixtures ──────────────────────────────────────────────────────────────

const SCHOOL_ID = "school-123";

function createMeta(overrides: Partial<ImportMeta> = {}): ImportMeta {
    return {
        school_id: SCHOOL_ID,
        current_stage: "MAPPING",
        column_mapping: {
            full_name: ["Full Name"],
            gender: "Gender",
            date_of_birth: null,
            class_room: null,
            nemis_number: null,
            assessment_number: null,
            birth_certificate_number: null,
        },
        academic_year_id: "year-2025",
        academic_term_id: "term-1",
        total_rows: 0,
        schema_version: 2,
        created_at: new Date().toISOString(),
        classes_last_fetched_at: null,
        idempotency_key: null,
        import_job_id: null,
        ...overrides,
    };
}

function createRow(rowNumber: number, overrides: Partial<StagedRow> = {}): StagedRow {
    return {
        row_number: rowNumber,
        raw_data: { "Full Name": "Test" },
        processed_data: {
            full_name: "Test Student",
            gender: "M",
            date_of_birth: null,
            class_id: null,
            grade_level: "",
            stream_name: "",
            nemis_number: null,
            assessment_number: null,
            birth_certificate_number: null,
        },
        ui_meta: {
            has_error: false,
            skipped: false,
            submitted: false,
            errors: {
                missing_required: null,
                invalid_class: null,
                invalid_date: null,
                server_rejected: null,
                server_error_type: null,
            },
        },
        ...overrides,
    };
}

// ─── Tests ─────────────────────────────────────────────────────────────────

describe("IndexedDB — import_meta", () => {
    beforeEach(async () => {
        // Clean up between tests
        const meta = await getImportMeta(SCHOOL_ID);
        if (meta) await deleteImportMeta(SCHOOL_ID);
    });

    it("set and get import meta", async () => {
        const meta = createMeta({ total_rows: 150 });
        await setImportMeta(meta);

        const fetched = await getImportMeta(SCHOOL_ID);
        expect(fetched).toBeDefined();
        expect(fetched!.school_id).toBe(SCHOOL_ID);
        expect(fetched!.total_rows).toBe(150);
        expect(fetched!.current_stage).toBe("MAPPING");
        expect(fetched!.schema_version).toBe(2);
    });

    it("update import meta partially", async () => {
        const meta = createMeta();
        await setImportMeta(meta);

        await updateImportMeta(SCHOOL_ID, { current_stage: "PREVIEW" });
        const fetched = await getImportMeta(SCHOOL_ID);
        expect(fetched!.current_stage).toBe("PREVIEW");
        // Other fields preserved
        expect(fetched!.academic_term_id).toBe("term-1");
    });

    it("returns undefined for non-existent school", async () => {
        const fetched = await getImportMeta("nonexistent");
        expect(fetched).toBeUndefined();
    });

    it("delete import meta", async () => {
        const meta = createMeta();
        await setImportMeta(meta);
        await deleteImportMeta(SCHOOL_ID);

        const fetched = await getImportMeta(SCHOOL_ID);
        expect(fetched).toBeUndefined();
    });
});

describe("IndexedDB — staged_rows", () => {
    beforeEach(async () => {
        await deleteAllStagedRows();
    });

    it("put and get staged rows", async () => {
        const rows = [createRow(0), createRow(1), createRow(2)];
        await putStagedRows(rows);

        const all = await getAllStagedRows();
        expect(all).toHaveLength(3);
        expect(all[0].row_number).toBe(0);
        expect(all[1].row_number).toBe(1);
        expect(all[2].row_number).toBe(2);
    });

    it("update a single staged row", async () => {
        await putStagedRow(
            createRow(0, {
                processed_data: { ...createRow(0).processed_data, full_name: "Original" },
            })
        );
        await updateStagedRow(0, {
            processed_data: { ...createRow(0).processed_data, full_name: "Updated" },
        });

        const all = await getAllStagedRows();
        expect(all).toHaveLength(1);
        expect(all[0].processed_data.full_name).toBe("Updated");
    });

    it("pagination returns correct slice", async () => {
        const rows = Array.from({ length: 120 }, (_, i) => createRow(i));
        await putStagedRows(rows);

        // Page 1 (50 rows)
        const page1 = await getStagedRowsByPage(1, 50);
        expect(page1.rows).toHaveLength(50);
        expect(page1.rows[0].row_number).toBe(0);
        expect(page1.rows[49].row_number).toBe(49);

        // Page 3 (last page: 20 rows)
        const page3 = await getStagedRowsByPage(3, 50);
        expect(page3.rows).toHaveLength(20);
        expect(page3.rows[0].row_number).toBe(100);
    });

    it("filter by has_error", async () => {
        const rows = [
            createRow(0),
            createRow(1, { ui_meta: { ...createRow(1).ui_meta, has_error: true } }),
            createRow(2),
            createRow(3, { ui_meta: { ...createRow(3).ui_meta, has_error: true } }),
        ];
        await putStagedRows(rows);

        const errorRows = await getStagedRowsByPage(1, 50, { hasError: true });
        expect(errorRows.total).toBe(2);
        expect(errorRows.rows).toHaveLength(2);
    });

    it("count returns correct number of rows", async () => {
        const rows = Array.from({ length: 10 }, (_, i) => createRow(i));
        await putStagedRows(rows);

        const count = await getStagedRowCount();
        expect(count).toBe(10);
    });
});

describe("IndexedDB — error counts", () => {
    beforeEach(async () => {
        await deleteAllStagedRows();
    });

    it("getErrorRowCount excludes skipped rows", async () => {
        const rows = [
            createRow(0, { ui_meta: { ...createRow(0).ui_meta, has_error: true } }),
            createRow(1, { ui_meta: { ...createRow(1).ui_meta, has_error: true, skipped: true } }),
        ];
        await putStagedRows(rows);

        const count = await getErrorRowCount();
        expect(count).toBe(1);
        const skipped = await getSkippedRowCount();
        expect(skipped).toBe(1);
    });

    it("getSkippedRowCount counts only skipped rows", async () => {
        const rows = [
            createRow(0),
            createRow(1, { ui_meta: { ...createRow(1).ui_meta, skipped: true } }),
            createRow(2, { ui_meta: { ...createRow(2).ui_meta, skipped: true } }),
        ];
        await putStagedRows(rows);

        const count = await getSkippedRowCount();
        expect(count).toBe(2);
    });
});

describe("IndexedDB — clearAll", () => {
    it("clears both import_meta and staged_rows", async () => {
        await setImportMeta(createMeta());
        await putStagedRows([createRow(0), createRow(1)]);

        await clearAll(SCHOOL_ID);

        const meta = await getImportMeta(SCHOOL_ID);
        expect(meta).toBeUndefined();

        const rows = await getAllStagedRows();
        expect(rows).toHaveLength(0);
    });
});

describe("IndexedDB — schema version check", () => {
    it("detects stale schema version", async () => {
        // Write directly to IndexedDB to bypass setImportMeta's version override
        const { openDB } = await import("idb");
        const db = await openDB("student_imports_db", 2);
        await db.put("import_meta", { ...createMeta({ schema_version: 1 }) });
        db.close();

        const { isStale } = await checkSchemaVersion(SCHOOL_ID);
        expect(isStale).toBe(true);
    });

    it("returns fresh for current schema version", async () => {
        await setImportMeta(createMeta({ schema_version: 2 }));
        const { isStale } = await checkSchemaVersion(SCHOOL_ID);
        expect(isStale).toBe(false);
    });
});
