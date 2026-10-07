/**
 * TypeScript interfaces for the School feature.
 *
 * Maps to backend internal/cbcschools/domain.go
 */

// ─── Domain types ─────────────────────────────────────────────────────────

export interface SchoolWithMemberCount {
    id: string;
    name: string;
    member_count: number;
}

export interface ListSchoolsResponse {
    items: SchoolWithMemberCount[];
    total: number;
    page: number;
    limit: number;
}

export interface CreateSchoolResponse {
    id: string;
    name: string;
    message: string;
}

// ─── Payload types ────────────────────────────────────────────────────────

export interface CreateSchoolPayload {
    name: string;
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
