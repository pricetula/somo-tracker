/**
 * Web Worker for off-main-thread CSV row processing.
 *
 * Receives raw rows + column mapping + class lookup, processes them
 * in chunks of ~100, and posts results back to the main thread.
 *
 * Messages:
 *   In: { type: "process", rows: Record<string, unknown>[], column_mapping, classLookup }
 *   Out:
 *     { type: "progress", processed: number, total: number }
 *     { type: "chunk", rows: ProcessedRow[] }
 *     { type: "done" }
 *     { type: "error", message: string }
 *
 * No IndexedDB or DOM access — pure worker.
 */

import {
    normalizeClassName,
    normalizeGender,
    normalizeDateOfBirth,
    similarity,
} from "../lib/import-data/matching";

// ─── Types ────────────────────────────────────────────────────────────────

interface ColumnMapping {
    full_name: string[];
    gender: string | null;
    date_of_birth: string | null;
    class_room: string | null;
    nemis_number: string | null;
    assessment_number: string | null;
    birth_certificate_number: string | null;
}

interface ClassLookupEntry {
    class_id: string;
    grade_level: string;
    stream_name: string;
    display_label: string;
}

interface ProcessedRow {
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

interface InMessage {
    type: "process";
    rows: Record<string, unknown>[];
    column_mapping: ColumnMapping;
    classLookup: [string, ClassLookupEntry][]; // serialized Map entries
}

// ─── Worker message handler ───────────────────────────────────────────────

self.onmessage = (e: MessageEvent<InMessage>) => {
    const { type, rows, column_mapping, classLookup: classLookupEntries } = e.data;

    if (type !== "process") return;

    // Rebuild the lookup map from serialized entries
    const classLookup = new Map<string, ClassLookupEntry>();
    for (const [key, entry] of classLookupEntries) {
        classLookup.set(key, entry);
    }

    // Build a flat map from normalized class name → class entry for fast matching
    const classMap = new Map<string, ClassLookupEntry>();
    for (const [, entry] of classLookup) {
        const key = normalizeClassName(entry.display_label);
        if (key && !classMap.has(key)) {
            classMap.set(key, entry);
        }
        const combined = normalizeClassName(`${entry.grade_level} ${entry.stream_name}`);
        if (combined && !classMap.has(combined)) {
            classMap.set(combined, entry);
        }
        const gsKey = normalizeClassName(entry.grade_level) + normalizeClassName(entry.stream_name);
        if (gsKey && !classMap.has(gsKey)) {
            classMap.set(gsKey, entry);
        }
    }

    const total = rows.length;
    const CHUNK_SIZE = 100;
    let processed = 0;

    try {
        for (let start = 0; start < total; start += CHUNK_SIZE) {
            const end = Math.min(start + CHUNK_SIZE, total);
            const chunk: ProcessedRow[] = [];

            for (let i = start; i < end; i++) {
                const raw = rows[i];
                const row = processOneRow(i, raw, column_mapping, classMap);
                chunk.push(row);
            }

            processed = end;
            // Post progress
            self.postMessage({ type: "progress", processed, total });

            // Post chunk
            self.postMessage({ type: "chunk", rows: chunk });
        }

        self.postMessage({ type: "done" });
    } catch (err) {
        self.postMessage({
            type: "error",
            message: err instanceof Error ? err.message : "Unknown worker error",
        });
    }
};

// ─── Error handler ────────────────────────────────────────────────────────

self.onerror = (event: Event | string) => {
    const message =
        typeof event === "string"
            ? event
            : ((event as ErrorEvent).message ?? "Unhandled worker error");
    self.postMessage({
        type: "error",
        message,
    });
};

// ─── Per-row processing ───────────────────────────────────────────────────

function processOneRow(
    rowNumber: number,
    raw_data: Record<string, unknown>,
    mapping: ColumnMapping,
    classMap: Map<string, ClassLookupEntry>
): ProcessedRow {
    // Full name
    const nameParts = mapping.full_name.map((col) => String(raw_data[col] ?? "").trim());
    const fullName = nameParts.filter(Boolean).join(" ").trim();

    // Gender
    const rawGender = mapping.gender ? String(raw_data[mapping.gender] ?? "") : "";
    const gender = normalizeGender(rawGender);

    // Date of birth
    const rawDob = mapping.date_of_birth ? raw_data[mapping.date_of_birth] : null;
    const dateOfBirth = normalizeDateOfBirth(rawDob);

    // Class
    const rawClass = mapping.class_room ? String(raw_data[mapping.class_room] ?? "") : "";
    let classId: string | null = null;
    let gradeLevel = "";
    let streamName = "";
    let invalidClass: string | null = null;

    if (rawClass.trim()) {
        const normalized = normalizeClassName(rawClass);
        const exact = classMap.get(normalized);
        if (exact) {
            classId = exact.class_id;
            gradeLevel = exact.grade_level;
            streamName = exact.stream_name;
        } else {
            // Fuzzy fallback
            let bestScore = 0;
            let bestEntry: ClassLookupEntry | null = null;
            for (const [, entry] of classMap) {
                const clsNorm = normalizeClassName(entry.display_label);
                const score = similarity(normalized, clsNorm);
                if (score > bestScore) {
                    bestScore = score;
                    bestEntry = entry;
                }
                const combined = normalizeClassName(`${entry.grade_level} ${entry.stream_name}`);
                const combinedScore = similarity(normalized, combined);
                if (combinedScore > bestScore) {
                    bestScore = combinedScore;
                    bestEntry = entry;
                }
            }
            if (bestEntry && bestScore >= 0.75) {
                classId = bestEntry.class_id;
                gradeLevel = bestEntry.grade_level;
                streamName = bestEntry.stream_name;
            } else {
                invalidClass = `Could not resolve "${rawClass}" to a class`;
            }
        }
    }

    // Tracking numbers
    const nemisNumber = mapping.nemis_number
        ? String(raw_data[mapping.nemis_number] ?? "") || null
        : null;
    const assessmentNumber = mapping.assessment_number
        ? String(raw_data[mapping.assessment_number] ?? "") || null
        : null;
    const birthCertificateNumber = mapping.birth_certificate_number
        ? String(raw_data[mapping.birth_certificate_number] ?? "") || null
        : null;

    // Validation
    const missingFields: string[] = [];
    if (!fullName) missingFields.push("Full Name");
    if (!gender) missingFields.push("Gender");

    const invalidDate =
        mapping.date_of_birth && String(raw_data[mapping.date_of_birth] ?? "").trim()
            ? dateOfBirth === null
                ? "Could not parse date"
                : null
            : null;

    return {
        row_number: rowNumber,
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
                    ? `${missingFields.join(" and ")} ${
                          missingFields.length === 1 ? "is" : "are"
                      } required`
                    : null,
            invalid_class: invalidClass,
            invalid_date: invalidDate,
            server_rejected: null,
            server_error_type: null,
        },
    };
}
