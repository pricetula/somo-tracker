/**
 * Student Bulk Import — Public API.
 *
 * Import only from this entry point. Never import from internal paths.
 */

export { StudentImportContainer } from "./components/student-import-container";

export type {
    StagedStudentRecord,
    ImportSession,
    ImportStep,
    MappingConfig,
    ParsedFileResult,
    ParentRecord,
    ClassRecord,
    StudentImportPayload,
    ImportResultSummary,
    ImportResponseRow,
} from "./types";

export {
    normalizeClassName,
    normalizeGender,
    parseDate,
    validateUPI,
    validateKNEC,
    validateRecord,
    detectDuplicates,
} from "./lib/validation";

export {
    saveSession,
    loadSession,
    clearSession,
    saveRecords,
    loadRecords,
    updateRecord,
    hasStoredSession,
} from "./lib/indexeddb";
