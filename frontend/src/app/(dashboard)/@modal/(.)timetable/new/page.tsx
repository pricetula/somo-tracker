import { CreateTimetableDialog } from "@/features/timetable";

/**
 * Intercepted `@modal` slot for `/timetable/new`.
 *
 * Matches only during client-side navigation, so the CreateTimetableDialog
 * renders in a dialog. On a hard navigation/refresh this intercept is skipped
 * and `@modal/default.tsx` renders, leaving `/timetable/new` to render
 * the full-page form (if implemented).
 */
export default function TimetableCreateModal() {
    return <CreateTimetableDialog />;
}
