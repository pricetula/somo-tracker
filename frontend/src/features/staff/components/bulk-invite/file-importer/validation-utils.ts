/**
 * Validation utilities for staff invite staging records.
 * Matches the student import validation-utils pattern.
 */

import type { StagedInviteRecord } from "./types";

// ─── Constants ────────────────────────────────────────────────────────────

const EMAIL_REGEX = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

// ─── Record-Level Validation ──────────────────────────────────────────────

/**
 * Validate a complete staged invite record.
 * Returns the updated record with status and errors.
 */
export function validateRecord(record: StagedInviteRecord): StagedInviteRecord {
    const errors: string[] = [];
    const email = record.email.trim();

    if (!email) {
        errors.push("Email is required");
    } else if (!EMAIL_REGEX.test(email)) {
        errors.push(`Invalid email format: "${email}"`);
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
export function validateRecords(records: StagedInviteRecord[]): StagedInviteRecord[] {
    return records.map(validateRecord);
}

// ─── Duplicate Detection ──────────────────────────────────────────────────

/**
 * Detect duplicate emails within a set of staged records.
 * Returns records marked as "duplicate" with conflict info in errors.
 */
export function detectDuplicates(records: StagedInviteRecord[]): StagedInviteRecord[] {
    const idToRowLabel = new Map<number, string>();
    const emailMap = new Map<string, number[]>();

    for (let i = 0; i < records.length; i++) {
        const record = records[i];
        const id = record.id ?? -(i + 1);
        idToRowLabel.set(id, `row ${i + 1}`);

        const email = record.email.trim().toLowerCase();
        if (email) {
            const existing = emailMap.get(email) ?? [];
            existing.push(id);
            emailMap.set(email, existing);
        }
    }

    const conflictIds = new Set<number>();
    const idToMsg = new Map<number, string>();

    for (const [, ids] of emailMap) {
        if (ids.length <= 1) continue;
        for (const id of ids) {
            conflictIds.add(id);
            if (!idToMsg.has(id)) {
                const others = ids.filter((d) => d !== id);
                const labels = others.map((oid) => idToRowLabel.get(oid) ?? `#${oid}`).join(", ");
                idToMsg.set(id, `Duplicate email — also used in ${labels}`);
            }
        }
    }

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
export function validateAndDetectDuplicates(records: StagedInviteRecord[]): StagedInviteRecord[] {
    const validated = validateRecords(records);
    return detectDuplicates(validated);
}
