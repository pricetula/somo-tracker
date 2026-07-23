/**
 * React Query hooks for the Health feature.
 */
"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
    createMedicalIncident,
    deleteMedicalIncident,
    getHealthProfile,
    getMedicalIncident,
    getStudentHealth,
    listMedicalIncidents,
    updateMedicalIncident,
    upsertHealthProfile,
} from "@/lib/api/health";
import type { CreateMedicalIncidentPayload, UpsertHealthProfilePayload } from "@/lib/api/health";

// ─── Query Keys ───────────────────────────────────────────────────────────

export const healthKeys = {
    all: ["health"] as const,
    incidents: {
        all: ["health", "incidents"] as const,
        list: (params?: Record<string, unknown>) =>
            ["health", "incidents", "list", params] as const,
        detail: (id: string) => ["health", "incidents", id] as const,
    },
    profiles: {
        all: ["health", "profiles"] as const,
        byStudent: (studentId: string) => ["health", "profiles", studentId] as const,
    },
    studentHealth: (studentId: string) => ["health", "student", studentId] as const,
};

// ─── Queries ──────────────────────────────────────────────────────────────

/** List medical incidents, optionally scoped to a student or school. */
export function useMedicalIncidents(params?: {
    student_id?: string;
    school_id?: string;
    page?: number;
    limit?: number;
}) {
    return useQuery({
        queryKey: healthKeys.incidents.list(params),
        queryFn: () => listMedicalIncidents(params),
    });
}

/** Get a single medical incident by ID. */
export function useMedicalIncident(id: string) {
    return useQuery({
        queryKey: healthKeys.incidents.detail(id),
        queryFn: () => getMedicalIncident(id),
        enabled: !!id,
    });
}

/** Get a student's health profile. */
export function useHealthProfile(studentId: string) {
    return useQuery({
        queryKey: healthKeys.profiles.byStudent(studentId),
        queryFn: () => getHealthProfile(studentId),
        enabled: !!studentId,
    });
}

/** Get composite student health (profile + incidents). */
export function useStudentHealth(studentId: string) {
    return useQuery({
        queryKey: healthKeys.studentHealth(studentId),
        queryFn: () => getStudentHealth(studentId),
        enabled: !!studentId,
    });
}

// ─── Mutations ────────────────────────────────────────────────────────────

/** Create a new medical incident. */
export function useCreateMedicalIncident() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: (data: CreateMedicalIncidentPayload) => createMedicalIncident(data),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: healthKeys.incidents.all });
            queryClient.invalidateQueries({ queryKey: healthKeys.studentHealth("") });
        },
    });
}

/** Update a medical incident. */
export function useUpdateMedicalIncident(id: string) {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: (data: Parameters<typeof updateMedicalIncident>[1]) =>
            updateMedicalIncident(id, data),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: healthKeys.incidents.all });
        },
    });
}

/** Delete a medical incident. */
export function useDeleteMedicalIncident() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => deleteMedicalIncident(id),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: healthKeys.incidents.all });
        },
    });
}

/** Upsert a student's health profile. */
export function useUpsertHealthProfile(studentId: string) {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: (data: UpsertHealthProfilePayload) => upsertHealthProfile(studentId, data),
        onSuccess: () => {
            queryClient.invalidateQueries({
                queryKey: healthKeys.profiles.byStudent(studentId),
            });
            queryClient.invalidateQueries({
                queryKey: healthKeys.studentHealth(studentId),
            });
        },
    });
}
