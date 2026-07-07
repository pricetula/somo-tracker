/**
 * Validation utilities for student import records.
 *
 * Implements the explicit field validation rules from the spec:
 * - full_name: required, 2-100 chars, no digits, allows Swahili/accented chars
 * - upi_number: optional
 * - knec_assessment_number: optional
 * - date_of_birth: optional, must be valid ISO date, not future, 3-20 years warning
 * - class_id: optional, populated by class resolver
 */

import type { StagedStudentRecord } from "../types";

// ─── Constants ────────────────────────────────────────────────────────────

const FULL_NAME_MIN = 2;
const FULL_NAME_MAX = 100;
const NAME_PATTERN = /^[\p{L}\p{M}'\-\s]+$/u;
const MIN_AGE_YEARS = 3;
const WARNING_AGE_YEARS = 20;

// ─── Individual Field Validation ──────────────────────────────────────────

export interface FieldValidationResult {
    valid: boolean;
    errors: string[];
    warnings: string[];
}

export interface RecordValidationResult {
    status: "valid" | "error";
    errors: string[];
}

/**
 * Validate a single field value against a target field key.
 */
function validateField(value: string | undefined | null, targetKey: string): FieldValidationResult {
    const errors: string[] = [];
    const warnings: string[] = [];

    // ── full_name ──────────────────────────────────────────────────────
    if (targetKey === "full_name") {
        if (!value || value.trim().length === 0) {
            errors.push("Full name is required");
            return { valid: false, errors, warnings };
        }

        const trimmed = value.trim();

        if (trimmed.length < FULL_NAME_MIN) {
            errors.push(`Full name must be at least ${FULL_NAME_MIN} characters`);
        }

        if (trimmed.length > FULL_NAME_MAX) {
            errors.push(`Full name must be at most ${FULL_NAME_MAX} characters`);
        }

        if (!NAME_PATTERN.test(trimmed)) {
            errors.push("Full name can only contain letters, spaces, hyphens, and apostrophes");
        }

        if (/\d/.test(trimmed)) {
            errors.push("Full name cannot contain digits");
        }
    }

    // ── date_of_birth ──────────────────────────────────────────────────
    if (targetKey === "date_of_birth" && value && value.trim().length > 0) {
        const dateStr = value.trim();

        // Must parse to valid ISO date
        const parsed = new Date(dateStr);
        if (isNaN(parsed.getTime())) {
            errors.push("Date of birth is not a valid date");
            return { valid: errors.length === 0, errors, warnings };
        }

        // Must not be a future date
        const today = new Date();
        today.setHours(0, 0, 0, 0);
        if (parsed > today) {
            errors.push("Date of birth cannot be in the future");
        }

        // Age bounds check (warning, not hard-block)
        const ageMs = today.getTime() - parsed.getTime();
        const ageYears = ageMs / (365.25 * 86400000);
        if (ageYears < MIN_AGE_YEARS) {
            warnings.push(
                `Student appears to be younger than ${MIN_AGE_YEARS} years old (${Math.floor(ageYears)} years) — please verify`
            );
        }
        if (ageYears > WARNING_AGE_YEARS) {
            warnings.push(
                `Student appears to be older than ${WARNING_AGE_YEARS} years old (${Math.floor(ageYears)} years) — please verify`
            );
        }
    }

    return { valid: errors.length === 0, errors, warnings };
}

// ─── Record-Level Validation ──────────────────────────────────────────────

/**
 * Validate a complete staged record against all field rules.
 * Returns the updated record with status and errors.
 */
export function validateRecord(record: StagedStudentRecord): StagedStudentRecord {
    const errors: string[] = [];
    const allWarnings: string[] = [];

    // Validate each mapped field
    for (const [key, value] of Object.entries(record.payload)) {
        if (key === "full_name" || key === "date_of_birth") {
            const result = validateField(value as string | undefined, key);
            errors.push(...result.errors);
            allWarnings.push(...result.warnings);
        }
    }

    return {
        ...record,
        status: errors.length > 0 ? "error" : "valid",
        errors: [...errors, ...(record.errors ?? [])],
    };
}

/**
 * Batch-validate all records in an array.
 */
export function validateRecords(records: StagedStudentRecord[]): StagedStudentRecord[] {
    return records.map(validateRecord);
}

// ─── Duplicate Detection ──────────────────────────────────────────────────

export interface DuplicateResult {
    record: StagedStudentRecord;
    duplicate_of: number[]; // ids of rows this conflicts with
}

/**
 * Detect duplicates within a set of staged records.
 *
 * Two types of duplicates:
 * 1. Same `upi_number` across different rows (when present)
 * 2. Same `full_name` + `date_of_birth` combination (when both present)
 *
 * Returns records marked as "duplicate" with conflict info in errors.
 */
export function detectDuplicates(records: StagedStudentRecord[]): StagedStudentRecord[] {
    const upiMap = new Map<string, number[]>();
    const nameDobMap = new Map<string, number[]>();

    // Build conflict maps — use unique temp ids for records without DB ids
    for (let i = 0; i < records.length; i++) {
        const record = records[i];
        const id = record.id ?? -(i + 1); // negative index is unique, won't collide with autoIncrement
        const { upi_number, full_name, date_of_birth } = record.payload;

        // UPI duplicates
        if (upi_number && upi_number.trim().length > 0) {
            const key = upi_number.trim().toLowerCase();
            const existing = upiMap.get(key) ?? [];
            existing.push(id);
            upiMap.set(key, existing);
        }

        // Name + DOB duplicates
        if (full_name && date_of_birth) {
            const key = `${full_name.trim().toLowerCase()}::${date_of_birth}`;
            const existing = nameDobMap.get(key) ?? [];
            existing.push(id);
            nameDobMap.set(key, existing);
        }
    }

    // Collect all conflicting IDs
    const conflictIds = new Set<number>();
    const idToConflictMap = new Map<number, number[]>();

    for (const [, ids] of upiMap) {
        if (ids.length > 1) {
            for (const id of ids) {
                conflictIds.add(id);
                const existingDups = idToConflictMap.get(id) ?? [];
                idToConflictMap.set(id, [
                    ...new Set([...existingDups, ...ids.filter((d) => d !== id)]),
                ]);
            }
        }
    }

    for (const [, ids] of nameDobMap) {
        if (ids.length > 1) {
            for (const id of ids) {
                conflictIds.add(id);
                const existingDups = idToConflictMap.get(id) ?? [];
                idToConflictMap.set(id, [
                    ...new Set([...existingDups, ...ids.filter((d) => d !== id)]),
                ]);
            }
        }
    }

    // Mark duplicates
    return records.map((record, i) => {
        const id = record.id ?? -(i + 1);
        if (conflictIds.has(id)) {
            const conflictIds_ = idToConflictMap.get(id) ?? [];
            return {
                ...record,
                status: "duplicate",
                errors: [
                    ...record.errors,
                    `Duplicate: conflicts with row(s) ${conflictIds_.join(", ")}`,
                ],
            };
        }
        return record;
    });
}

// ─── Combined Validation + Duplicates ─────────────────────────────────────

/**
 * Full validation pipeline: validate field rules, then detect duplicates.
 */
export function validateAndDetectDuplicates(records: StagedStudentRecord[]): StagedStudentRecord[] {
    const validated = validateRecords(records);
    return detectDuplicates(validated);
}
