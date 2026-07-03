/**
 * state-machine.test.ts — stage transition and resume logic tests.
 *
 * Tests the ImportStoreProvider + useImportStore for:
 * - Correct stage transitions
 * - Resume into non-MAPPING stages
 * - Cross-school stale guard
 * - Schema version mismatch reset
 * - Clear import
 * - Stale import prompt (7 days)
 * - 2,000-row gating (canContinue behavior)
 */

import { describe, it, expect, beforeEach } from "vitest";
import { renderHook, act } from "@testing-library/react";

import { setImportMeta, putStagedRows, clearAll } from "@/lib/import-data/db";
import type { ImportMeta, StagedRow } from "@/lib/import-data/types";

// We need to test the ImportStoreProvider + useImportStore.
// Since the provider wraps React context, we test via a helper.
import React from "react";
import { ImportStoreProvider, useImportStore } from "../hooks/use-import-store";

// ─── Helpers ──────────────────────────────────────────────────────────────

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
        total_rows: 100,
        schema_version: 2,
        created_at: new Date().toISOString(),
        classes_last_fetched_at: null,
        idempotency_key: null,
        import_job_id: null,
        ...overrides,
    };
}

function renderStoreHook() {
    const wrapper = ({ children }: { children: React.ReactNode }) =>
        React.createElement(ImportStoreProvider, null, children);
    return renderHook(() => useImportStore(), { wrapper });
}

beforeEach(async () => {
    // Clear all IndexedDB data between tests
    await clearAll(SCHOOL_ID);
});

// ─── Tests ─────────────────────────────────────────────────────────────────

describe("State Machine — Stage Transitions", () => {
    // Test 17: State recovery integrity (Stage 2/3)
    it("TC17 — initializing with PREVIEW meta resumes on PREVIEW stage", async () => {
        await setImportMeta(createMeta({ current_stage: "PREVIEW" }));

        const { result } = renderStoreHook();
        await act(async () => {
            await result.current.initialize(SCHOOL_ID);
        });

        expect(result.current.meta?.current_stage).toBe("PREVIEW");
        expect(result.current.initialized).toBe(true);
    });

    it("initializing with READY meta resumes on READY stage", async () => {
        await setImportMeta(createMeta({ current_stage: "READY" }));

        const { result } = renderStoreHook();
        await act(async () => {
            await result.current.initialize(SCHOOL_ID);
        });

        expect(result.current.meta?.current_stage).toBe("READY");
    });

    // Test 18: Stage 1 resume preserves manual overrides
    it("TC18 — resume preserves column_mapping and does not re-run auto-detection", async () => {
        const meta = createMeta({
            column_mapping: {
                full_name: ["Full Name"],
                gender: "Jinsia", // user manually changed this
                date_of_birth: "DOB",
                class_room: null,
                nemis_number: null,
                assessment_number: null,
                birth_certificate_number: null,
            },
            academic_term_id: "term-2", // user manually selected this
        });
        await setImportMeta(meta);

        const { result } = renderStoreHook();
        await act(async () => {
            await result.current.initialize(SCHOOL_ID);
        });

        expect(result.current.meta?.column_mapping.gender).toBe("Jinsia");
        expect(result.current.meta?.academic_term_id).toBe("term-2");
    });

    // Test 19: Cross-school stale data guard
    it("TC19 — mismatched school_id clears DB rather than resuming", async () => {
        // Meta stored for school-OLD, but we initialize with school-123
        await setImportMeta(createMeta({ school_id: "school-old" }));

        const { result } = renderStoreHook();
        await act(async () => {
            await result.current.initialize("school-123");
        });

        // Should have cleared the old meta and started fresh
        expect(result.current.meta).toBeNull();
        expect(result.current.initialized).toBe(true);
    });

    // Test 20: Schema version mismatch
    it("TC20 — schema_version 1 (older than current 2) triggers clean reset", async () => {
        // Write directly to IndexedDB to bypass setImportMeta's version override
        const { openDB } = await import("idb");
        const db = await openDB("student_imports_db", 2);
        await db.put("import_meta", { ...createMeta({ schema_version: 1 }) });
        db.close();

        const { result } = renderStoreHook();
        await act(async () => {
            await result.current.initialize(SCHOOL_ID);
        });

        expect(result.current.meta).toBeNull();
        expect(result.current.isStale).toBe(true);
    });

    // Test 21: Clear Current Import
    it("TC21 — clearImport wipes meta and staged rows", async () => {
        await setImportMeta(createMeta());
        await putStagedRows([
            {
                row_number: 0,
                raw_data: {},
                processed_data: {
                    full_name: "Test",
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
            },
        ]);

        const { result } = renderStoreHook();
        await act(async () => {
            await result.current.clearImport(SCHOOL_ID);
        });

        expect(result.current.meta).toBeNull();
        expect(result.current.stagedRows).toHaveLength(0);
    });
});

describe("State Machine — Idempotency Key", () => {
    it("setIdempotencyKey persists to meta", async () => {
        await setImportMeta(createMeta());

        const { result } = renderStoreHook();
        await act(async () => {
            await result.current.initialize(SCHOOL_ID);
        });

        const key = crypto.randomUUID();
        await act(async () => {
            await result.current.setIdempotencyKey(key);
        });

        expect(result.current.meta?.idempotency_key).toBe(key);
    });

    it("setImportJobId persists to meta", async () => {
        await setImportMeta(createMeta({ current_stage: "SUBMITTING" }));

        const { result } = renderStoreHook();
        await act(async () => {
            await result.current.initialize(SCHOOL_ID);
        });

        await act(async () => {
            await result.current.setImportJobId("job-abc-123");
        });

        expect(result.current.meta?.import_job_id).toBe("job-abc-123");
    });
});

describe("State Machine — Derived Counts", () => {
    it("errorCount and skippedCount are computed correctly", async () => {
        await setImportMeta(createMeta());

        const { result } = renderStoreHook();
        await act(async () => {
            await result.current.initialize(SCHOOL_ID);
        });

        const rowsWithErrors: StagedRow[] = [
            {
                row_number: 0,
                raw_data: {},
                processed_data: {
                    full_name: "A",
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
                    has_error: true,
                    skipped: false,
                    submitted: false,
                    errors: {
                        missing_required: "Name required",
                        invalid_class: null,
                        invalid_date: null,
                        server_rejected: null,
                        server_error_type: null,
                    },
                },
            },
            {
                row_number: 1,
                raw_data: {},
                processed_data: {
                    full_name: "B",
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
                    has_error: true,
                    skipped: true,
                    submitted: false,
                    errors: {
                        missing_required: "Name required",
                        invalid_class: null,
                        invalid_date: null,
                        server_rejected: null,
                        server_error_type: null,
                    },
                },
            },
            {
                row_number: 2,
                raw_data: {},
                processed_data: {
                    full_name: "C",
                    gender: "F",
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
            },
        ];

        await act(async () => {
            await result.current.writeRowsBatch(rowsWithErrors);
        });

        expect(result.current.errorCount).toBe(1); // row 0 has error and is not skipped
        expect(result.current.skippedCount).toBe(1); // row 1 is skipped
    });
});

describe("State Machine — getSubmitRows", () => {
    it("filters out skipped and submitted rows", async () => {
        await setImportMeta(createMeta());
        await putStagedRows([
            {
                row_number: 0,
                raw_data: {},
                processed_data: {
                    full_name: "A",
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
            },
            {
                row_number: 1,
                raw_data: {},
                processed_data: {
                    full_name: "B",
                    gender: "F",
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
                    skipped: true,
                    submitted: false,
                    errors: {
                        missing_required: null,
                        invalid_class: null,
                        invalid_date: null,
                        server_rejected: null,
                        server_error_type: null,
                    },
                },
            },
            {
                row_number: 2,
                raw_data: {},
                processed_data: {
                    full_name: "C",
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
                    submitted: true,
                    errors: {
                        missing_required: null,
                        invalid_class: null,
                        invalid_date: null,
                        server_rejected: null,
                        server_error_type: null,
                    },
                },
            },
        ]);

        const { result } = renderStoreHook();
        await act(async () => {
            await result.current.initialize(SCHOOL_ID);
        });

        const submitRows = await result.current.getSubmitRows();
        expect(submitRows).toHaveLength(1);
        expect(submitRows[0].row_number).toBe(0);
    });
});
