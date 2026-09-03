/**
 * Enrollment store — holds selected student IDs for batch enrollment.
 *
 * The Students page sets these IDs when the user clicks the
 * "Enroll Selected Students" button, then navigates to the enroll form.
 * The enrolled page consumes the IDs and clears them on success or cancel.
 */

import { create } from "zustand";

interface EnrollmentState {
    /** Student IDs selected for batch enrollment. */
    selectedStudentIds: string[];
    /** Set the selected student IDs before navigating to the enroll form. */
    setSelectedStudentIds: (ids: string[]) => void;
    /** Clear selected student IDs (on success, cancel, or unmount). */
    clearSelectedStudentIds: () => void;
}

export const useEnrollmentStore = create<EnrollmentState>((set) => ({
    selectedStudentIds: [],
    setSelectedStudentIds: (ids) => set({ selectedStudentIds: ids }),
    clearSelectedStudentIds: () => {
        set({ selectedStudentIds: [] });
    },
}));
