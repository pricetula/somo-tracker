"use client";

import { useQuery } from "@tanstack/react-query";
import { listClasses, type Class } from "@/lib/api/classes";
import { listTeachers, type TeacherMember } from "@/lib/api/teachers";
import { listLearningAreas, type LearningArea } from "@/lib/api/curriculum";

// ─── Query keys ───────────────────────────────────────────────────────────

export const timetableDataKeys = {
    classes: ["timetable", "classes"] as const,
    teachers: ["timetable", "teachers"] as const,
    learningAreas: ["timetable", "learning-areas"] as const,
    rooms: ["timetable", "rooms"] as const,
};

/**
 * Fetch all classes for the active school.
 * Returns array of { id, name } for Select components.
 */
export function useTimetableClasses() {
    return useQuery({
        queryKey: timetableDataKeys.classes,
        queryFn: async () => {
            const response = await listClasses({ limit: 500 });
            return response.items.map((c: Class) => ({
                id: c.id,
                name: c.display_label,
            }));
        },
        staleTime: 5 * 60 * 1000, // 5 minutes
    });
}

/**
 * Fetch all teachers for the active school.
 * Returns array of { id, name } for Select components.
 */
export function useTimetableTeachers() {
    return useQuery({
        queryKey: timetableDataKeys.teachers,
        queryFn: async () => {
            const response = await listTeachers({ limit: 500 });
            return response.items.map((t: TeacherMember) => ({
                id: t.id,
                name: t.full_name,
            }));
        },
        staleTime: 5 * 60 * 1000, // 5 minutes
    });
}

/**
 * Fetch all learning areas for the active school.
 * Returns array of { id, name } for Select components.
 */
export function useTimetableLearningAreas() {
    return useQuery({
        queryKey: timetableDataKeys.learningAreas,
        queryFn: async () => {
            const response = await listLearningAreas({ limit: 500 });
            return response.items.map((la: LearningArea) => ({
                id: la.id,
                name: la.name,
            }));
        },
        staleTime: 5 * 60 * 1000, // 5 minutes
    });
}

/**
 * Fetch unique room identifiers from enriched slots.
 * This uses the slots data already fetched by useTimetableGrid.
 */
export function useTimetableRooms(slots: Array<{ room_identifier?: string | null }>) {
    return useQuery({
        queryKey: timetableDataKeys.rooms,
        queryFn: async () => {
            const uniqueRooms = new Set<string>();
            for (const slot of slots) {
                if (slot.room_identifier) {
                    uniqueRooms.add(slot.room_identifier);
                }
            }
            return Array.from(uniqueRooms).map((identifier) => ({ id: identifier, identifier }));
        },
        enabled: slots.length > 0,
        staleTime: 5 * 60 * 1000, // 5 minutes
    });
}
