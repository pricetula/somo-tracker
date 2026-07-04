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
