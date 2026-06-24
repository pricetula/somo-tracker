/**
 * Validation utilities for student bulk import.
 *
 * Includes gender normalization, date parsing, UPI/KNEC format validation,
 * class name normalization, duplicate detection, and per-record validation.
 */

import type {
    Gender,
    StagedStudentRecord,
    ParentRecord,
    ClassRecord,
    ExistingStudent,
} from "../types";

// ─── GENDER NORMALIZATION ──────────────────────────────────────────────────

const GENDER_MAP: Record<string, Gender> = {
    m: "M",
    male: "M",
    boy: "M",
    b: "M",
    "1": "M",
    me: "M", // mwanaume
    f: "F",
    female: "F",
    girl: "F",
    g: "F",
    "2": "F",
    ke: "F", // kike
};

export function normalizeGender(raw: string | null | undefined): {
    gender: Gender | null;
    error?: string;
    advisory?: string;
} {
    if (!raw || raw.trim() === "") {
        return { gender: null, error: "Gender is required" };
    }
    const normalized = raw.toString().toLowerCase().trim();
    const resolved = GENDER_MAP[normalized];
    if (!resolved) {
        return { gender: null, error: `Unrecognized gender value: '${raw}'` };
    }
    return { gender: resolved };
}

// ─── DATE PARSING ──────────────────────────────────────────────────────────

export function parseDate(raw: string | null | undefined): {
    date: string | null;
    error?: string;
    advisory?: string;
} {
    if (!raw || raw.trim() === "") {
        return { date: null };
    }

    const trimmed = raw.trim();

    // ISO YYYY-MM-DD
    const isoMatch = trimmed.match(/^(\d{4})-(\d{2})-(\d{2})$/);
    if (isoMatch) {
        return { date: trimmed };
    }

    // DD/MM/YYYY or D/M/YYYY
    const ddmmMatch = trimmed.match(/^(\d{1,2})\/(\d{1,2})\/(\d{4})$/);
    if (ddmmMatch) {
        const d = parseInt(ddmmMatch[1], 10);
        const m = parseInt(ddmmMatch[2], 10);
        const y = parseInt(ddmmMatch[3], 10);
        if (m >= 1 && m <= 12 && d >= 1 && d <= 31 && y >= 1900 && y <= 2100) {
            // Check if ambiguous (DD <= 12 and MM <= 12)
            if (d <= 12 && m <= 12) {
                return {
                    date: `${y}-${String(m).padStart(2, "0")}-${String(d).padStart(2, "0")}`,
                    advisory: "Ambiguous date — assumed DD/MM/YYYY. Verify if incorrect.",
                };
            }
            return {
                date: `${y}-${String(m).padStart(2, "0")}-${String(d).padStart(2, "0")}`,
            };
        }
    }

    // DD-MM-YYYY or D-M-YYYY
    const ddmmDashMatch = trimmed.match(/^(\d{1,2})-(\d{1,2})-(\d{4})$/);
    if (ddmmDashMatch) {
        const d = parseInt(ddmmDashMatch[1], 10);
        const m = parseInt(ddmmDashMatch[2], 10);
        const y = parseInt(ddmmDashMatch[3], 10);
        if (m >= 1 && m <= 12 && d >= 1 && d <= 31 && y >= 1900 && y <= 2100) {
            if (d <= 12 && m <= 12) {
                return {
                    date: `${y}-${String(m).padStart(2, "0")}-${String(d).padStart(2, "0")}`,
                    advisory: "Ambiguous date — assumed DD/MM/YYYY. Verify if incorrect.",
                };
            }
            return {
                date: `${y}-${String(m).padStart(2, "0")}-${String(d).padStart(2, "0")}`,
            };
        }
    }

    // MM/DD/YYYY (only if DD/MM didn't match)
    const mmddMatch = trimmed.match(/^(\d{1,2})\/(\d{1,2})\/(\d{4})$/);
    if (mmddMatch) {
        // Only try MM/DD if DD/MM already failed
        // This is a fallback — but spec says assume DD/MM
        // So we treat this as error
        return {
            date: null,
            error: `Unrecognized date format: '${raw}'`,
        };
    }

    return { date: null, error: `Unrecognized date format: '${raw}'` };
}

// ─── UPI FORMAT VALIDATION (Kenya NEMIS) ──────────────────────────────────

const UPI_REGEX = /^KP\d{7}[A-Z0-9]$/;

export function validateUPI(raw: string | null | undefined): {
    upi: string | null;
    error?: string;
} {
    if (!raw || raw.trim() === "") {
        return { upi: null };
    }
    const trimmed = raw.trim().toUpperCase();
    if (!UPI_REGEX.test(trimmed)) {
        return {
            upi: null,
            error: `Invalid UPI format: '${raw}'. Expected format: KP followed by 7 digits and 1 alphanumeric character.`,
        };
    }
    return { upi: trimmed };
}

// ─── KNEC ASSESSMENT NUMBER VALIDATION ────────────────────────────────────

const KNEC_REGEX = /^[A-Za-z0-9-]{6,14}$/;

export function validateKNEC(raw: string | null | undefined): {
    knec: string | null;
    error?: string;
} {
    if (!raw || raw.trim() === "") {
        return { knec: null };
    }
    const trimmed = raw.trim().toUpperCase();
    if (!KNEC_REGEX.test(trimmed)) {
        return {
            knec: null,
            error: `Invalid KNEC assessment number format: '${raw}'. Expected 6-14 alphanumeric characters, optionally with hyphens.`,
        };
    }
    return { knec: trimmed };
}

// ─── CLASS NAME NORMALIZATION ──────────────────────────────────────────────

export function normalizeClassName(raw: string): string {
    return (
        raw
            .toLowerCase()
            .trim()
            // Strip known instructional prefixes only at the START of the string
            .replace(/^(class|grade|std|form)\s+/, "")
            // Strip single-letter prefix "g" or "g." only at the START, not mid-word
            .replace(/^g\.?\s*/, "")
            // Collapse all remaining whitespace
            .replace(/\s+/g, "")
    );
}

// ─── PARENT NAME NORMALIZATION ────────────────────────────────────────────

export function normalizeParentName(raw: string): string {
    return raw.toLowerCase().replace(/\s+/g, "");
}

// ─── DUPLICATE DETECTION ──────────────────────────────────────────────────

export function detectDuplicates(
    records: StagedStudentRecord[],
    existingStudents: ExistingStudent[]
): StagedStudentRecord[] {
    const existingByUPI = new Map<string, ExistingStudent>();
    const existingByNameDOB = new Map<string, ExistingStudent[]>();

    for (const es of existingStudents) {
        if (es.upi_number) {
            existingByUPI.set(es.upi_number.toUpperCase(), es);
        }
        if (es.full_name && es.date_of_birth) {
            const key = `${es.full_name.toLowerCase().trim()}|${es.date_of_birth}`;
            const list = existingByNameDOB.get(key) ?? [];
            list.push(es);
            existingByNameDOB.set(key, list);
        }
    }

    return records.map((record) => {
        let isDuplicate = false;

        // Check UPI match
        if (record.upi_number) {
            const match = existingByUPI.get(record.upi_number.toUpperCase());
            if (match) {
                isDuplicate = true;
            }
        }

        // Check name + DOB match
        if (record.full_name && record.date_of_birth) {
            const key = `${record.full_name.toLowerCase().trim()}|${record.date_of_birth}`;
            const matches = existingByNameDOB.get(key);
            if (matches && matches.length > 0) {
                isDuplicate = true;
            }
        }

        return {
            ...record,
            isDuplicate,
            // If already marked importAnyway, keep it
            importAnyway: record.importAnyway ?? false,
        };
    });
}

// ─── FULL RECORD VALIDATION ───────────────────────────────────────────────

export interface ValidatedRecord {
    record: StagedStudentRecord;
}

export function validateRecord(
    rowIndex: number,
    raw: Record<string, string>,
    mapping: {
        nameColumns: string[];
        genderColumn: string | null;
        dobColumn: string | null;
        upiColumn: string | null;
        knecColumn: string | null;
        parentColumns: string[];
        classColumns: string[];
    },
    // Optional lookups — if null, no matching is attempted
    parentsMap?: Map<string, ParentRecord> | null,
    classesMap?: Map<string, ClassRecord> | null
): StagedStudentRecord {
    const errors: Record<string, string> = {};
    const advisories: Record<string, string> = {};

    // ── Full Name ─────────────────────────────────────────────────
    const rawName = mapping.nameColumns
        .map((col) => raw[col] ?? "")
        .filter(Boolean)
        .map((s) => s.trim())
        .join(" ")
        .trim();

    const full_name = rawName || "";
    if (!full_name) {
        errors.full_name = "Full name is required";
    }

    // ── Gender ────────────────────────────────────────────────────
    const rawGender = mapping.genderColumn ? (raw[mapping.genderColumn] ?? null) : null;
    const genderResult = normalizeGender(rawGender);
    if (genderResult.error) {
        errors.gender = genderResult.error;
    }
    if (genderResult.advisory) {
        advisories.gender = genderResult.advisory;
    }

    // ── Date of Birth ─────────────────────────────────────────────
    const rawDOB = mapping.dobColumn ? (raw[mapping.dobColumn] ?? null) : null;
    const dobResult = parseDate(rawDOB);
    if (dobResult.error) {
        errors.date_of_birth = dobResult.error;
    }
    if (dobResult.advisory) {
        advisories.date_of_birth = dobResult.advisory;
    }

    // ── UPI Number ───────────────────────────────────────────────-
    const rawUPI = mapping.upiColumn ? (raw[mapping.upiColumn] ?? null) : null;
    const upiResult = validateUPI(rawUPI);
    if (upiResult.error) {
        errors.upi_number = upiResult.error;
    }

    // ── KNEC Assessment Number ───────────────────────────────────
    const rawKNEC = mapping.knecColumn ? (raw[mapping.knecColumn] ?? null) : null;
    const knecResult = validateKNEC(rawKNEC);
    if (knecResult.error) {
        errors.knec_assessment_number = knecResult.error;
    }

    // ── Parent Name Lookup ───────────────────────────────────────
    let cbc_student_parents_id: string | null = null;
    let parent_name_normalized: string | undefined;
    if (mapping.parentColumns.length > 0 && parentsMap) {
        const rawParentName = mapping.parentColumns
            .map((col) => raw[col] ?? "")
            .filter(Boolean)
            .map((s) => s.trim())
            .join(" ")
            .trim();

        if (rawParentName) {
            const normalizedKey = normalizeParentName(rawParentName);
            parent_name_normalized = normalizedKey;
            const matchedParent = parentsMap.get(normalizedKey);
            if (matchedParent) {
                cbc_student_parents_id = matchedParent.id;
            } else {
                advisories.parent = `Parent not found in system: '${rawParentName}'`;
            }
        }
    }

    // ── Class Name Lookup ────────────────────────────────────────
    let class_id: string | null = null;
    let class_name_normalized: string | undefined;
    if (mapping.classColumns.length > 0 && classesMap) {
        const rawClassName = mapping.classColumns
            .map((col) => raw[col] ?? "")
            .filter(Boolean)
            .map((s) => s.trim())
            .join(" ")
            .trim();

        if (rawClassName) {
            const normalizedKey = normalizeClassName(rawClassName);
            class_name_normalized = normalizedKey;
            const matchedClass = classesMap.get(normalizedKey);
            if (matchedClass) {
                class_id = matchedClass.id;
            } else {
                advisories.class = `Class not found in system: '${rawClassName}'`;
            }
        }
    }

    const isValid = Object.keys(errors).length === 0;

    return {
        _rowIndex: rowIndex,
        full_name,
        gender:
            genderResult.gender === "M" || genderResult.gender === "F" ? genderResult.gender : null,
        date_of_birth: dobResult.date ?? null,
        upi_number: upiResult.upi,
        knec_assessment_number: knecResult.knec,
        cbc_student_parents_id,
        class_id,
        parent_name_normalized,
        class_name_normalized,
        isValid,
        isDuplicate: false,
        importAnyway: false,
        errors,
        advisories,
    };
}

/**
 * Validate a single field update on an existing record.
 * Returns a partial update with only the changed field errors/advisories.
 */
export function validateField(
    record: StagedStudentRecord,
    field: keyof StagedStudentRecord,
    value: string
): Partial<StagedStudentRecord> {
    const update: Partial<StagedStudentRecord> = {};

    switch (field) {
        case "full_name": {
            const trimmed = value.trim();
            if (!trimmed) {
                update.errors = { ...record.errors, full_name: "Full name is required" };
            } else {
                const newErrors = { ...record.errors };
                delete newErrors.full_name;
                update.errors = newErrors;
            }
            update.full_name = trimmed;
            break;
        }
        case "gender": {
            const result = normalizeGender(value);
            if (result.error) {
                update.errors = { ...record.errors, gender: result.error };
            } else {
                const newErrors = { ...record.errors };
                delete newErrors.gender;
                update.errors = newErrors;
                update.gender = result.gender;
            }
            break;
        }
        case "date_of_birth": {
            const result = parseDate(value);
            if (result.error) {
                update.errors = { ...record.errors, date_of_birth: result.error };
            } else {
                const newErrors = { ...record.errors };
                delete newErrors.date_of_birth;
                update.errors = newErrors;
                update.date_of_birth = result.date;
                if (result.advisory) {
                    update.advisories = { ...record.advisories, date_of_birth: result.advisory };
                } else {
                    const newAdv = { ...record.advisories };
                    delete newAdv.date_of_birth;
                    update.advisories = newAdv;
                }
            }
            break;
        }
        case "upi_number": {
            const result = validateUPI(value);
            if (result.error) {
                update.errors = { ...record.errors, upi_number: result.error };
            } else {
                const newErrors = { ...record.errors };
                delete newErrors.upi_number;
                update.errors = newErrors;
                update.upi_number = result.upi;
            }
            break;
        }
        case "knec_assessment_number": {
            const result = validateKNEC(value);
            if (result.error) {
                update.errors = { ...record.errors, knec_assessment_number: result.error };
            } else {
                const newErrors = { ...record.errors };
                delete newErrors.knec_assessment_number;
                update.errors = newErrors;
                update.knec_assessment_number = result.knec;
            }
            break;
        }
    }

    // Recompute isValid
    const newErrors = update.errors ?? record.errors;
    update.isValid = Object.keys(newErrors).length === 0;

    return update;
}
