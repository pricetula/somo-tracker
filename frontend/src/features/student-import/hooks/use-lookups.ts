/**
 * Phase 1: Parallel background lookup hooks for parents, classes, and existing students.
 *
 * Each lookup is independent — failures degrade gracefully with warnings + retry.
 */

"use client";

import * as React from "react";
import { getErrorMessage } from "@/lib/errors";
import { fetchParents, fetchClasses, fetchExistingStudents } from "../services/import-api";
import type { ParentsMap, ClassesMap, ParentRecord, ClassRecord, ExistingStudent } from "../types";

// ─── Parent Lookup ────────────────────────────────────────────────────────

export interface ParentLookupState {
    parentsMap: ParentsMap;
    parentsError: string | null;
    parentsLoading: boolean;
    retryParents: () => void;
}

export function useParentLookup(): ParentLookupState {
    const [parentsMap, setParentsMap] = React.useState<ParentsMap>(new Map());
    const [parentsError, setParentsError] = React.useState<string | null>(null);
    const [parentsLoading, setParentsLoading] = React.useState(true);
    const [retryCount, setRetryCount] = React.useState(0);

    React.useEffect(() => {
        let cancelled = false;

        fetchParents()
            .then((data: ParentRecord[]) => {
                if (cancelled) return;
                const map: ParentsMap = new Map();
                for (const p of data) {
                    const key = p.full_name.toLowerCase().replace(/\s+/g, "");
                    map.set(key, p);
                }
                setParentsMap(map);
                setParentsError(null);
                setParentsLoading(false);
            })
            .catch((err: unknown) => {
                if (cancelled) return;
                setParentsMap(new Map());
                setParentsError(getErrorMessage(err));
                setParentsLoading(false);
            });

        return () => {
            cancelled = true;
        };
    }, [retryCount]);

    const retryParents = React.useCallback(() => {
        setParentsLoading(true);
        setParentsError(null);
        setRetryCount((c) => c + 1);
    }, []);

    return { parentsMap, parentsError, parentsLoading, retryParents };
}

// ─── Class Lookup ─────────────────────────────────────────────────────────

export interface ClassLookupState {
    classesMap: ClassesMap;
    classesError: string | null;
    classesLoading: boolean;
    retryClasses: () => void;
}

export function useClassLookup(): ClassLookupState {
    const [classesMap, setClassesMap] = React.useState<ClassesMap>(new Map());
    const [classesError, setClassesError] = React.useState<string | null>(null);
    const [classesLoading, setClassesLoading] = React.useState(true);
    const [retryCount, setRetryCount] = React.useState(0);

    React.useEffect(() => {
        let cancelled = false;

        fetchClasses()
            .then((data: ClassRecord[]) => {
                if (cancelled) return;
                const map: ClassesMap = new Map();
                for (const c of data) {
                    // Normalize with the same function used during mapping
                    const normalized = c.name
                        .toLowerCase()
                        .trim()
                        .replace(/^(class|grade|std|form)\s+/, "")
                        .replace(/^g\.?\s*/, "")
                        .replace(/\s+/g, "");
                    map.set(normalized, c);
                }
                setClassesMap(map);
                setClassesError(null);
                setClassesLoading(false);
            })
            .catch((err: unknown) => {
                if (cancelled) return;
                setClassesMap(new Map());
                setClassesError(getErrorMessage(err));
                setClassesLoading(false);
            });

        return () => {
            cancelled = true;
        };
    }, [retryCount]);

    const retryClasses = React.useCallback(() => {
        setClassesLoading(true);
        setClassesError(null);
        setRetryCount((c) => c + 1);
    }, []);

    return { classesMap, classesError, classesLoading, retryClasses };
}

// ─── Existing Students Lookup (for duplicates) ────────────────────────────

export interface ExistingStudentsState {
    existingStudents: ExistingStudent[];
    existingStudentsError: string | null;
    existingStudentsLoading: boolean;
}

export function useExistingStudents(): ExistingStudentsState {
    const [existingStudents, setExistingStudents] = React.useState<ExistingStudent[]>([]);
    const [error, setError] = React.useState<string | null>(null);
    const [loading, setLoading] = React.useState(true);

    React.useEffect(() => {
        let cancelled = false;

        fetchExistingStudents()
            .then((data: ExistingStudent[]) => {
                if (cancelled) return;
                setExistingStudents(data);
                setError(null);
                setLoading(false);
            })
            .catch((err: unknown) => {
                if (cancelled) return;
                setExistingStudents([]);
                setError(getErrorMessage(err));
                setLoading(false);
            });

        return () => {
            cancelled = true;
        };
    }, []);

    return { existingStudents, existingStudentsError: error, existingStudentsLoading: loading };
}

// ─── Combined Lookups ─────────────────────────────────────────────────────

export interface CombinedLookupState {
    parentsMap: ParentsMap;
    classesMap: ClassesMap;
    existingStudents: ExistingStudent[];
    parentsError: string | null;
    classesError: string | null;
    parentsLoading: boolean;
    classesLoading: boolean;
    retryParents: () => void;
    retryClasses: () => void;
}

export function useLookups(): CombinedLookupState {
    const { parentsMap, parentsError, parentsLoading, retryParents } = useParentLookup();
    const { classesMap, classesError, classesLoading, retryClasses } = useClassLookup();
    const { existingStudents } = useExistingStudents();

    return {
        parentsMap,
        classesMap,
        existingStudents,
        parentsError,
        classesError,
        parentsLoading,
        classesLoading,
        retryParents,
        retryClasses,
    };
}
