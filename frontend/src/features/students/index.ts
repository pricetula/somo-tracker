/**
 * Students feature — public API barrel.
 */

export { StudentsTable } from "./components/students-table";
export { StudentProfileCard } from "./components/student-profile-card";
export { StudentForm } from "./components/student-form";
export { EnrollmentTimeline } from "./components/enrollment-timeline";
export { EnrollDialog } from "./components/enroll-dialog";

export { useStudents, studentKeys } from "./hooks/use-students";
export {
    useStudentDetail,
    useCreateStudent,
    useUpdateStudent,
    useCreateEnrollment,
} from "./hooks/use-student-detail";
export { listStudents } from "./services/students-api";

export type {
    Student,
    StudentDetail,
    Enrollment,
    ListStudentsResponse,
    ListStudentsParams,
    CreateStudentPayload,
    UpdateStudentPayload,
    CreateEnrollmentPayload,
} from "./types";
