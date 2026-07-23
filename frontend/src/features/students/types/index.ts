/**
 * TypeScript interfaces for the Students feature.
 *
 * Types are defined in src/lib/api/students.ts and re-exported here
 * so the feature barrel remains the single import entry point.
 */

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
} from "@/lib/api/students";
