/**
 * Pure functions for CSV header matching, gender normalization,
 * date parsing, and class name normalization.
 *
 * These are designed to be tree-shakeable and Worker-compatible
 * (no DOM, no IndexedDB, no React).
 */

// ============================================================================
// Header Dictionary & Matching
// ============================================================================

export const HEADER_DICTIONARY: Record<string, string[]> = {
    full_name: [
        "full name",
        "fullname",
        "student name",
        "name",
        "jina",
        "jina kamili",
        "majina",
        "jina la mwanafunzi",
        "jina la kwanza",
    ],
    gender: ["gender", "sex", "m/f", "jinsia", "ume/uke"],
    class_room: ["class room", "classroom", "class", "stream", "form", "grade", "darasa", "mkondo"],
    date_of_birth: ["date of birth", "dob", "birth date", "tarehe ya kuzaliwa"],
    nemis_number: ["nemis", "nemis number", "nemis no"],
    assessment_number: ["assessment number", "assessment no", "knec"],
    birth_certificate_number: ["birth certificate", "birth cert", "cheti cha kuzaliwa"],
};

export type HeaderField = keyof typeof HEADER_DICTIONARY;

/**
 * Normalize a header string: lowercase, trim, collapse whitespace, strip punctuation.
 */
export function normalizeHeader(raw: string): string {
    return raw
        .toLowerCase()
        .trim()
        .replace(/\s+/g, " ")
        .replace(/[^\w\s]/g, "");
}

/**
 * Levenshtein distance between two strings.
 */
export function levenshtein(a: string, b: string): number {
    const m = a.length;
    const n = b.length;
    const dp: number[][] = [];

    for (let i = 0; i <= m; i++) {
        dp[i] = [i];
    }
    for (let j = 0; j <= n; j++) {
        dp[0][j] = j;
    }

    for (let i = 1; i <= m; i++) {
        for (let j = 1; j <= n; j++) {
            const cost = a[i - 1] === b[j - 1] ? 0 : 1;
            dp[i][j] = Math.min(dp[i - 1][j] + 1, dp[i][j - 1] + 1, dp[i - 1][j - 1] + cost);
        }
    }

    return dp[m][n];
}

/**
 * Compute similarity ratio (0–1) between two strings using Levenshtein.
 * 1 = exact match, 0 = completely different.
 */
export function similarity(a: string, b: string): number {
    if (a === b) return 1;
    const dist = levenshtein(a, b);
    const maxLen = Math.max(a.length, b.length);
    if (maxLen === 0) return 1;
    return 1 - dist / maxLen;
}

/**
 * Try to match a raw CSV header to one of the dictionary fields.
 *
 * Strategy:
 * 1. Normalize the raw header.
 * 2. Try exact match against dictionary values.
 * 3. If no exact match, fuzzy match with threshold ≥ 0.8.
 * 4. If still no match at ≥ 0.8 confidence, return null (unmapped).
 */
export function matchHeader(header: string): HeaderField | null {
    const normalized = normalizeHeader(header);

    let bestField: HeaderField | null = null;
    let bestScore = 0;

    for (const [field, aliases] of Object.entries(HEADER_DICTIONARY)) {
        // Exact match against any alias (case-insensitive, already normalized)
        for (const alias of aliases) {
            if (normalized === normalizeHeader(alias)) {
                return field as HeaderField;
            }
        }

        // Fuzzy fallback
        for (const alias of aliases) {
            const score = similarity(normalized, normalizeHeader(alias));
            if (score > bestScore) {
                bestScore = score;
                bestField = field as HeaderField;
            }
        }
    }

    // Only return if confidence ≥ 0.8
    if (bestScore >= 0.8) {
        return bestField;
    }

    return null;
}

/**
 * Match all headers at once, returning a mapping of normalized header → field.
 * Unmatched headers get null.
 */
export function matchAllHeaders(headers: string[]): Record<string, HeaderField | null> {
    const result: Record<string, HeaderField | null> = {};
    const assigned = new Set<HeaderField>();

    for (const header of headers) {
        const normalized = normalizeHeader(header);
        const field = matchHeader(header);
        if (field && !assigned.has(field)) {
            // Deterministic single assignment: first header wins for each field
            result[normalized] = field;
            assigned.add(field);
        } else {
            // Field already assigned or no match
            result[normalized] = null;
        }
    }

    return result;
}

// ============================================================================
// Gender Normalization
// ============================================================================

export const GENDER_VALUE_MAP: Record<string, "M" | "F"> = {
    m: "M",
    male: "M",
    boy: "M",
    ume: "M",
    mvulana: "M",
    f: "F",
    female: "F",
    girl: "F",
    uke: "F",
    msichana: "F",
};

/**
 * Normalize a gender cell value.
 * Returns "M" | "F" | null (null = unrecognized or blank).
 */
export function normalizeGender(value: string): "M" | "F" | "" {
    const cleaned = value?.toLowerCase().trim() ?? "";
    if (!cleaned) return "";
    return GENDER_VALUE_MAP[cleaned] ?? "";
}

// ============================================================================
// Class Name Normalization & Matching
// ============================================================================

/**
 * Normalize a class name for fuzzy matching.
 * Lowercase, strip noise tokens (grade, class, form, stream),
 * strip all whitespace and punctuation.
 */
export function normalizeClassName(raw: string): string {
    return raw
        .toLowerCase()
        .trim()
        .replace(/\b(grade|class|form|stream)\b/g, "")
        .replace(/[\s\p{P}]+/gu, "")
        .trim();
}

/**
 * A single class record for matching.
 */
export interface ClassMatchRecord {
    id: string;
    grade_level: string;
    stream_name: string;
    display_label: string;
}

/**
 * Build a lookup map from normalized class name → class ID.
 * Keys are generated from grade_level + stream_name combinations.
 */
export function buildClassLookup(classes: ClassMatchRecord[]): Map<string, string> {
    const map = new Map<string, string>();

    for (const cls of classes) {
        // Key from display_label
        const key = normalizeClassName(cls.display_label);
        if (key && !map.has(key)) {
            map.set(key, cls.id);
        }

        // Also key from grade_level + stream_name combined
        const combined = normalizeClassName(`${cls.grade_level} ${cls.stream_name}`);
        if (combined && !map.has(combined)) {
            map.set(combined, cls.id);
        }

        // Key from grade_level alone + stream_name alone — useful for partial matches
        const gradeKey = normalizeClassName(cls.grade_level);
        const streamKey = normalizeClassName(cls.stream_name);
        if (gradeKey && streamKey) {
            const gsKey = gradeKey + streamKey;
            if (!map.has(gsKey)) {
                map.set(gsKey, cls.id);
            }
        }
    }

    return map;
}

/**
 * Fuzzy-match a classroom text to a class ID.
 * Returns { class_id, grade_level, stream_name } or null if below threshold.
 * Threshold: ≥ 0.75 (Levenshtein-based).
 */
export function fuzzyMatchClass(
    classroomText: string,
    classes: ClassMatchRecord[],
    lookupMap: Map<string, string>
): { class_id: string; grade_level: string; stream_name: string } | null {
    if (!classroomText?.trim()) return null;

    const normalized = normalizeClassName(classroomText);

    // Exact match in lookup map first
    const exactId = lookupMap.get(normalized);
    if (exactId) {
        const cls = classes.find((c) => c.id === exactId);
        if (cls) {
            return {
                class_id: cls.id,
                grade_level: cls.grade_level,
                stream_name: cls.stream_name,
            };
        }
    }

    // Fuzzy fallback
    let bestMatch: (typeof classes)[0] | null = null;
    let bestScore = 0;

    for (const cls of classes) {
        const clsNorm = normalizeClassName(cls.display_label);
        const score = similarity(normalized, clsNorm);
        if (score > bestScore) {
            bestScore = score;
            bestMatch = cls;
        }

        // Also try grade_level + stream_name
        const combined = normalizeClassName(`${cls.grade_level} ${cls.stream_name}`);
        const combinedScore = similarity(normalized, combined);
        if (combinedScore > bestScore) {
            bestScore = combinedScore;
            bestMatch = cls;
        }
    }

    if (bestMatch && bestScore >= 0.75) {
        return {
            class_id: bestMatch.id,
            grade_level: bestMatch.grade_level,
            stream_name: bestMatch.stream_name,
        };
    }

    return null;
}

// ============================================================================
// Date of Birth Normalization
// ============================================================================

/**
 * Normalize a date of birth value to ISO YYYY-MM-DD.
 * Accepts:
 *   - Excel serial date numbers (as string or number)
 *   - DD/MM/YYYY
 *   - YYYY-MM-DD
 *   - DD-MM-YYYY
 *   - Common Swahili-locale text dates
 *
 * Returns ISO string or null if unparseable.
 */
export function normalizeDateOfBirth(value: unknown): string | null {
    if (value === null || value === undefined || value === "") return null;

    // If it's a number (Excel serial date)
    if (typeof value === "number" || (typeof value === "string" && /^\d+$/.test(value.trim()))) {
        const serial = typeof value === "number" ? value : parseInt(value.trim(), 10);
        const date = excelSerialToDate(serial);
        if (date && !isNaN(date.getTime())) {
            return formatISODate(date);
        }
        return null;
    }

    const str = String(value).trim();

    // Try ISO format YYYY-MM-DD first
    const isoMatch = str.match(/^(\d{4})-(\d{1,2})-(\d{1,2})$/);
    if (isoMatch) {
        const date = new Date(
            parseInt(isoMatch[1]),
            parseInt(isoMatch[2]) - 1,
            parseInt(isoMatch[3])
        );
        if (!isNaN(date.getTime())) return formatISODate(date);
    }

    // Try DD/MM/YYYY or DD-MM-YYYY
    const dmyMatch = str.match(/^(\d{1,2})[\/-](\d{1,2})[\/-](\d{4})$/);
    if (dmyMatch) {
        const date = new Date(
            parseInt(dmyMatch[3]),
            parseInt(dmyMatch[2]) - 1,
            parseInt(dmyMatch[1])
        );
        if (!isNaN(date.getTime())) return formatISODate(date);
    }

    // Try generic Date parse as last resort
    const date = new Date(str);
    if (!isNaN(date.getTime()) && date.getFullYear() > 1900) {
        return formatISODate(date);
    }

    return null;
}

/**
 * Convert an Excel serial date number to a Date object.
 * Excel serial date 1 = January 1, 1900.
 */
function excelSerialToDate(serial: number): Date {
    // Excel incorrectly treats 1900 as a leap year
    const corrected = serial > 60 ? serial - 1 : serial;
    const epoch = new Date(1899, 11, 30); // Day 0 in Excel is 1899-12-30
    return new Date(epoch.getTime() + corrected * 86_400_000);
}

function formatISODate(date: Date): string {
    const y = date.getFullYear();
    const m = String(date.getMonth() + 1).padStart(2, "0");
    const d = String(date.getDate()).padStart(2, "0");
    return `${y}-${m}-${d}`;
}

// ============================================================================
// Row Processing (pure, Worker-compatible)
// ============================================================================

/**
 * Processed row state (matches the StagedRow.processed_data shape).
 */
export interface ProcessedRow {
    row_number: number;
    full_name: string;
    gender: "M" | "F" | "";
    date_of_birth: string | null;
    class_id: string | null;
    grade_level: string;
    stream_name: string;
    nemis_number: string | null;
    assessment_number: string | null;
    birth_certificate_number: string | null;
    has_error: boolean;
    skipped: boolean;
    errors: {
        missing_required: string | null;
        invalid_class: string | null;
        invalid_date: string | null;
        server_rejected: string | null;
        server_error_type: string | null;
    };
}

/**
 * Process a single raw row into a ProcessedRow.
 * Pure function — no side effects.
 */
export function processRow(
    row_number: number,
    raw_data: Record<string, unknown>,
    column_mapping: {
        full_name: string[];
        gender: string | null;
        date_of_birth: string | null;
        class_room: string | null;
        nemis_number: string | null;
        assessment_number: string | null;
        birth_certificate_number: string | null;
    },
    classLookup: Map<string, { class_id: string; grade_level: string; stream_name: string }>
): ProcessedRow {
    // Build full_name from concatenation of mapped columns
    const nameParts = column_mapping.full_name.map((col) => String(raw_data[col] ?? "").trim());
    const fullName = nameParts.filter(Boolean).join(" ").trim();

    // Gender
    const rawGender = column_mapping.gender ? String(raw_data[column_mapping.gender] ?? "") : "";
    const gender = normalizeGender(rawGender);

    // Date of birth
    const rawDob = column_mapping.date_of_birth ? raw_data[column_mapping.date_of_birth] : null;
    const dateOfBirth = normalizeDateOfBirth(rawDob);

    // Class resolution
    const rawClass = column_mapping.class_room
        ? String(raw_data[column_mapping.class_room] ?? "")
        : "";
    let classId: string | null = null;
    let gradeLevel = "";
    let streamName = "";
    let invalidClass: string | null = null;

    if (rawClass.trim()) {
        // Try exact match first, then fuzzy
        const normalized = normalizeClassName(rawClass);
        const exact = classLookup.get(normalized);
        if (exact) {
            classId = exact.class_id;
            gradeLevel = exact.grade_level;
            streamName = exact.stream_name;
        } else {
            // Fall back to fuzzy
            for (const [, entry] of classLookup) {
                const clsNorm = normalizeClassName(`${entry.grade_level} ${entry.stream_name}`);
                const score = similarity(normalized, clsNorm);
                if (score >= 0.75) {
                    classId = entry.class_id;
                    gradeLevel = entry.grade_level;
                    streamName = entry.stream_name;
                    break;
                }
            }
            if (!classId) {
                invalidClass = `Could not resolve "${rawClass}" to a class`;
            }
        }
    }

    // Tracking numbers
    const nemisNumber = column_mapping.nemis_number
        ? String(raw_data[column_mapping.nemis_number] ?? "") || null
        : null;
    const assessmentNumber = column_mapping.assessment_number
        ? String(raw_data[column_mapping.assessment_number] ?? "") || null
        : null;
    const birthCertificateNumber = column_mapping.birth_certificate_number
        ? String(raw_data[column_mapping.birth_certificate_number] ?? "") || null
        : null;

    // Validation
    const missingFields: string[] = [];
    if (!fullName) missingFields.push("Full Name");
    if (!gender) missingFields.push("Gender");

    const invalidDate =
        dateOfBirth === null
            ? column_mapping.date_of_birth &&
              String(raw_data[column_mapping.date_of_birth] ?? "").trim()
                ? "Could not parse date"
                : null
            : null;

    return {
        row_number,
        full_name: fullName,
        gender,
        date_of_birth: dateOfBirth,
        class_id: classId,
        grade_level: gradeLevel,
        stream_name: streamName,
        nemis_number: nemisNumber,
        assessment_number: assessmentNumber,
        birth_certificate_number: birthCertificateNumber,
        has_error: missingFields.length > 0 || invalidClass !== null,
        skipped: false,
        errors: {
            missing_required:
                missingFields.length > 0
                    ? `${missingFields.join(" and ")} ${missingFields.length === 1 ? "is" : "are"} required`
                    : null,
            invalid_class: invalidClass,
            invalid_date: invalidDate,
            server_rejected: null,
            server_error_type: null,
        },
    };
}
