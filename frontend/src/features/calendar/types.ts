// ─── Domain Types ───────────────────────────────────────────────────────────

/** A single academic period (e.g., Term 1, Term 2, Term 3) */
export interface AcademicPeriod {
  id?: string;
  name: string;
  start_date: string; // ISO date string
  end_date: string; // ISO date string
  is_final: boolean;
}

/** An academic year containing its periods */
export interface AcademicYear {
  id?: string;
  year: number;
  periods: AcademicPeriod[];
}

/** Payload sent when creating/updating the academic calendar */
export interface CreateAcademicCalendarPayload {
  year: number;
  periods: Omit<AcademicPeriod, "id">[];
}

// ─── Form Types ────────────────────────────────────────────────────────────

/** The shape of the dynamic form data used in react-hook-form */
export interface AcademicCalendarFormData {
  year: number;
  periods: {
    name: string;
    startDate: Date | undefined;
    endDate: Date | undefined;
    isFinal: boolean;
  }[];
}

// ─── Evaluator Result ─────────────────────────────────────────────────────

export type CalendarState =
  | { type: "loading" }
  | { type: "form"; mode: "setup" | "next-cycle" }
  | { type: "hidden"; alert?: "prep-mode" };
