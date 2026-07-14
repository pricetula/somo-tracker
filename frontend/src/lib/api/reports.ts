/**
 * Reports API functions.
 *
 * Endpoints:
 *   GET    /api/v1/reports/terms/:term_id/students/:student_id  — parent view
 *   POST   /api/v1/reports/terms/:term_id/generate              — admin generate
 *   GET    /api/v1/reports/terms/:term_id/status                — generation status
 */

import { api } from "./client";

// ─── Types ────────────────────────────────────────────────────────────────

export interface TermReportBehaviorNote {
    date: string;
    category_name: string;
    description: string;
    subject: string;
}

export interface TermReportAttendance {
    attendance_percentage: number;
    absences_count: number;
    late_count: number;
}

export interface TermReportCompetency {
    learning_area_name: string;
    final_level: string;
    teacher_narrative_summary?: string;
}

export interface TermReport {
    id: string;
    student_id: string;
    term_id: string;
    attendance: TermReportAttendance;
    behavior_notes: TermReportBehaviorNote[];
    competency_summary: TermReportCompetency[];
    status: "DRAFT" | "PUBLISHED";
    generated_at: string;
    published_at?: string;
}

export interface TermReportGenerateResponse {
    message: string;
    report_id: string;
    status: string;
}

// ─── API Functions ────────────────────────────────────────────────────────

/** Get a compiled term report for a student (parent read-only view). */
export async function getTermReport(termId: string, studentId: string): Promise<TermReport> {
    return api.get<TermReport>(`/api/v1/reports/terms/${termId}/students/${studentId}`);
}

/** Trigger generation of a term report (admin action). */
export async function generateTermReport(
    termId: string,
    studentId: string
): Promise<TermReportGenerateResponse> {
    return api.post<TermReportGenerateResponse>(`/api/v1/reports/terms/${termId}/generate`, {
        student_id: studentId,
    });
}

/** Generate term reports for all students in a class (admin action). */
export async function generateClassTermReports(
    termId: string,
    classId: string
): Promise<{ message: string; count: number }> {
    return api.post<{ message: string; count: number }>(
        `/api/v1/reports/terms/${termId}/generate`,
        { class_id: classId }
    );
}

/** List all generated reports for a term. */
export async function listTermReports(
    termId: string,
    classId?: string
): Promise<{ items: TermReport[]; total: number }> {
    const params = classId ? `?class_id=${classId}` : "";
    return api.get<{ items: TermReport[]; total: number }>(
        `/api/v1/reports/terms/${termId}${params}`
    );
}

/** Update term report status (publish). */
export async function publishTermReport(reportId: string): Promise<{ message: string }> {
    return api.post<{ message: string }>(`/api/v1/reports/${reportId}/publish`);
}
