/**
 * Import Store Context & Provider.
 *
 * Provides useImportStore hook + ImportStoreProvider component.
 * The internal state management lives in useImportStoreInternal.
 */

"use client";

import * as React from "react";
import {
    updateImportMeta,
    clearAll,
    getAllStagedRows,
    getStagedRowsByPage,
    putStagedRows,
    updateStagedRow,
    checkSchemaVersion,
    getErrorRowCount,
    getSkippedRowCount,
} from "@/lib/import-data/db";
import {
    type ImportMeta,
    type StagedRow,
    type ImportStage,
    type ColumnMapping,
} from "@/lib/import-data/types";

// ─── Types ──────────────────────────────────────────────────────────────

interface ImportStoreState {
    /** Current import meta record, or null if none exists / stale */
    meta: ImportMeta | null;
    /** All staged rows (loaded on demand for performance) */
    stagedRows: StagedRow[];
    /** Whether the store has been initialized */
    initialized: boolean;
    /** Whether the stored schema is stale and needs clearing */
    isStale: boolean;
    /** Derived counts */
    errorCount: number;
    skippedCount: number;
    /** Loading state */
    loading: boolean;
}

interface ImportStoreActions {
    /** Initialize the store for a given school_id */
    initialize: (schoolId: string) => Promise<void>;
    /** Clear all import data for the current school */
    clearImport: (schoolId: string) => Promise<void>;
    /** Set the current stage */
    setStage: (stage: ImportStage) => Promise<void>;
    /** Set column mapping (debounced externally) */
    setColumnMapping: (mapping: ColumnMapping) => Promise<void>;
    /** Set academic year / term */
    setAcademicYear: (academicYearId: string) => Promise<void>;
    setAcademicTerm: (academicTermId: string) => Promise<void>;
    /** Write a batch of processed rows */
    writeRowsBatch: (rows: StagedRow[]) => Promise<void>;
    /** Update a single row's ui_meta (skip, class, etc.) */
    updateRow: (rowNumber: number, updates: Partial<StagedRow>) => Promise<void>;
    /** Load staged rows for a page */
    loadPage: (
        page: number,
        pageSize?: number,
        filter?: { hasError?: boolean }
    ) => Promise<{ rows: StagedRow[]; total: number }>;
    /** Set idempotency key */
    setIdempotencyKey: (key: string) => Promise<void>;
    /** Set import job ID */
    setImportJobId: (jobId: string) => Promise<void>;
    /** Refresh counts */
    refreshCounts: () => Promise<void>;
    /** Get rows eligible for submission (skipped=false, submitted=false) */
    getSubmitRows: () => Promise<StagedRow[]>;
}

type ImportStore = ImportStoreState & ImportStoreActions;

const ImportStoreContext = React.createContext<ImportStore | null>(null);

// ─── Hook ─────────────────────────────────────────────────────────────────

export function useImportStore(): ImportStore {
    const ctx = React.useContext(ImportStoreContext);
    if (!ctx) {
        throw new Error("useImportStore must be used within an ImportStoreProvider");
    }
    return ctx;
}

// ─── Provider ─────────────────────────────────────────────────────────────

export function ImportStoreProvider({ children }: { children: React.ReactNode }) {
    const store = useImportStoreInternal();
    return <ImportStoreContext.Provider value={store}>{children}</ImportStoreContext.Provider>;
}

// ─── Internal hook (single instance) ──────────────────────────────────────

function useImportStoreInternal(): ImportStore {
    const [state, setState] = React.useState<ImportStoreState>({
        meta: null,
        stagedRows: [],
        initialized: false,
        isStale: false,
        errorCount: 0,
        skippedCount: 0,
        loading: false,
    });

    const metaRef = React.useRef(state.meta);

    // Sync ref with latest meta after render
    React.useEffect(() => {
        metaRef.current = state.meta;
    }, [state.meta]);

    // ── Initialize ────────────────────────────────────────────────────────
    const initialize = React.useCallback(async (schoolId: string) => {
        setState((s) => ({ ...s, loading: true }));

        try {
            const { meta, isStale } = await checkSchemaVersion(schoolId);

            if (isStale && meta) {
                // Schema version mismatch — clear and restart
                await clearAll(schoolId);
                setState((s) => ({
                    ...s,
                    meta: null,
                    stagedRows: [],
                    initialized: true,
                    isStale: true,
                    loading: false,
                }));
                return;
            }

            // Check cross-school stale guard
            if (meta && meta.school_id !== schoolId) {
                await clearAll(schoolId);
                setState((s) => ({
                    ...s,
                    meta: null,
                    stagedRows: [],
                    initialized: true,
                    loading: false,
                }));
                return;
            }

            // Check staleness (7 days)
            if (meta) {
                const created = new Date(meta.created_at);
                const now = new Date();
                const daysOld = (now.getTime() - created.getTime()) / (1000 * 60 * 60 * 24);
                if (daysOld > 7) {
                    setState((s) => ({
                        ...s,
                        meta,
                        initialized: true,
                        isStale: true,
                        loading: false,
                    }));
                    return;
                }
            }

            if (meta) {
                const stagedRows = await getAllStagedRows();
                const errorCount = stagedRows.filter(
                    (r) => r.ui_meta.has_error && !r.ui_meta.skipped
                ).length;
                const skippedCount = stagedRows.filter((r) => r.ui_meta.skipped).length;

                setState((s) => ({
                    ...s,
                    meta,
                    stagedRows,
                    initialized: true,
                    isStale: false,
                    errorCount,
                    skippedCount,
                    loading: false,
                }));
            } else {
                setState((s) => ({
                    ...s,
                    meta: null,
                    stagedRows: [],
                    initialized: true,
                    isStale: false,
                    loading: false,
                }));
            }
        } catch {
            setState((s) => ({
                ...s,
                initialized: true,
                loading: false,
            }));
        }
    }, []);

    // ── Clear ─────────────────────────────────────────────────────────────
    const clearImport = React.useCallback(async (schoolId: string) => {
        await clearAll(schoolId);
        setState({
            meta: null,
            stagedRows: [],
            initialized: true,
            isStale: false,
            errorCount: 0,
            skippedCount: 0,
            loading: false,
        });
    }, []);

    // ── Set stage ─────────────────────────────────────────────────────────
    const setStage = React.useCallback(async (stage: ImportStage) => {
        const meta = metaRef.current;
        if (!meta) return;
        await updateImportMeta(meta.school_id, { current_stage: stage });
        setState((s) => (s.meta ? { ...s, meta: { ...s.meta, current_stage: stage } } : s));
    }, []);

    // ── Set column mapping ────────────────────────────────────────────────
    const setColumnMapping = React.useCallback(async (mapping: ColumnMapping) => {
        const meta = metaRef.current;
        if (!meta) return;
        await updateImportMeta(meta.school_id, { column_mapping: mapping });
        setState((s) => (s.meta ? { ...s, meta: { ...s.meta, column_mapping: mapping } } : s));
    }, []);

    // ── Set academic year ─────────────────────────────────────────────────
    const setAcademicYear = React.useCallback(async (academicYearId: string) => {
        const meta = metaRef.current;
        if (!meta) return;
        await updateImportMeta(meta.school_id, { academic_year_id: academicYearId });
        setState((s) =>
            s.meta ? { ...s, meta: { ...s.meta, academic_year_id: academicYearId } } : s
        );
    }, []);

    // ── Set academic term ─────────────────────────────────────────────────
    const setAcademicTerm = React.useCallback(async (academicTermId: string) => {
        const meta = metaRef.current;
        if (!meta) return;
        await updateImportMeta(meta.school_id, { academic_term_id: academicTermId });
        setState((s) =>
            s.meta ? { ...s, meta: { ...s.meta, academic_term_id: academicTermId } } : s
        );
    }, []);

    // ── Write rows batch ──────────────────────────────────────────────────
    const writeRowsBatch = React.useCallback(async (rows: StagedRow[]) => {
        await putStagedRows(rows);
        setState((s) => {
            const existing = new Map(s.stagedRows.map((r) => [r.row_number, r]));
            for (const row of rows) {
                existing.set(row.row_number, row);
            }
            const updated = Array.from(existing.values()).sort(
                (a, b) => a.row_number - b.row_number
            );
            const errorCount = updated.filter(
                (r) => r.ui_meta.has_error && !r.ui_meta.skipped
            ).length;
            const skippedCount = updated.filter((r) => r.ui_meta.skipped).length;

            return {
                ...s,
                stagedRows: updated,
                errorCount,
                skippedCount,
                meta: s.meta ? { ...s.meta, total_rows: updated.length } : s.meta,
            };
        });
    }, []);

    // ── Update row ────────────────────────────────────────────────────────
    const updateRow = React.useCallback(async (rowNumber: number, updates: Partial<StagedRow>) => {
        await updateStagedRow(rowNumber, updates);
        setState((s) => {
            const rows = s.stagedRows.map((r) =>
                r.row_number === rowNumber ? { ...r, ...updates } : r
            );
            const errorCount = rows.filter((r) => r.ui_meta.has_error && !r.ui_meta.skipped).length;
            const skippedCount = rows.filter((r) => r.ui_meta.skipped).length;
            return { ...s, stagedRows: rows, errorCount, skippedCount };
        });
    }, []);

    // ── Load page ─────────────────────────────────────────────────────────
    const loadPage = React.useCallback(
        async (page: number, pageSize = 50, filter?: { hasError?: boolean }) => {
            return getStagedRowsByPage(page, pageSize, filter);
        },
        []
    );

    // ── Set idempotency key ───────────────────────────────────────────────
    const setIdempotencyKey = React.useCallback(async (key: string) => {
        const meta = metaRef.current;
        if (!meta) return;
        await updateImportMeta(meta.school_id, { idempotency_key: key });
        setState((s) => (s.meta ? { ...s, meta: { ...s.meta, idempotency_key: key } } : s));
    }, []);

    const setImportJobId = React.useCallback(async (jobId: string) => {
        const meta = metaRef.current;
        if (!meta) return;
        await updateImportMeta(meta.school_id, { import_job_id: jobId });
        setState((s) => (s.meta ? { ...s, meta: { ...s.meta, import_job_id: jobId } } : s));
    }, []);

    const refreshCounts = React.useCallback(async () => {
        const meta = metaRef.current;
        if (!meta) {
            setState((s) => ({ ...s, errorCount: 0, skippedCount: 0 }));
            return;
        }
        const errorCount = await getErrorRowCount();
        const skippedCount = await getSkippedRowCount();
        setState((s) => ({ ...s, errorCount, skippedCount }));
    }, []);

    const getSubmitRows = React.useCallback(async () => {
        const rows = await getAllStagedRows();
        return rows.filter((r) => !r.ui_meta.skipped && !r.ui_meta.submitted);
    }, []);

    const value = React.useMemo<ImportStore>(
        () => ({
            ...state,
            initialize,
            clearImport,
            setStage,
            setColumnMapping,
            setAcademicYear,
            setAcademicTerm,
            writeRowsBatch,
            updateRow,
            loadPage,
            setIdempotencyKey,
            setImportJobId,
            refreshCounts,
            getSubmitRows,
        }),
        [
            state,
            initialize,
            clearImport,
            setStage,
            setColumnMapping,
            setAcademicYear,
            setAcademicTerm,
            writeRowsBatch,
            updateRow,
            loadPage,
            setIdempotencyKey,
            setImportJobId,
            refreshCounts,
            getSubmitRows,
        ]
    );

    return value;
}
