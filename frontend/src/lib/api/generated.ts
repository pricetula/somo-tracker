// Auto-generated API types derived from the Somotracker Go backend.
// Generated from: backend/internal/*/handler.go, domain.go.
// Generator: openapi-typescript (planned) -- manually maintained for now.
// Run `pnpm generate:api` to regenerate from the backend swagger.json.

export interface DiscoveryPayload {
    email: string;
}

export interface VerifyResponse {
    session_ref: string;
}

export interface ExistingUserVerifyResponse {
    session_token: string;
    role: string;
    email: string;
}

export interface RegistrationPayload {
    school_name: string;
    session_ref: string;
    full_name: string;
}

export interface MeResponse {
    user_id: string;
    tenant_id: string;
    role: string;
    school_id: string;
    school_name: string;
    full_name: string;
    email: string;
}

export interface CreateTenantPayload {
    name: string;
    slug?: string;
}

export interface Tenant {
    id: string;
    name: string;
    slug: string;
    created_at: string;
}

export interface CreateSchoolPayload {
    name: string;
}

export interface CreateSchoolResponse {
    id: string;
}

export interface UpdateSchoolPayload {
    name?: string;
    county?: string;
    sub_county?: string;
    ward?: string;
    knec_school_code?: string;
    nemis_code?: string;
    school_type?: string;
    is_active?: boolean;
}

export interface SchoolWithMemberCount {
    id: string;
    tenant_id: string;
    name: string;
    knec_school_code?: string;
    county: string;
    sub_county: string;
    ward?: string;
    school_type: string;
    is_active: boolean;
    created_at: string;
    updated_at: string;
    admins: number;
    teachers: number;
    nurses: number;
    finance: number;
    parents: number;
    students: number;
    is_member_active_school: boolean;
}

export interface ListSchoolsResponse {
    schools: SchoolWithMemberCount[];
    total: number;
}

export interface CreateStreamPayload {
    name: string;
}

export interface UpdateStreamPayload {
    name: string;
}

export interface Stream {
    id: string;
    name: string;
    created_at: string;
    updated_at: string;
}

export interface ListStreamsResponse {
    data: Stream[];
}

export interface CreateClassPayload {
    grade_level: string;
    academic_year_id: string;
    academic_term_id: string;
    stream_id: string;
    student_ids?: string[];
}

export interface UpdateClassPayload {
    grade_level: string;
    stream_id: string;
    academic_term_id: string;
    student_ids?: string[];
}

export interface BulkDeleteClassesPayload {
    class_ids: string[];
}

export interface Class {
    id: string;
    grade_level: string;
    stream_name: string;
    display_label: string;
    stream_id: string;
    student_count?: number;
    created_at?: string;
    updated_at?: string;
}

export interface ClassListResult {
    data: Class[];
    total_records: number;
    current_page: number;
    limit: number;
    total_pages: number;
}

export interface Member {
    id: string;
    email: string;
    full_name: string;
    role: "TEACHER" | "NURSE" | "FINANCE";
    is_active: boolean;
    created_at: string;
}

export interface ListMembersResponse {
    members: Member[];
    total: number;
}

export type InvitationStatus = "pending" | "accepted" | "expired" | "revoked" | "invite_failed";

export type InvitationRole = "SYSTEM_ADMIN" | "SCHOOL_ADMIN" | "TEACHER" | "NURSE" | "FINANCE";

export interface Invitation {
    id: string;
    school_id: string;
    tenant_id: string;
    email: string;
    role: InvitationRole;
    status: InvitationStatus;
    full_name?: string;
    expires_at: string;
    created_at: string;
}

export interface ListInvitationsResponse {
    invitations: Invitation[];
    total: number;
}

export interface SwitchActiveSchoolPayload {
    school_id: string;
}

export interface ActiveSchoolResponse {
    school_id: string;
}

export interface ActiveSchoolUpdateResponse {
    message: string;
}

export interface AcademicYear {
    id: string;
    name: string;
    start_date: string;
    end_date: string;
    is_current: boolean;
    version: number;
    created_at: string;
    updated_at: string;
    terms: AcademicTerm[];
}

export interface AcademicTerm {
    id: string;
    academic_year_id: string;
    name: string;
    term_number: number;
    start_date: string;
    end_date: string;
    is_current: boolean;
    is_final: boolean;
    version: number;
    created_at: string;
    updated_at: string;
}

export interface PatchYearBody {
    name?: string;
    start_date?: string;
    end_date?: string;
    version: number;
}

export interface PatchYearResponse {
    id: string;
    name: string;
    start_date: string;
    end_date: string;
    is_current: boolean;
    version: number;
    warnings?: string[];
}

export interface SetCurrentYearResponse {
    message: string;
}

export interface CreateTermBody {
    academic_year_id: string;
    name: string;
    term_number: number;
    start_date: string;
    end_date: string;
}

export interface PatchTermBody {
    name?: string;
    start_date?: string;
    end_date?: string;
    version: number;
}

export interface PatchTermResponse {
    id: string;
    name: string;
    term_number: number;
    start_date: string;
    end_date: string;
    is_current: boolean;
    academic_year_id: string;
    version: number;
    warnings?: string[];
}

export interface ListYearsResponse {
    data: AcademicYear[];
}

export interface ListTermsResponse {
    data: AcademicTerm[];
}

export type TeacherRole = "PRIMARY_CLASS_TEACHER" | "SUBJECT_TEACHER" | "SUBSTITUTE_TEACHER";

export interface CreateTimetableSlotInput {
    class_id: string;
    teacher_id: string;
    learning_area_id?: string;
    room_identifier?: string;
    day_of_week: number;
    start_time: string;
    end_time: string;
}

export interface BulkCreateTimetableSlotsInput {
    academic_year_id: string;
    academic_term_id: string;
    slots: CreateTimetableSlotInput[];
}

export interface TimetableSlot {
    id: string;
    tenant_id: string;
    school_id: string;
    academic_year_id: string;
    academic_term_id: string;
    class_id: string;
    teacher_id: string;
    learning_area_id?: string;
    room_identifier?: string;
    day_of_week: number;
    start_time: string;
    end_time: string;
}

export interface ListTimetableSlotsResponse {
    data: TimetableSlot[];
}

export interface AssignTeacherPayload {
    user_id: string;
    learning_area_id?: string;
    teacher_role: TeacherRole;
}

export interface AssignTeacherResponse {
    code: string;
    message: string;
}

export interface RemoveTeacherResponse {
    code: string;
    message: string;
}

export interface BulkCreateSlotsResponse {
    code: string;
    message: string;
}

export type AttendanceStatus = "PRESENT" | "ABSENT" | "LATE" | "EXCUSED";

export interface AttendanceLogInput {
    student_id: string;
    status: AttendanceStatus;
    remarks?: string;
}

export interface MarkAttendanceInput {
    academic_term_id: string;
    class_id: string;
    learning_area_id: string;
    date: string;
    period_id?: string;
    students: AttendanceLogInput[];
}

export interface AttendancePeriod {
    id: string;
    tenant_id: string;
    school_id: string;
    academic_term_id: string;
    class_id: string;
    learning_area_id: string;
    date_recorded: string;
    recorded_by: string;
    authorized_by_role?: TeacherRole;
    created_at: string;
}

export interface AttendanceLog {
    id: string;
    tenant_id: string;
    attendance_period_id: string;
    student_id: string;
    status: AttendanceStatus;
    remarks?: string;
    recorded_by: string;
}

export interface GetAttendancePeriodResponse {
    period: AttendancePeriod;
    logs: AttendanceLog[];
}

export interface SubmitAttendanceResponse {
    code: string;
    message: string;
}

export interface ImportStaffRecord {
    temp_id: string;
    email: string;
    full_name: string;
    phone?: string;
    registration_number?: string;
}

export interface StartImportRequest {
    role: "SCHOOL_ADMIN" | "NURSE" | "FINANCE" | "TEACHER";
    records: ImportStaffRecord[];
    parent_import_job_id?: string;
}

export interface StartImportResponse {
    import_job_id: string;
    status: string;
    total: number;
}

export interface ImportJob {
    id: string;
    tenant_id: string;
    school_id: string;
    role: string;
    created_by?: string;
    status: string;
    total_records: number;
    processed_records: number;
    success_count: number;
    failed_count: number;
    parent_import_job_id?: string;
    created_at: string;
    started_at?: string;
    completed_at?: string;
}

export interface TrackImportResponse {
    job: ImportJob;
    failed_records: number;
}

export interface ImportProgressEvent {
    type: "connected" | "import_progress" | "import_finished" | "import_error";
    import_job_id: string;
    status?: string;
    processed_records?: number;
    success_count?: number;
    failed_count?: number;
    total_records?: number;
}

export interface FailedInvitation {
    id: string;
    email: string;
    full_name?: string;
    phone?: string;
    error_message?: string;
}

export interface ListFailedInvitationsResponse {
    invitations: FailedInvitation[];
}

export interface StudentRecord {
    full_name: string;
    gender: string;
    date_of_birth?: string;
    upi_number?: string;
    knec_assessment_number?: string;
    cbc_student_parents_id?: string;
    class_id?: string;
}

export interface StartStudentImportRequest {
    academic_year: string;
    term: string;
    students: StudentRecord[];
}

export interface StartStudentImportResponse {
    job_id: string;
    status: string;
}

export interface ProgressFrame {
    status: string;
    processed: number;
    total: number;
    success_count: number;
    failed_count: number;
}

export interface ParentRecord {
    id: string;
    full_name: string;
    phone?: string;
    email?: string;
}

export interface ClassRecord {
    id: string;
    name: string;
    grade_level: string;
    stream_name: string;
    display_label: string;
}

export interface ExistingStudentRecord {
    full_name: string;
    date_of_birth?: string;
    upi_number?: string;
}

export interface AcademicYearRecord {
    id: string;
    name: string;
    start_date: string;
    end_date: string;
    is_current: boolean;
}

export interface AcademicPeriodRecord {
    id: string;
    name: string;
    term_number: number;
    start_date: string;
    end_date: string;
    is_current: boolean;
}

export interface ApiErrorBody {
    code: string;
    message: string;
    errors?: Record<string, string[]>;
}
