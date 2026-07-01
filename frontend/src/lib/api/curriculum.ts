/**
 * Curriculum API functions.
 *
 * Endpoints:
 *   Learning Areas:     GET|POST /api/v1/curriculum/learning-areas, GET|PUT|DELETE /:id, GET /:id/tree
 *   Strands:            GET|POST /api/v1/curriculum/strands, PUT|DELETE /:id
 *   Sub-Strands:        GET|POST /api/v1/curriculum/sub-strands, PUT|DELETE /:id
 *   Performance Indicators: GET|POST /api/v1/curriculum/performance-indicators, PUT|DELETE /:id
 */

import { api } from "./client";

// ─── Domain Types ─────────────────────────────────────────────────────────

export interface LearningArea {
    id: string;
    name: string;
    code: string;
    education_level: string;
}

export interface Strand {
    id: string;
    learning_area_id: string;
    name: string;
}

export interface SubStrand {
    id: string;
    strand_id: string;
    name: string;
}

export interface PerformanceIndicator {
    id: string;
    sub_strand_id: string;
    description: string;
    sequence_order: number;
}

export interface StrandTree {
    id: string;
    learning_area_id: string;
    name: string;
    sub_strands: SubStrandTree[];
}

export interface SubStrandTree {
    id: string;
    strand_id: string;
    name: string;
    performance_indicators: PerformanceIndicator[];
}

export interface LearningAreaTree {
    id: string;
    name: string;
    code: string;
    education_level: string;
    strands: StrandTree[];
}

// ─── Response Types ───────────────────────────────────────────────────────

export interface ListLearningAreasResponse {
    learning_areas: LearningArea[];
    total: number;
}

export interface ListStrandsResponse {
    strands: Strand[];
    total: number;
}

export interface ListSubStrandsResponse {
    sub_strands: SubStrand[];
    total: number;
}

export interface ListPerformanceIndicatorsResponse {
    performance_indicators: PerformanceIndicator[];
    total: number;
}

// ─── Payload Types ────────────────────────────────────────────────────────

export interface CreateLearningAreaPayload {
    code: string;
    name: string;
    education_level: string;
}

export interface UpdateLearningAreaPayload {
    name?: string;
    code?: string;
    education_level?: string;
}

export interface CreateStrandPayload {
    learning_area_id: string;
    name: string;
}

export interface UpdateStrandPayload {
    name?: string;
}

export interface CreateSubStrandPayload {
    strand_id: string;
    name: string;
}

export interface UpdateSubStrandPayload {
    name?: string;
}

export interface CreatePerformanceIndicatorPayload {
    sub_strand_id: string;
    description: string;
    sequence_order?: number;
}

export interface UpdatePerformanceIndicatorPayload {
    description?: string;
    sequence_order?: number;
}

// ─── API Functions ────────────────────────────────────────────────────────

// ── Learning Areas ─────────────────────────────────────────────────────────

/** List all learning areas for the current school, optionally filtered by education_level. */
export async function listLearningAreas(
    params: { education_level?: string } = {}
): Promise<ListLearningAreasResponse> {
    const searchParams = new URLSearchParams();
    if (params.education_level) searchParams.set("education_level", params.education_level);

    const qs = searchParams.toString();
    return api.get<ListLearningAreasResponse>(`/api/v1/curriculum/learning-areas?${qs}`);
}

/** Create a new learning area. */
export async function createLearningArea(data: CreateLearningAreaPayload): Promise<{ id: string }> {
    return api.post<{ id: string }>("/api/v1/curriculum/learning-areas", data);
}

/** Get a single learning area by ID. */
export async function getLearningArea(id: string): Promise<LearningArea> {
    return api.get<LearningArea>(`/api/v1/curriculum/learning-areas/${id}`);
}

/** Get the full learning area tree (strands → sub-strands → indicators). */
export async function getLearningAreaTree(id: string): Promise<LearningAreaTree> {
    return api.get<LearningAreaTree>(`/api/v1/curriculum/learning-areas/${id}/tree`);
}

/** Update a learning area. */
export async function updateLearningArea(
    id: string,
    data: UpdateLearningAreaPayload
): Promise<void> {
    return api.put<void>(`/api/v1/curriculum/learning-areas/${id}`, data);
}

/** Delete a learning area. */
export async function deleteLearningArea(id: string): Promise<void> {
    return api.delete<void>(`/api/v1/curriculum/learning-areas/${id}`);
}

// ── Strands ───────────────────────────────────────────────────────────────

/** List strands for a learning area. */
export async function listStrands(learningAreaId: string): Promise<ListStrandsResponse> {
    return api.get<ListStrandsResponse>(
        `/api/v1/curriculum/strands?learning_area_id=${encodeURIComponent(learningAreaId)}`
    );
}

/** Create a new strand. */
export async function createStrand(data: CreateStrandPayload): Promise<{ id: string }> {
    return api.post<{ id: string }>("/api/v1/curriculum/strands", data);
}

/** Update a strand. */
export async function updateStrand(id: string, data: UpdateStrandPayload): Promise<void> {
    return api.put<void>(`/api/v1/curriculum/strands/${id}`, data);
}

/** Delete a strand. */
export async function deleteStrand(id: string): Promise<void> {
    return api.delete<void>(`/api/v1/curriculum/strands/${id}`);
}

// ── Sub-Strands ───────────────────────────────────────────────────────────

/** List sub-strands for a strand. */
export async function listSubStrands(strandId: string): Promise<ListSubStrandsResponse> {
    return api.get<ListSubStrandsResponse>(
        `/api/v1/curriculum/sub-strands?strand_id=${encodeURIComponent(strandId)}`
    );
}

/** Create a new sub-strand. */
export async function createSubStrand(data: CreateSubStrandPayload): Promise<{ id: string }> {
    return api.post<{ id: string }>("/api/v1/curriculum/sub-strands", data);
}

/** Update a sub-strand. */
export async function updateSubStrand(id: string, data: UpdateSubStrandPayload): Promise<void> {
    return api.put<void>(`/api/v1/curriculum/sub-strands/${id}`, data);
}

/** Delete a sub-strand. */
export async function deleteSubStrand(id: string): Promise<void> {
    return api.delete<void>(`/api/v1/curriculum/sub-strands/${id}`);
}

// ── Performance Indicators ───────────────────────────────────────────────

/** List performance indicators for a sub-strand. */
export async function listPerformanceIndicators(
    subStrandId: string
): Promise<ListPerformanceIndicatorsResponse> {
    return api.get<ListPerformanceIndicatorsResponse>(
        `/api/v1/curriculum/performance-indicators?sub_strand_id=${encodeURIComponent(subStrandId)}`
    );
}

/** Create a new performance indicator. */
export async function createPerformanceIndicator(
    data: CreatePerformanceIndicatorPayload
): Promise<{ id: string }> {
    return api.post<{ id: string }>("/api/v1/curriculum/performance-indicators", data);
}

/** Update a performance indicator. */
export async function updatePerformanceIndicator(
    id: string,
    data: UpdatePerformanceIndicatorPayload
): Promise<void> {
    return api.put<void>(`/api/v1/curriculum/performance-indicators/${id}`, data);
}

/** Delete a performance indicator. */
export async function deletePerformanceIndicator(id: string): Promise<void> {
    return api.delete<void>(`/api/v1/curriculum/performance-indicators/${id}`);
}
