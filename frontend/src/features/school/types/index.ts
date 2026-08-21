/**
 * TypeScript interfaces for the School feature.
 *
 * Maps to backend internal/cbcschools/domain.go
 */

import type {
    SchoolWithMemberCount as ApiSchoolWithMemberCount,
    CreateSchoolPayload as ApiCreateSchoolPayload,
    CreateSchoolResponse,
    ListSchoolsResponse,
} from "@/lib/api/generated";

// ─── Domain types ─────────────────────────────────────────────────────────

export type SchoolWithMemberCount = ApiSchoolWithMemberCount;
export type { ListSchoolsResponse, CreateSchoolResponse };

// ─── Payload types ────────────────────────────────────────────────────────

export type CreateSchoolPayload = ApiCreateSchoolPayload;

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
