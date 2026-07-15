/**
 * Health API functions.
 *
 * Endpoints (from backend/internal/health/handler.go):
 *   POST   /api/v1/health/incidents              — log medical incident
 *   GET    /api/v1/health/incidents/:id           — get incident detail
 *   GET    /api/v1/health/incidents?student_id=   — list by student
 *   GET    /api/v1/health/incidents?school_id=    — list by school
 *   PUT    /api/v1/health/incidents/:id           — update incident
 *   DELETE /api/v1/health/incidents/:id           — delete incident
 *   PUT    /api/v1/health/profiles/:studentId     — upsert health profile
 *   GET    /api/v1/health/profiles/:studentId     — get health profile
 *   GET    /api/v1/health/students/:studentId     — composite (profile + incidents)
 */

import { api } from "./client";

// ─── Domain Types ─────────────────────────────────────────────────────────

export interface MedicalIncident {
    id: string;
    student_id: string;
    incident_timestamp: string;
    symptoms: string;
    action_taken: string;
    logged_by: string;
    logged_by_name?: string;
    created_at: string;
    updated_at: string;
}

export interface StudentHealthProfile {
    id: string;
    student_id: string;
    blood_group?: string | null;
    allergies?: string[];
    chronic_conditions?: string[];
    emergency_instructions?: string | null;
    created_at: string;
    updated_at: string;
}

export interface StudentHealthResponse {
    profile: StudentHealthProfile | null;
    incidents: MedicalIncident[];
}

// ─── Response Types ───────────────────────────────────────────────────────

export interface MedicalIncidentListResponse {
    items: MedicalIncident[];
    total: number;
}

export interface HealthProfileResponse {
    data: StudentHealthProfile;
}

// ─── Payload Types ────────────────────────────────────────────────────────

export interface CreateMedicalIncidentPayload {
    student_id: string;
    incident_timestamp?: string;
    symptoms: string;
    action_taken: string;
}

export interface UpdateMedicalIncidentPayload {
    symptoms?: string;
    action_taken?: string;
}

export interface UpsertHealthProfilePayload {
    blood_group?: string | null;
    allergies?: string[];
    chronic_conditions?: string[];
    emergency_instructions?: string | null;
}

// ─── API Functions ─────────────────────────────────────────────────────────

/** Log a new medical incident. */
export async function createMedicalIncident(
    data: CreateMedicalIncidentPayload
): Promise<MedicalIncident> {
    return api.post<MedicalIncident>("/api/v1/health/incidents", data);
}

/** Get a medical incident by ID. */
export async function getMedicalIncident(id: string): Promise<MedicalIncident> {
    return api.get<MedicalIncident>(`/api/v1/health/incidents/${id}`);
}

/** List medical incidents by student or school. */
export async function listMedicalIncidents(
    params: {
        student_id?: string;
        school_id?: string;
        page?: number;
        limit?: number;
    } = {}
): Promise<MedicalIncidentListResponse> {
    const searchParams = new URLSearchParams();
    if (params.student_id) searchParams.set("student_id", params.student_id);
    if (params.school_id) searchParams.set("school_id", params.school_id);
    if (params.page) searchParams.set("page", String(params.page));
    if (params.limit) searchParams.set("limit", String(params.limit));
    const qs = searchParams.toString();
    return api.get<MedicalIncidentListResponse>(`/api/v1/health/incidents?${qs}`);
}

/** Update a medical incident. */
export async function updateMedicalIncident(
    id: string,
    data: UpdateMedicalIncidentPayload
): Promise<void> {
    return api.put<void>(`/api/v1/health/incidents/${id}`, data);
}

/** Delete a medical incident. */
export async function deleteMedicalIncident(id: string): Promise<void> {
    return api.delete<void>(`/api/v1/health/incidents/${id}`);
}

/** Upsert a student health profile. */
export async function upsertHealthProfile(
    studentId: string,
    data: UpsertHealthProfilePayload
): Promise<StudentHealthProfile> {
    return api.put<StudentHealthProfile>(`/api/v1/health/profiles/${studentId}`, data);
}

/** Get a student health profile. */
export async function getHealthProfile(studentId: string): Promise<StudentHealthProfile> {
    return api.get<StudentHealthProfile>(`/api/v1/health/profiles/${studentId}`);
}

/** Get composite student health (profile + recent incidents). */
export async function getStudentHealth(studentId: string): Promise<StudentHealthResponse> {
    return api.get<StudentHealthResponse>(`/api/v1/health/students/${studentId}`);
}
