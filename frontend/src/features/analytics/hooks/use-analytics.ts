/**
 * Analytics data fetching hooks.
 *
 * Each hook fetches from the Go backend via the API client.
 * See backend/internal/{domain}/handler.go for endpoint implementation.
 */

import { useQuery, type UseQueryResult } from "@tanstack/react-query";

import { api } from "@/lib/api/client";

import type {
    AnalyticsFilter,
    AttendanceTermSummary,
    ClassDailyAttendanceSummary,
    StudentTermSubjectSummary,
    StudentTermOverallSummary,
    StudentCohortPositionSummary,
    StudentSubjectStrandSummary,
    StudentPerformanceProjection,
    StudentBehaviorTermSummary,
    BehaviorCategoryCount,
    TeacherSubjectPerformanceSummary,
    TeacherDeliverySummary,
    TeacherWorkloadSummary,
} from "../types";

// ─── Query key factories ──────────────────────────────────────────────────

export const attendanceTermSummariesKeys = {
    all: ["attendance-term-summaries"] as const,
    filtered: (filter: AnalyticsFilter) => ["attendance-term-summaries", filter] as const,
};

export const classDailyAttendanceKeys = {
    all: ["class-daily-attendance"] as const,
    filtered: (filter: AnalyticsFilter) => ["class-daily-attendance", filter] as const,
};

export const studentTermSubjectKeys = {
    all: ["student-term-subject-summaries"] as const,
    filtered: (filter: AnalyticsFilter) => ["student-term-subject-summaries", filter] as const,
};

export const studentTermOverallKeys = {
    all: ["student-term-overall-summaries"] as const,
    filtered: (filter: AnalyticsFilter) => ["student-term-overall-summaries", filter] as const,
};

export const studentCohortPositionKeys = {
    all: ["student-cohort-positions"] as const,
    filtered: (filter: AnalyticsFilter) => ["student-cohort-positions", filter] as const,
};

export const studentSubjectStrandKeys = {
    all: ["student-subject-strand-summaries"] as const,
    filtered: (filter: AnalyticsFilter) => ["student-subject-strand-summaries", filter] as const,
};

export const studentProjectionKeys = {
    all: ["student-performance-projections"] as const,
    filtered: (filter: AnalyticsFilter) => ["student-performance-projections", filter] as const,
};

// ─── Hook option types ────────────────────────────────────────────────────

export interface UseAttendanceTermSummariesOptions {
    filter?: AnalyticsFilter;
}

export interface UseClassDailyAttendanceOptions {
    filter?: AnalyticsFilter;
}

export interface UseStudentTermSubjectSummariesOptions {
    filter?: AnalyticsFilter;
}

export interface UseStudentTermOverallSummariesOptions {
    filter?: AnalyticsFilter;
}

export interface UseStudentCohortPositionOptions {
    filter?: AnalyticsFilter;
}

export interface UseStudentSubjectStrandOptions {
    filter?: AnalyticsFilter;
}

export interface UseStudentProjectionsOptions {
    filter?: AnalyticsFilter;
}

// ─── Hooks ────────────────────────────────────────────────────────────────

export function useAttendanceTermSummaries(
    options?: UseAttendanceTermSummariesOptions
): UseQueryResult<AttendanceTermSummary[]> {
    const { filter = {} } = options ?? {};

    return useQuery({
        queryKey: attendanceTermSummariesKeys.filtered(filter),
        queryFn: async () => {
            if (!filter.student_id || !filter.academic_term_id) {
                return [];
            }
            const res = await api.get<{ items: AttendanceTermSummary[]; total: number }>(
                `/api/v1/attendance/summaries/student/${filter.student_id}?term_id=${filter.academic_term_id}`
            );
            return res.items;
        },
    });
}

export function useClassDailyAttendance(
    options?: UseClassDailyAttendanceOptions
): UseQueryResult<ClassDailyAttendanceSummary[]> {
    const { filter = {} } = options ?? {};

    return useQuery({
        queryKey: classDailyAttendanceKeys.filtered(filter),
        queryFn: async () => {
            if (!filter.class_id) {
                return [];
            }
            const res = await api.get<{ items: ClassDailyAttendanceSummary[]; total: number }>(
                `/api/v1/attendance/daily/class/${filter.class_id}`
            );
            return res.items;
        },
    });
}

export function useStudentTermSubjectSummaries(
    options?: UseStudentTermSubjectSummariesOptions
): UseQueryResult<StudentTermSubjectSummary[]> {
    const { filter = {} } = options ?? {};

    return useQuery({
        queryKey: studentTermSubjectKeys.filtered(filter),
        queryFn: async () => {
            if (!filter.student_id || !filter.academic_term_id) {
                return [];
            }
            const res = await api.get<{ items: StudentTermSubjectSummary[] }>(
                `/api/v1/parent/students/${filter.student_id}/term-subject-summaries?academic_term_id=${filter.academic_term_id}`
            );
            return res.items;
        },
    });
}

export function useStudentTermOverallSummaries(
    options?: UseStudentTermOverallSummariesOptions
): UseQueryResult<StudentTermOverallSummary[]> {
    const { filter = {} } = options ?? {};

    return useQuery({
        queryKey: studentTermOverallKeys.filtered(filter),
        queryFn: async () => {
            if (!filter.student_id || !filter.academic_term_id) {
                return [];
            }
            const res = await api.get<{ data: StudentTermOverallSummary }>(
                `/api/v1/assessments/term-overall-summaries/${filter.student_id}/${filter.academic_term_id}`
            );
            return [res.data];
        },
    });
}

export function useStudentCohortPosition(
    options?: UseStudentCohortPositionOptions
): UseQueryResult<StudentCohortPositionSummary[]> {
    const { filter = {} } = options ?? {};

    return useQuery({
        queryKey: studentCohortPositionKeys.filtered(filter),
        queryFn: async () => {
            if (!filter.student_id || !filter.academic_term_id) {
                return [];
            }
            const res = await api.get<{ data: StudentCohortPositionSummary }>(
                `/api/v1/cohort-positions/${filter.student_id}?term_id=${filter.academic_term_id}`
            );
            return [res.data];
        },
    });
}

export function useStudentSubjectStrandSummaries(
    options?: UseStudentSubjectStrandOptions
): UseQueryResult<StudentSubjectStrandSummary[]> {
    const { filter = {} } = options ?? {};

    return useQuery({
        queryKey: studentSubjectStrandKeys.filtered(filter),
        queryFn: async () => {
            if (!filter.student_id || !filter.academic_term_id) {
                return [];
            }
            const res = await api.get<{ items: StudentSubjectStrandSummary[]; total: number }>(
                `/api/v1/assessments/subject-strand-summaries/${filter.student_id}/${filter.academic_term_id}`
            );
            return res.items;
        },
    });
}

export function useStudentProjections(
    options?: UseStudentProjectionsOptions
): UseQueryResult<StudentPerformanceProjection[]> {
    const { filter = {} } = options ?? {};

    return useQuery({
        queryKey: studentProjectionKeys.filtered(filter),
        queryFn: async () => {
            if (!filter.student_id || !filter.academic_term_id) {
                return [];
            }
            const params = new URLSearchParams();
            if (filter.learning_area_id) {
                params.set("learning_area_id", filter.learning_area_id);
            }
            const qs = params.toString();
            const res = await api.get<{ data: StudentPerformanceProjection }>(
                `/api/v1/assessments/projections/${filter.student_id}/${filter.academic_term_id}${qs ? `?${qs}` : ""}`
            );
            return [res.data];
        },
    });
}

// ─── Behaviour hooks ────────────────────────────────────────────────────

export const studentBehaviorKeys = {
    all: ["student-behavior-summaries"] as const,
    filtered: (filter: AnalyticsFilter) => ["student-behavior-summaries", filter] as const,
};

export function useStudentBehaviorSummaries(options?: {
    filter?: AnalyticsFilter;
}): UseQueryResult<StudentBehaviorTermSummary[]> {
    const { filter = {} } = options ?? {};
    return useQuery({
        queryKey: studentBehaviorKeys.filtered(filter),
        queryFn: async () => {
            if (!filter.student_id || !filter.academic_term_id) {
                return [];
            }
            const res = await api.get<StudentBehaviorTermSummary>(
                `/api/v1/behavior/summaries/${filter.student_id}?term_id=${filter.academic_term_id}`
            );
            return [res];
        },
    });
}

// ─── Teacher hooks ────────────────────────────────────────────────────────

export const teacherPerformanceKeys = {
    all: ["teacher-performance-summaries"] as const,
    filtered: (filter: AnalyticsFilter) => ["teacher-performance-summaries", filter] as const,
};

export function useTeacherPerformanceSummaries(options?: {
    filter?: AnalyticsFilter;
}): UseQueryResult<TeacherSubjectPerformanceSummary[]> {
    const { filter = {} } = options ?? {};
    return useQuery({
        queryKey: teacherPerformanceKeys.filtered(filter),
        queryFn: async () => {
            if (!filter.academic_term_id) {
                return [];
            }
            const res = await api.get<{ items: TeacherSubjectPerformanceSummary[]; total: number }>(
                `/api/v1/teacher-performance/summaries?term_id=${filter.academic_term_id}`
            );
            return res.items;
        },
    });
}

export const teacherDeliveryKeys = {
    all: ["teacher-delivery-summaries"] as const,
    filtered: (filter: AnalyticsFilter) => ["teacher-delivery-summaries", filter] as const,
};

export function useTeacherDeliverySummaries(options?: {
    filter?: AnalyticsFilter;
}): UseQueryResult<TeacherDeliverySummary[]> {
    const { filter = {} } = options ?? {};
    return useQuery({
        queryKey: teacherDeliveryKeys.filtered(filter),
        queryFn: async () => {
            if (!filter.academic_term_id) {
                return [];
            }
            const res = await api.get<{ items: TeacherDeliverySummary[]; total: number }>(
                `/api/v1/teacher-delivery-summaries?term_id=${filter.academic_term_id}`
            );
            return res.items;
        },
    });
}

export const teacherWorkloadKeys = {
    all: ["teacher-workload-summaries"] as const,
    filtered: (filter: AnalyticsFilter) => ["teacher-workload-summaries", filter] as const,
};

export function useTeacherWorkloadSummaries(options?: {
    filter?: AnalyticsFilter;
}): UseQueryResult<TeacherWorkloadSummary[]> {
    const { filter = {} } = options ?? {};
    return useQuery({
        queryKey: teacherWorkloadKeys.filtered(filter),
        queryFn: async () => {
            if (!filter.academic_year_id) {
                return [];
            }
            const res = await api.get<{ items: TeacherWorkloadSummary[]; total: number }>(
                `/api/v1/teacher-workload-summaries?academic_year_id=${filter.academic_year_id}`
            );
            return res.items;
        },
    });
}

// ─── Behavior category breakdown helper (static — no backend endpoint yet) ─

export function mockBehaviorCategoryBreakdown(): BehaviorCategoryCount[] {
    return [
        { category_name: "Helping Others", category_type: "COMMENDATION", count: 1 },
        { category_name: "Academic Achievement", category_type: "COMMENDATION", count: 1 },
        { category_name: "Punctuality", category_type: "DISCIPLINARY", count: 1 },
        { category_name: "Conduct", category_type: "DISCIPLINARY", count: 1 },
        { category_name: "Violence", category_type: "DISCIPLINARY", count: 1 },
    ];
}
