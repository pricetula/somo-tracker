export type StudentListItem = {
    student_id: string;
    admission_number: string;
    full_name: string;
    date_of_birth: string;
    gender: string;
    class_id?: string;
    class_name?: string;
    grade_level?: string;
    guardian_linked: boolean;
    attendance_rate?: number;
};

export type StudentProfile = {
    student_id: string;
    admission_number: string;
    nemis_upn?: string;
    full_name: string;
    date_of_birth: string;
    gender: string;
    metadata?: Record<string, unknown>;
};

export type UpdateStudentRequest = {
    full_name?: string;
    date_of_birth?: string;
    gender?: string;
    nemis_upn?: string;
    metadata?: Record<string, unknown>;
};
