// Public API for timetable feature

export { TimetableTemplateWizard } from "./components/timetable-template-wizard";
export { TimeSlotRow } from "./components/time-slot-row";
export { WeeklyMatrix } from "./components/weekly-matrix";
export { TimetableGrid } from "./components/timetable-grid";
export { TimetableDetail } from "./components/timetable-detail";
export { SubstitutionsTable } from "./components/substitutions-table";
export { SubstitutionForm } from "./components/substitution-form";
export { SubstitutionDetail } from "./components/substitution-detail";
export { useTimeSlots } from "./hooks/use-time-slots";
export { useTimetableTemplate } from "./hooks/use-timetable-template";
export { useUpdateTimetableTemplate } from "./hooks/use-timetable-templates";
export { useSubstitutions } from "./hooks/use-substitutions";
export { useCreateSubstitution } from "./hooks/use-substitutions";
export { useUpdateSubstitution } from "./hooks/use-substitutions";
export { useDeleteSubstitution } from "./hooks/use-substitutions";
export type { TimeSlotResponse } from "./services/timetable-api";
export type {
    TimetableSubstitution,
    ListSubstitutionsParams,
    CreateSubstitutionRequest,
    UpdateSubstitutionRequest,
    SubstitutionStatus,
} from "./types/substitution";
