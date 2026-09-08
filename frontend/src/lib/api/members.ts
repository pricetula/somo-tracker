/**
 * Members API functions.
 *
 * Endpoints:
 *   GET  /api/v1/members?role=...  — list members by role
 */

import { api } from "./client";
import type { Member, ListMembersResponse } from "./generated";

export type UserRole = "TEACHER" | "NURSE" | "FINANCE" | "SCHOOL_ADMIN";

/** Aggregate member counts returned by GET /api/v1/members/member-counts. */
export interface MemberCounts {
    students: number;
    admins: number;
    nurses: number;
    teachers: number;
    parents: number;
    finance: number;
}

/** Envelope returned by GET /api/v1/members/member-counts. */
export interface MemberCountsResponse {
    code: string;
    message: string;
    data: MemberCounts;
}

export const MappedUserRoles: Map<string, string> = new Map([
    ["TEACHER", "Teacher"],
    ["NURSE", "Nurse"],
    ["FINANCE", "Finance"],
    ["SCHOOL_ADMIN", "School Admin"],
]);

// ─── Re-export generated types ───────────────────────────────────────────

export type { Member, ListMembersResponse };

// ─── API Functions ─────────────────────────────────────────────────────────

/** Get a single member by ID. */
export async function getMember(userId: string): Promise<Member> {
    return Promise.resolve({
        id: "",
        full_name: "",
        email: "",
        role: "",
        is_active: true,
        tenant_id: "",
        created_at: "",
    });
}

/** Update a member's profile (full_name). */
export async function updateMember(userId: string, payload: { full_name: string }): Promise<void> {
    return undefined;
}

/** List members by role with pagination and optional search. */
export async function listMembers(
    role: UserRole,
    params: { page?: number; limit?: number; search?: string } = {}
): Promise<ListMembersResponse> {
    const searchParams = new URLSearchParams({ role });
    if (params.page) searchParams.set("page", String(params.page));
    if (params.limit) searchParams.set("limit", String(params.limit));
    if (params.search) searchParams.set("search", params.search);

    const qs = searchParams.toString();
    return { items: [], total: 0 };
}

/** Get aggregate member counts (students, admins, nurses, teachers, parents, finance). */
export async function getMemberCounts(): Promise<MemberCountsResponse> {
    return Promise.resolve({
        SCHOOL_ADMIN: 0,
        TEACHER: 0,
        STUDENT: 0,
        PARENT: 0,
        NURSE: 0,
        FINANCE: 0,
    });
}
