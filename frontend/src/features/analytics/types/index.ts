/**
 * TypeScript interfaces for all 11 materialised summary tables.
 *
 * Maps to backend domains and the analysis documented in
 * docs/summary-tables-analysis.md.
 */

// ─── Shared Enums ─────────────────────────────────────────────────────────

export type PerformanceLevel = "EE" | "ME" | "AE" | "BE";

export type RiskLevel = "High" | "Medium" | "Low";

// ─── 2.1  attendance_term_summaries ─────────────────────────────────────

export interface AttendanceTermSummary {
    id: string;
    student_id: string;
    academic_term_id: string;
    learning_area_id: string;
    learning_area_name?: string;
    student_name?: string;
    periods_total: number;
    periods_present: number;
    periods_late: number;
    periods_excused: number;
    periods_absent: number;
    attendance_percentage: number;
    last_refreshed_at: string;
}

// ─── 2.2  class_daily_attendance_summaries ───────────────────────────────

export interface ClassDailyAttendanceSummary {
    id: string;
    class_id: string;
    class_name?: string;
    date: string;
    total_enrolled: number;
    present_count: number;
    absent_count: number;
    late_count: number;
    excused_count: number;
    daily_attendance_rate: number;
}

// ─── 2.3  student_term_subject_summaries ─────────────────────────────────

export interface StudentTermSubjectSummary {
    id: string;
    student_id: string;
    student_name?: string;
    academic_term_id: string;
    learning_area_id: string;
    learning_area_name: string;
    average_percentage: number;
    has_quantitative_data: boolean;
    has_rubric_data: boolean;
    mapped_performance_level: PerformanceLevel;
    quantitative_percentage?: number;
    rubric_percentage?: number;
}

// ─── 2.4  student_term_overall_summaries ─────────────────────────────────

export interface StudentTermOverallSummary {
    id: string;
    student_id: string;
    student_name?: string;
    academic_term_id: string;
    overall_mean_percentage: number;
    is_weighted_exam_score: boolean;
    mapped_performance_level: PerformanceLevel;
    exceeding_count: number;
    meeting_count: number;
    approaching_count: number;
    below_count: number;
    headteacher_remark?: string;
}

// ─── 2.5  student_cohort_position_summaries ──────────────────────────────

export interface StudentCohortPositionSummary {
    id: string;
    student_id: string;
    student_name?: string;
    class_id: string;
    class_name?: string;
    academic_term_id: string;
    overall_mean_percentage: number;
    class_rank: number;
    class_headcount: number;
    class_percentile: number;
    grade_rank: number;
    grade_headcount: number;
    grade_percentile: number;
    class_average: number;
    grade_average: number;
    variance_from_grade_mean: number;
}

// ─── 2.6  student_subject_strand_summaries ───────────────────────────────

export interface StudentSubjectStrandSummary {
    id: string;
    student_id: string;
    student_name?: string;
    academic_term_id: string;
    sub_strand_id: string;
    sub_strand_name: string;
    strand_id: string;
    strand_name: string;
    learning_area_id: string;
    learning_area_name: string;
    exceeding_count: number;
    meeting_count: number;
    approaching_count: number;
    below_count: number;
    total_indicators: number;
    mastery_percentage: number;
    requires_remediation: boolean;
    has_data: boolean;
    mapped_performance_level: PerformanceLevel;
}

// ─── 2.7  student_performance_projections ────────────────────────────────

export interface StudentPerformanceProjection {
    id: string;
    student_id: string;
    student_name?: string;
    academic_term_id: string;
    learning_area_id?: string;
    learning_area_name?: string;
    last_term_score: number;
    momentum_score: number;
    projected_score: number;
    target_gap_points: number;
    confidence_percentage: number;
    risk_level: RiskLevel;
    historical_scores: HistoricalScore[];
}

export interface HistoricalScore {
    term_index: number;
    term_name: string;
    score: number;
}

// ─── 3.1  student_behavior_term_summaries ──────────────────────────────

export interface StudentBehaviorTermSummary {
    id: string;
    student_id: string;
    student_name?: string;
    academic_term_id: string;
    total_incidents: number;
    urgent_count: number;
    commendations_count: number;
    disciplinary_count: number;
    pending_review_count: number;
    resolved_count: number;
    primary_category_id?: string;
    primary_category_name?: string;
    primary_category_type?: "COMMENDATION" | "DISCIPLINARY";
}

export interface BehaviorCategoryCount {
    category_name: string;
    category_type: "COMMENDATION" | "DISCIPLINARY";
    count: number;
}

// ─── 4.1  teacher_subject_performance_summaries ─────────────────────────

export interface TeacherSubjectPerformanceSummary {
    id: string;
    user_id: string;
    teacher_name?: string;
    learning_area_id: string;
    learning_area_name: string;
    class_id: string;
    class_name?: string;
    academic_term_id: string;
    subject_mean_score: number;
    cohort_mastery_rate: number;
    student_growth_rate: number;
    assessment_timeliness_index: number;
    strand_coverage_rate: number;
}

// ─── 4.2  teacher_delivery_summaries ────────────────────────────────────

export interface TeacherDeliverySummary {
    id: string;
    user_id: string;
    teacher_name?: string;
    academic_term_id: string;
    total_assigned_slots: number;
    marked_slots: number;
    missed_slots: number;
    sessions_created: number;
    sessions_approved: number;
    on_time_submission_rate: number;
}

// ─── 4.3  teacher_workload_summaries ────────────────────────────────────

export interface TeacherWorkloadSummary {
    id: string;
    user_id: string;
    teacher_name?: string;
    academic_year_id: string;
    total_assigned_periods: number;
    unique_subjects: number;
    classes_taught: number;
    utilization_percentage: number;
    is_overcapacity: boolean;
}

// ─── Aggregated / Query Types ────────────────────────────────────────────

export interface AnalyticsFilter {
    student_id?: string;
    class_id?: string;
    academic_term_id?: string;
    learning_area_id?: string;
    academic_year_id?: string;
}

export interface TermOverTermComparison {
    term_id: string;
    term_name: string;
    value: number;
}
