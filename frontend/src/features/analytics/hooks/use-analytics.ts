/**
 * Analytics data fetching hooks.
 *
 * Each hook returns a TanStack Query result for its summary table.
 * When the backend API is ready, replace the queryFn with real API calls.
 */

import { useQuery, type UseQueryResult } from "@tanstack/react-query";

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
            // TODO: Replace with actual API call
            // const res = await api.get("/api/v1/analytics/attendance-term-summaries", { params: filter });
            // return res.data;
            return mockAttendanceTermSummaries(filter);
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
            return mockClassDailyAttendance(filter);
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
            return mockStudentTermSubjectSummaries(filter);
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
            return mockStudentTermOverallSummaries(filter);
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
            return mockStudentCohortPositionSummaries(filter);
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
            return mockStudentSubjectStrandSummaries(filter);
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
            return mockStudentPerformanceProjections(filter);
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
        queryFn: async () => mockStudentBehaviorSummaries(filter),
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
        queryFn: async () => mockTeacherPerformanceSummaries(filter),
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
        queryFn: async () => mockTeacherDeliverySummaries(filter),
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
        queryFn: async () => mockTeacherWorkloadSummaries(filter),
    });
}

// ─── Mock data (development only — remove when backend is ready) ──────────

function mockAttendanceTermSummaries(_filter: AnalyticsFilter): AttendanceTermSummary[] {
    return [
        {
            id: "att-1",
            student_id: "stu-1",
            student_name: "Aisha Mohamed",
            academic_term_id: "term-1",
            learning_area_id: "la-1",
            learning_area_name: "Mathematics",
            periods_total: 120,
            periods_present: 108,
            periods_late: 4,
            periods_excused: 2,
            periods_absent: 6,
            attendance_percentage: 90.0,
            last_refreshed_at: new Date().toISOString(),
        },
        {
            id: "att-2",
            student_id: "stu-1",
            student_name: "Aisha Mohamed",
            academic_term_id: "term-1",
            learning_area_id: "la-2",
            learning_area_name: "English",
            periods_total: 110,
            periods_present: 95,
            periods_late: 5,
            periods_excused: 3,
            periods_absent: 7,
            attendance_percentage: 86.36,
            last_refreshed_at: new Date().toISOString(),
        },
        {
            id: "att-3",
            student_id: "stu-1",
            student_name: "Aisha Mohamed",
            academic_term_id: "term-1",
            learning_area_id: "la-3",
            learning_area_name: "Kiswahili",
            periods_total: 100,
            periods_present: 80,
            periods_late: 8,
            periods_excused: 2,
            periods_absent: 10,
            attendance_percentage: 80.0,
            last_refreshed_at: new Date().toISOString(),
        },
        {
            id: "att-4",
            student_id: "stu-1",
            student_name: "Aisha Mohamed",
            academic_term_id: "term-1",
            learning_area_id: "la-4",
            learning_area_name: "Science",
            periods_total: 90,
            periods_present: 85,
            periods_late: 2,
            periods_excused: 1,
            periods_absent: 2,
            attendance_percentage: 94.44,
            last_refreshed_at: new Date().toISOString(),
        },
        {
            id: "att-5",
            student_id: "stu-1",
            student_name: "Aisha Mohamed",
            academic_term_id: "term-1",
            learning_area_id: "la-5",
            learning_area_name: "Social Studies",
            periods_total: 95,
            periods_present: 88,
            periods_late: 3,
            periods_excused: 1,
            periods_absent: 3,
            attendance_percentage: 92.63,
            last_refreshed_at: new Date().toISOString(),
        },
    ];
}

function mockClassDailyAttendance(_filter: AnalyticsFilter): ClassDailyAttendanceSummary[] {
    const summaries: ClassDailyAttendanceSummary[] = [];
    const startDate = new Date("2026-01-12");
    for (let i = 0; i < 30; i++) {
        const d = new Date(startDate);
        d.setDate(d.getDate() + i);
        if (d.getDay() === 0 || d.getDay() === 6) continue; // skip weekends
        const present = 28 + Math.floor(Math.random() * 7);
        const absent = Math.floor(Math.random() * 3);
        const late = Math.floor(Math.random() * 3);
        const excused = Math.floor(Math.random() * 2);
        summaries.push({
            id: `daily-${i}`,
            class_id: "class-1",
            class_name: "G4 Blue",
            date: d.toISOString().split("T")[0],
            total_enrolled: present + absent + late + excused,
            present_count: present,
            absent_count: absent,
            late_count: late,
            excused_count: excused,
            daily_attendance_rate: (present / (present + absent + late + excused)) * 100,
        });
    }
    return summaries;
}

function mockStudentTermSubjectSummaries(_filter: AnalyticsFilter): StudentTermSubjectSummary[] {
    return [
        {
            id: "sts-1",
            student_id: "stu-1",
            student_name: "Aisha Mohamed",
            academic_term_id: "term-1",
            learning_area_id: "la-1",
            learning_area_name: "English",
            average_percentage: 81.67,
            has_quantitative_data: true,
            has_rubric_data: true,
            mapped_performance_level: "AE",
            quantitative_percentage: 90.0,
            rubric_percentage: 77.5,
        },
        {
            id: "sts-2",
            student_id: "stu-1",
            student_name: "Aisha Mohamed",
            academic_term_id: "term-1",
            learning_area_id: "la-2",
            learning_area_name: "Mathematics",
            average_percentage: 72.0,
            has_quantitative_data: true,
            has_rubric_data: false,
            mapped_performance_level: "AE",
        },
        {
            id: "sts-3",
            student_id: "stu-1",
            student_name: "Aisha Mohamed",
            academic_term_id: "term-1",
            learning_area_id: "la-3",
            learning_area_name: "Kiswahili",
            average_percentage: 55.0,
            has_quantitative_data: true,
            has_rubric_data: true,
            mapped_performance_level: "BE",
        },
        {
            id: "sts-4",
            student_id: "stu-1",
            student_name: "Aisha Mohamed",
            academic_term_id: "term-1",
            learning_area_id: "la-4",
            learning_area_name: "Science",
            average_percentage: 88.0,
            has_quantitative_data: true,
            has_rubric_data: false,
            mapped_performance_level: "ME",
        },
        {
            id: "sts-5",
            student_id: "stu-1",
            student_name: "Aisha Mohamed",
            academic_term_id: "term-1",
            learning_area_id: "la-5",
            learning_area_name: "Social Studies",
            average_percentage: 91.0,
            has_quantitative_data: true,
            has_rubric_data: false,
            mapped_performance_level: "EE",
        },
    ];
}

function mockStudentTermOverallSummaries(_filter: AnalyticsFilter): StudentTermOverallSummary[] {
    return [
        {
            id: "overall-1",
            student_id: "stu-1",
            student_name: "Aisha Mohamed",
            academic_term_id: "term-1",
            overall_mean_percentage: 77.53,
            is_weighted_exam_score: false,
            mapped_performance_level: "AE",
            exceeding_count: 1,
            meeting_count: 1,
            approaching_count: 2,
            below_count: 1,
            headteacher_remark:
                "Aisha has shown consistent effort this term. Keep up the good work in Mathematics and English.",
        },
    ];
}

function mockStudentCohortPositionSummaries(
    _filter: AnalyticsFilter
): StudentCohortPositionSummary[] {
    return [
        {
            id: "cohort-1",
            student_id: "stu-1",
            student_name: "Aisha Mohamed",
            class_id: "class-1",
            class_name: "G4 Blue",
            academic_term_id: "term-1",
            overall_mean_percentage: 85.0,
            class_rank: 5,
            class_headcount: 40,
            class_percentile: 87.5,
            grade_rank: 18,
            grade_headcount: 120,
            grade_percentile: 85.0,
            class_average: 62.3,
            grade_average: 60.1,
            variance_from_grade_mean: 24.9,
        },
    ];
}

function mockStudentSubjectStrandSummaries(
    _filter: AnalyticsFilter
): StudentSubjectStrandSummary[] {
    const strands = [
        {
            strand: "Reading",
            sub_strands: [
                { name: "Reading Comprehension", ee: 0, me: 2, ae: 2, be: 1 },
                { name: "Vocabulary Development", ee: 1, me: 3, ae: 1, be: 0 },
                { name: "Phonics & Decoding", ee: 0, me: 1, ae: 2, be: 2 },
            ],
        },
        {
            strand: "Writing",
            sub_strands: [
                { name: "Creative Writing", ee: 2, me: 2, ae: 1, be: 0 },
                { name: "Grammar & Mechanics", ee: 0, me: 2, ae: 3, be: 0 },
                { name: "Handwriting", ee: 1, me: 4, ae: 0, be: 0 },
            ],
        },
        {
            strand: "Speaking & Listening",
            sub_strands: [
                { name: "Oral Communication", ee: 3, me: 2, ae: 0, be: 0 },
                { name: "Listening Comprehension", ee: 1, me: 3, ae: 1, be: 0 },
            ],
        },
    ];

    const results: StudentSubjectStrandSummary[] = [];
    let id = 0;
    const laName = "English";
    const laId = "la-2";

    for (const strand of strands) {
        for (const sub of strand.sub_strands) {
            id++;
            const total = sub.ee + sub.me + sub.ae + sub.be;
            const mastery = total > 0 ? ((sub.ee + sub.me) / total) * 100 : 0;
            const level: StudentSubjectStrandSummary["mapped_performance_level"] =
                mastery >= 80 ? "EE" : mastery >= 60 ? "ME" : mastery >= 40 ? "AE" : "BE";

            results.push({
                id: `strand-${id}`,
                student_id: "stu-1",
                student_name: "Aisha Mohamed",
                academic_term_id: "term-1",
                sub_strand_id: `sub-${id}`,
                sub_strand_name: sub.name,
                strand_id: `strand-${id}`,
                strand_name: strand.strand,
                learning_area_id: laId,
                learning_area_name: laName,
                exceeding_count: sub.ee,
                meeting_count: sub.me,
                approaching_count: sub.ae,
                below_count: sub.be,
                total_indicators: total,
                mastery_percentage: mastery,
                requires_remediation: sub.be > 0 || mastery < 50,
                has_data: true,
                mapped_performance_level: level,
            });
        }
    }

    return results;
}

// ─── Behaviour mock ─────────────────────────────────────────────────────

function mockStudentBehaviorSummaries(_filter: AnalyticsFilter): StudentBehaviorTermSummary[] {
    return [
        {
            id: "beh-1",
            student_id: "stu-1",
            student_name: "Aisha Mohamed",
            academic_term_id: "term-1",
            total_incidents: 5,
            urgent_count: 1,
            commendations_count: 2,
            disciplinary_count: 3,
            pending_review_count: 0,
            resolved_count: 5,
            primary_category_name: "Disciplinary",
            primary_category_type: "DISCIPLINARY",
        },
        {
            id: "beh-2",
            student_id: "stu-2",
            student_name: "Brian Kiprop",
            academic_term_id: "term-1",
            total_incidents: 2,
            urgent_count: 0,
            commendations_count: 2,
            disciplinary_count: 0,
            pending_review_count: 0,
            resolved_count: 2,
            primary_category_name: "Commendation",
            primary_category_type: "COMMENDATION",
        },
    ];
}

export function mockBehaviorCategoryBreakdown(): BehaviorCategoryCount[] {
    return [
        { category_name: "Helping Others", category_type: "COMMENDATION", count: 1 },
        { category_name: "Academic Achievement", category_type: "COMMENDATION", count: 1 },
        { category_name: "Punctuality", category_type: "DISCIPLINARY", count: 1 },
        { category_name: "Conduct", category_type: "DISCIPLINARY", count: 1 },
        { category_name: "Violence", category_type: "DISCIPLINARY", count: 1 },
    ];
}

// ─── Teacher mocks ───────────────────────────────────────────────────────

function mockTeacherPerformanceSummaries(
    _filter: AnalyticsFilter
): TeacherSubjectPerformanceSummary[] {
    return [
        {
            id: "tp-1",
            user_id: "teacher-1",
            teacher_name: "Mr. Kamau",
            learning_area_id: "la-1",
            learning_area_name: "Mathematics",
            class_id: "class-1",
            class_name: "G4 Blue",
            academic_term_id: "term-1",
            subject_mean_score: 72.5,
            cohort_mastery_rate: 60.0,
            student_growth_rate: 3.2,
            assessment_timeliness_index: 80.0,
            strand_coverage_rate: 80.0,
        },
        {
            id: "tp-2",
            user_id: "teacher-1",
            teacher_name: "Mr. Kamau",
            learning_area_id: "la-2",
            learning_area_name: "English",
            class_id: "class-1",
            class_name: "G4 Blue",
            academic_term_id: "term-1",
            subject_mean_score: 78.3,
            cohort_mastery_rate: 70.0,
            student_growth_rate: 1.8,
            assessment_timeliness_index: 90.0,
            strand_coverage_rate: 100.0,
        },
    ];
}

function mockTeacherDeliverySummaries(_filter: AnalyticsFilter): TeacherDeliverySummary[] {
    return [
        {
            id: "td-1",
            user_id: "teacher-1",
            teacher_name: "Ms. Ochieng",
            academic_term_id: "term-1",
            total_assigned_slots: 104,
            marked_slots: 92,
            missed_slots: 8,
            sessions_created: 92,
            sessions_approved: 88,
            on_time_submission_rate: 96.15,
        },
        {
            id: "td-2",
            user_id: "teacher-2",
            teacher_name: "Mr. Juma",
            academic_term_id: "term-1",
            total_assigned_slots: 104,
            marked_slots: 85,
            missed_slots: 10,
            sessions_created: 85,
            sessions_approved: 80,
            on_time_submission_rate: 91.35,
        },
    ];
}

function mockTeacherWorkloadSummaries(_filter: AnalyticsFilter): TeacherWorkloadSummary[] {
    return [
        {
            id: "tw-1",
            user_id: "teacher-1",
            teacher_name: "Mr. Juma",
            academic_year_id: "year-1",
            total_assigned_periods: 32,
            unique_subjects: 3,
            classes_taught: 6,
            utilization_percentage: 16.0,
            is_overcapacity: true,
        },
        {
            id: "tw-2",
            user_id: "teacher-2",
            teacher_name: "Ms. Ochieng",
            academic_year_id: "year-1",
            total_assigned_periods: 24,
            unique_subjects: 2,
            classes_taught: 4,
            utilization_percentage: 12.0,
            is_overcapacity: false,
        },
        {
            id: "tw-3",
            user_id: "teacher-3",
            teacher_name: "Mrs. Wanjiku",
            academic_year_id: "year-1",
            total_assigned_periods: 28,
            unique_subjects: 1,
            classes_taught: 5,
            utilization_percentage: 14.0,
            is_overcapacity: true,
        },
    ];
}

function mockStudentPerformanceProjections(
    _filter: AnalyticsFilter
): StudentPerformanceProjection[] {
    return [
        {
            id: "proj-1",
            student_id: "stu-1",
            student_name: "Aisha Mohamed",
            academic_term_id: "term-3",
            learning_area_id: "la-1",
            learning_area_name: "Mathematics",
            last_term_score: 75.0,
            momentum_score: 6.5,
            projected_score: 81.5,
            target_gap_points: 31.5,
            confidence_percentage: 85,
            risk_level: "Low",
            historical_scores: [
                { term_index: 0, term_name: "Term 1 (G4)", score: 62.0 },
                { term_index: 1, term_name: "Term 2 (G4)", score: 70.0 },
                { term_index: 2, term_name: "Term 3 (G4)", score: 75.0 },
            ],
        },
        {
            id: "proj-2",
            student_id: "stu-1",
            student_name: "Aisha Mohamed",
            academic_term_id: "term-3",
            learning_area_id: "la-2",
            learning_area_name: "English",
            last_term_score: 81.67,
            momentum_score: 3.2,
            projected_score: 84.87,
            target_gap_points: 34.87,
            confidence_percentage: 85,
            risk_level: "Low",
            historical_scores: [
                { term_index: 0, term_name: "Term 1 (G4)", score: 72.0 },
                { term_index: 1, term_name: "Term 2 (G4)", score: 78.0 },
                { term_index: 2, term_name: "Term 3 (G4)", score: 81.67 },
            ],
        },
        {
            id: "proj-3",
            student_id: "stu-1",
            student_name: "Aisha Mohamed",
            academic_term_id: "term-3",
            learning_area_id: "la-3",
            learning_area_name: "Kiswahili",
            last_term_score: 55.0,
            momentum_score: -2.1,
            projected_score: 52.9,
            target_gap_points: 2.9,
            confidence_percentage: 85,
            risk_level: "Medium",
            historical_scores: [
                { term_index: 0, term_name: "Term 1 (G4)", score: 60.0 },
                { term_index: 1, term_name: "Term 2 (G4)", score: 58.0 },
                { term_index: 2, term_name: "Term 3 (G4)", score: 55.0 },
            ],
        },
    ];
}
