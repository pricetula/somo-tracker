/**
 * Validation utilities for staff import.
 *
 * Shared by BulkStaffImport, ManualEntryPanel, and review/sub-components.
 */

import { isValidPhoneNumber, parsePhoneNumber } from "libphonenumber-js";
import { useVirtualizer } from "@tanstack/react-virtual";
import type { ImportDraftRow } from "@/lib/db";

// Re-export for sub-components that need them
export { useVirtualizer, isValidPhoneNumber, parsePhoneNumber };

/** Check if email has valid '@' structural layout. */
export function hasValidEmailStructure(email: string): boolean {
    if (!email || !email.includes("@")) return false;
    const parts = email.split("@");
    if (parts.length !== 2) return false;
    const [local, domain] = parts;
    if (local.length === 0 || domain.length === 0) return false;
    if (!domain.includes(".")) return false;
    return true;
}

/** Normalize phone to E.164 with default country KE. Returns null if unparseable. */
export function normalizePhone(phone: string): string | null {
    if (!phone || phone.trim() === "") return null;
    const cleaned = phone.trim();
    try {
        if (isValidPhoneNumber(cleaned, "KE")) {
            return parsePhoneNumber(cleaned, "KE")!.format("E.164");
        }
        // Try parsing anyway
        const parsed = parsePhoneNumber(cleaned, "KE");
        if (parsed && isValidPhoneNumber(cleaned, "KE")) {
            return parsed.format("E.164");
        }
        return null;
    } catch {
        return null;
    }
}

/** Count critical errors (block submission). Empty rows (no email) are skipped. */
export function getCriticalErrorCount(rows: ImportDraftRow[]): number {
    let errors = 0;
    const emails = new Set<string>();
    for (const row of rows) {
        const email = row.email.trim();
        if (!email) continue; // skip empty rows

        if (!row.first_name) errors++;
        if (!row.last_name) errors++;
        if (!hasValidEmailStructure(email)) errors++;
        const lowerEmail = email.toLowerCase();
        if (emails.has(lowerEmail)) errors++;
        emails.add(lowerEmail);
    }
    return errors;
}
