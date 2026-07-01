/**
 * TypeScript interfaces for the Parents feature.
 *
 * Maps to backend internal/parents/domain.go
 */

export interface Parent {
    id: string;
    user_id: string;
    full_name: string;
    email: string;
    phone_number: string;
    is_active: boolean;
    created_at: string;
}

export interface StudentLink {
    student_id: string;
    full_name: string;
    relationship?: string | null;
    is_primary: boolean;
}

export interface ParentDetail {
    id: string;
    user_id: string;
    full_name: string;
    email: string;
    phone_number: string;
    is_active: boolean;
    created_at: string;
    linked_students: StudentLink[];
}

export interface ListParentsResponse {
    data: Parent[];
}

export interface ParentDetailResponse {
    data: ParentDetail;
}

export interface CreateParentPayload {
    email: string;
    full_name: string;
    phone_number: string;
}

export interface UpdateParentPayload {
    phone_number?: string;
    is_active?: boolean;
}

export interface LinkStudentPayload {
    student_id: string;
    relationship?: string | null;
    is_primary?: boolean;
}
