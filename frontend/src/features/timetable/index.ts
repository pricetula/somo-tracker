// Public API for timetable feature

export { TimetableTemplateWizard } from "./components/timetable-template-wizard";
export { TimeSlotRow } from "./components/time-slot-row";
export { WeeklyMatrix } from "./components/weekly-matrix";
export { TimetableGrid } from "./components/timetable-grid";
export { TimetableDetail } from "./components/timetable-detail";
export { useTimeSlots } from "./hooks/use-time-slots";
export { useTimetableTemplate } from "./hooks/use-timetable-template";
export { useUpdateTimetableTemplate } from "./hooks/use-timetable-templates";
export type { TimeSlotResponse } from "./services/timetable-api";
