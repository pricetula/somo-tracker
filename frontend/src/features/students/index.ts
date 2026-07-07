/**
 * Students feature — public API barrel.
 */

export { StudentProfileCard } from "./components/student-profile-card";
export { StudentForm } from "./components/student-form";
export { EnrollmentTimeline } from "./components/enrollment-timeline";
export { EnrollDialog } from "./components/enroll-dialog";

export { useStudents, useDeleteStudent, studentKeys } from "./hooks/use-students";
export {
    useStudentDetail,
    useCreateStudent,
    useCreateStudents,
    useUpdateStudent,
    useCreateEnrollment,
} from "./hooks/use-student-detail";

export { listStudents } from "./services/students-api";
export { createStudents } from "@/lib/api/students";

export type {
    Student,
    StudentDetail,
    Enrollment,
    ListStudentsResponse,
    ListStudentsParams,
    CreateStudentPayload,
    CreateStudentsPayload,
    CreateStudentsResponse,
    UpdateStudentPayload,
    CreateEnrollmentPayload,
} from "./types";

export type {
    ImportRequest,
    ImportResponse,
    ImportJob,
    ImportJobStatus,
    ImportFailureType,
    ImportProgressEvent,
    ImportRowFailure,
    ImportRow,
} from "@/lib/api/imports";
