/**
 * TypeScript interfaces for the Student Bulk Import feature.
 *
 * Note: UPI formatting follows Kenya NEMIS specification:
 *   - A 10-character alphanumeric code
 *   - Format: KP\d{7}[A-Z0-9]
 *
 * KNEC Assessment Number format:
 *   - 8-12 character alphanumeric
 *   - Alphanumeric, may include hyphens
 */

// ─── Existing Student (for duplicate detection) ─────────────────────────

export interface ExistingStudent {
    full_name: string;
    date_of_birth?: string | null;
    upi_number?: string | null;
}

// ─── Reference Lookups ───────────────────────────────────────────────────

export interface ParentRecord {
    id: string;
    full_name: string;
    phone?: string;
    email?: string;
}

export interface ClassRecord {
    id: string;
    name: string;
}

// ─── Lookup Maps (O(1) keyed by normalized name) ─────────────────────────

export type ParentsMap = Map<string, ParentRecord>;
export type ClassesMap = Map<string, ClassRecord>;

export interface LookupState {
    parents: ParentsMap;
    classes: ClassesMap;
    parentsLoaded: boolean;
    classesLoaded: boolean;
    parentsError: string | null;
    classesError: string | null;
}

// ─── Staged Record (Phase 3) ──────────────────────────────────────────────

export type Gender = "M" | "F";

export interface StagedStudentRecord {
    _rowIndex: number;
    full_name: string;
    gender: Gender | null;
    date_of_birth: string | null; // YYYY-MM-DD or null
    upi_number: string | null;
    knec_assessment_number: string | null;
    cbc_student_parents_id: string | null;
    parent_name_normalized?: string; // Raw normalized parent name (for UI)
    class_name_normalized?: string; // Raw normalized class name (for UI)
    class_id: string | null;
    isValid: boolean;
    isDuplicate: boolean;
    importAnyway: boolean;
    errors: Record<string, string>;
    advisories: Record<string, string>;
}

// ─── Session Metadata (IndexedDB) ────────────────────────────────────────

export interface ImportSession {
    sessionId: string;
    createdAt: string; // ISO timestamp
    lastUpdatedAt: string; // ISO timestamp
    currentStep: ImportStep;
    totalRecords: number;
    ingestionPattern: "manual" | "csv";
    mappingConfig: MappingConfig;
    academicYear: string;
    term: string;
}

export type ImportStep =
    | "selector" // Choose manual vs file
    | "term-select" // Select academic year & term
    | "manual-entry" // Pattern A: manual grid
    | "file-wizard" // Pattern B: file column wizard
    | "staging" // Background processing
    | "validation" // Phase 4: review
    | "submitting" // POST in flight
    | "results"; // Done / partial success

export interface MappingConfig {
    nameColumns: string[];
    genderColumn: string | null;
    dobColumn: string | null;
    upiColumn: string | null;
    knecColumn: string | null;
    parentColumns: string[];
    classColumns: string[];
}

// ─── API Payload Types ────────────────────────────────────────────────────

export interface StudentImportPayload {
    full_name: string;
    gender: "M" | "F";
    date_of_birth?: string | null;
    upi_number?: string | null;
    knec_assessment_number?: string | null;
    cbc_student_parents_id?: string | null;
    class_id?: string | null;
}

// ─── Import Request (wraps students + academic context) ────────────────

export interface StudentBulkImportRequest {
    academic_year: string;
    term: string;
    students: StudentImportPayload[];
}

export interface ImportResponseRow {
    index: number;
    status: "success" | "error";
    full_name: string;
    error_message?: string;
    field_errors?: Record<string, string>;
}

export type ImportResponse = ImportResponseRow[];

export interface ImportResultSummary {
    total: number;
    successCount: number;
    failureCount: number;
    failures: ImportResponseRow[];
    status: "success" | "partial" | "error";
    message?: string;
}

// ─── Academic Reference Data ─────────────────────────────────────────────

export interface AcademicYearRecord {
    id: string;
    name: string;
    start_date: string;
    end_date: string;
    is_current: boolean;
}

export interface AcademicPeriodRecord {
    id: string;
    name: string;
    term_number: number;
    start_date: string;
    end_date: string;
    is_current: boolean;
}

// ─── Wizard Step Types (Pattern B) ───────────────────────────────────────

export interface ParsedFileResult {
    headers: string[];
    previewRows: Record<string, string>[];
    totalRows: number;
    fullData: Record<string, string>[]; // Deferred — populated later
    fileName: string;
}
