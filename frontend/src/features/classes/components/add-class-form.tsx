/**
 * AddClassForm — Creates a new class.
 *
 * Uses reusable comboboxes from their respective feature modules:
 *  - GradeLevelCombobox from grade-level feature
 *  - StreamCombobox from streams feature
 *  - AcademicYearCombobox from academic-terms feature
 *
 * The academic_term_id is resolved server-side from the current active term.
 */

"use client";

import { GradeLevelCombobox } from "@/features/grade-level";

// ─── Props ─────────────────────────────────────────────────────────────────

interface AddClassFormProps {
    /** Called when the class is successfully created. */
    onSuccess?: (cls: Class) => void;
}

// ─── Component ─────────────────────────────────────────────────────────────

export function AddClassForm({ onSuccess: _onSuccess }: AddClassFormProps) {
    // TODO: Implement full form UI
    return <GradeLevelCombobox />;
}

// Type-only import for props interface
import type { Class } from "@/lib/api/classes";
