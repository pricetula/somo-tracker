/**
 * Students feature — public API barrel.
 */

export { StudentDetailContent } from "./components/student-detail-content";
export { StudentProfileCard } from "./components/student-profile-card";
export { StudentForm } from "./components/student-form";
export { EnrollmentTimeline } from "./components/enrollment-timeline";
export { EnrollDialog } from "./components/enroll-dialog";
export { BatchEnrollForm } from "./components/batch-enroll-form";
export { useEnrollmentStore } from "./store/enrollment-store";
export { StudentsList } from "./components/student-list";

export { useStudents, useStudentMap, useDeleteStudent, studentKeys } from "./hooks/use-students";
export {
    useStudentDetail,
    useCreateStudent,
    useCreateStudents,
    useUpdateStudent,
    useCreateEnrollment,
    useBatchEnrollStudents,
} from "./hooks/use-student-detail";

export type {
    Student,
    StudentDetail,
    Enrollment,
    LinkedParent,
    ListStudentsResponse,
    ListStudentsParams,
    CreateStudentPayload,
    CreateStudentsPayload,
    CreateStudentsResponse,
    UpdateStudentPayload,
    CreateEnrollmentPayload,
    BatchEnrollItem,
    BatchEnrollRequest,
    BatchEnrollResponse,
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
