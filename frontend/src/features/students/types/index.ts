/**
 * TypeScript interfaces for the Students feature.
 */

export interface Student {
    id: string;
    full_name: string;
    gender: string;
    date_of_birth?: string | null;
    upi_number?: string | null;
    knec_assessment_number?: string | null;
    class_name?: string | null;
    class_id?: string | null;
    is_active: boolean;
    created_at: string;
}

export interface ListStudentsResponse {
    students: Student[];
    total: number;
    page: number;
    limit: number;
}

export interface ListStudentsParams {
    page?: number;
    limit?: number;
    search?: string;
    class_id?: string;
    gender?: string;
}
