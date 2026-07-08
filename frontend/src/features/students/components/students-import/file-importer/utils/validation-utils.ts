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

    // ── gender ─────────────────────────────────────────────────────────
    if (targetKey === "gender" && value && value.trim().length > 0) {
        const lower = value.trim().toLowerCase();
        if (lower !== "m" && lower !== "f") {
            errors.push(
                `Gender must be "M" or "F". Got "${value.trim()}". ` +
                    `Common values like "Male", "Female", "Boy", "Girl" are auto-normalized.`
            );
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
        if (key === "full_name" || key === "date_of_birth" || key === "gender") {
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
 * Checks these fields for within-batch duplicates:
 * 1. `admission_number`
 * 2. `upi_number`
 * 3. `knec_assessment_number`
 * 4. `full_name` + `date_of_birth` combination (when both present)
 *
 * Returns records marked as "duplicate" with conflict info in errors.
 *
 * NOTE: No format/pattern validation is applied to admission_number, upi_number,
 * or knec_assessment_number — their format is unknown/variable.
 */
export function detectDuplicates(records: StagedStudentRecord[]): StagedStudentRecord[] {
    // Build conflict maps — use unique temp ids for records without DB ids
    const idToRowLabel = new Map<number, string>();
    const admMap = new Map<string, number[]>();
    const upiMap = new Map<string, number[]>();
    const knecMap = new Map<string, number[]>();
    const nameDobMap = new Map<string, number[]>();

    for (let i = 0; i < records.length; i++) {
        const record = records[i];
        const id = record.id ?? -(i + 1);
        idToRowLabel.set(id, `row ${i + 1}`);

        const { admission_number, upi_number, knec_assessment_number, full_name, date_of_birth } =
            record.payload;

        if (admission_number && admission_number.trim().length > 0) {
            const key = admission_number.trim().toLowerCase();
            const existing = admMap.get(key) ?? [];
            existing.push(id);
            admMap.set(key, existing);
        }

        if (upi_number && upi_number.trim().length > 0) {
            const key = upi_number.trim().toLowerCase();
            const existing = upiMap.get(key) ?? [];
            existing.push(id);
            upiMap.set(key, existing);
        }

        if (knec_assessment_number && knec_assessment_number.trim().length > 0) {
            const key = knec_assessment_number.trim().toLowerCase();
            const existing = knecMap.get(key) ?? [];
            existing.push(id);
            knecMap.set(key, existing);
        }

        if (full_name && date_of_birth) {
            const key = `${full_name.trim().toLowerCase()}::${date_of_birth}`;
            const existing = nameDobMap.get(key) ?? [];
            existing.push(id);
            nameDobMap.set(key, existing);
        }
    }

    // Collect conflicting IDs with field-specific messages
    const conflictIds = new Set<number>();
    const idToMsg = new Map<number, string>();

    function addConflicts(
        map: Map<string, number[]>,
        fieldLabel: string,
        msgTemplate: (labels: string) => string
    ) {
        for (const [, ids] of map) {
            if (ids.length <= 1) continue;
            for (const id of ids) {
                conflictIds.add(id);
                if (!idToMsg.has(id)) {
                    const others = ids.filter((d) => d !== id);
                    const labels = others
                        .map((oid) => idToRowLabel.get(oid) ?? `#${oid}`)
                        .join(", ");
                    idToMsg.set(id, msgTemplate(labels));
                }
            }
        }
    }

    addConflicts(
        admMap,
        "admission",
        (labels) => `Duplicate admission number — also used in ${labels}`
    );
    addConflicts(upiMap, "upi", (labels) => `Duplicate UPI number — also used in ${labels}`);
    addConflicts(knecMap, "knec", (labels) => `Duplicate KNEC number — also used in ${labels}`);
    addConflicts(
        nameDobMap,
        "name+dob",
        (labels) => `Duplicate name + date of birth — also used in ${labels}`
    );

    return records.map((record, i) => {
        const id = record.id ?? -(i + 1);
        if (conflictIds.has(id)) {
            const msg = idToMsg.get(id) ?? "Duplicate record";
            return {
                ...record,
                status: "duplicate",
                errors: [...record.errors, msg],
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
