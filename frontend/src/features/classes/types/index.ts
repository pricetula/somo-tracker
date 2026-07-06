/**
 * Classes feature — type definitions.
 *
 * Re-exports the canonical Class type from the generated API client
 * and defines feature‑specific types for the ClassCombobox.
 */

import type { Class } from "@/lib/api/generated";

export type { Class };

/** Minimal shape expected by the combobox for rendering options. */
export interface ClassOption {
    value: string; // Class.id
    label: string; // Class.display_label
}
