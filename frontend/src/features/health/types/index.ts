/**
 * TypeScript interfaces for the Health feature.
 *
 * Maps to backend/internal/health/domain.go
 */

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

export interface MedicalIncidentListResponse {
    items: MedicalIncident[];
    total: number;
}

export interface CreateMedicalIncidentPayload {
    student_id: string;
    incident_timestamp?: string;
    symptoms: string;
    action_taken: string;
}

export interface UpsertHealthProfilePayload {
    blood_group?: string | null;
    allergies?: string[];
    chronic_conditions?: string[];
    emergency_instructions?: string | null;
}
