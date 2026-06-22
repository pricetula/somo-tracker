/**
 * Factory for building ImportDraftRow objects in tests.
 *
 * Usage:
 *   buildRow()                         // default valid row
 *   buildRow({ email: '' })           // missing email
 *   buildRow({ first_name: '', last_name: '' })  // missing name fields
 */

import type { ImportDraftRow } from "@/lib/db";

let counter = 0;

export function buildRow(overrides?: Partial<ImportDraftRow>): ImportDraftRow {
    counter++;
    return {
        temp_id: `test-row-id-${counter}-${crypto.randomUUID()}`,
        email: `teacher${counter}@school.edu`,
        first_name: "Jane",
        last_name: `Doe${counter}`,
        phone: "+254712345678",
        registration_number: `REG-${counter}`,
        ...overrides,
    };
}

/** Reset the internal counter (call in beforeEach if needed). */
export function resetRowCounter() {
    counter = 0;
}

/**
 * Build a batch of valid rows.
 */
export function buildRows(count: number, overrides?: Partial<ImportDraftRow>): ImportDraftRow[] {
    return Array.from({ length: count }, () => buildRow(overrides));
}
