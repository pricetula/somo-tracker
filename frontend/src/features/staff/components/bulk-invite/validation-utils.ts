/**
 * Validation utilities for the Bulk Invite form.
 * Runs client-side before submission.
 */

// ============================================================================
// Types
// ============================================================================

export interface InviteRowInput {
    email: string;
    full_name: string; // empty string when not provided
}

export interface InviteRowError {
    rowIndex: number;
    field: "email" | "full_name";
    message: string;
}

// ============================================================================
// Constants
// ============================================================================

export const EMAIL_REGEX = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

// ============================================================================
// Validation
// ============================================================================

/**
 * Validates a single invite row.
 * Returns an array of error messages (empty = valid).
 */
export function validateInviteRow(row: InviteRowInput, rowIndex: number): InviteRowError[] {
    const errors: InviteRowError[] = [];
    const email = row.email.trim();

    if (!email) {
        errors.push({ rowIndex, field: "email", message: "Email is required" });
    } else if (!EMAIL_REGEX.test(email)) {
        errors.push({ rowIndex, field: "email", message: "Invalid email format" });
    }

    return errors;
}

/**
 * Detects duplicate emails within the batch.
 * Returns errors for rows that share the same email with an earlier row.
 */
export function detectDuplicateEmails(rows: InviteRowInput[]): InviteRowError[] {
    const errors: InviteRowError[] = [];
    const seen = new Map<string, number[]>(); // email → row indices

    rows.forEach((row, i) => {
        const email = row.email.trim().toLowerCase();
        if (!email) return;

        const existing = seen.get(email) || [];
        existing.push(i);
        seen.set(email, existing);
    });

    for (const [, indices] of seen) {
        if (indices.length > 1) {
            // All rows after the first are duplicates
            for (let k = 1; k < indices.length; k++) {
                errors.push({
                    rowIndex: indices[k],
                    field: "email",
                    message: `Duplicate email — also used in row ${indices[0] + 1}`,
                });
            }
        }
    }

    return errors;
}

/**
 * Validates the entire batch.
 * Combines per-row validation and duplicate detection.
 */
export function validateBatch(rows: InviteRowInput[]): InviteRowError[] {
    const errors: InviteRowError[] = [];

    rows.forEach((row, i) => {
        errors.push(...validateInviteRow(row, i));
    });

    errors.push(...detectDuplicateEmails(rows));

    return errors;
}
