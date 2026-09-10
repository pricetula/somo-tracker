/**
 * Streams API functions.
 *
 * Backend contract (backend/internal/api/streams_handler.go):
 *
 *   POST   /api/school/streams  — create streams (protected, active_school_id required)
 */

import { api } from "./client";

// ─── Types ────────────────────────────────────────────────────────────────

export interface Stream {
    id: string;
    name: string;
}

export interface CreateStreamsResponse {
    code: string;
    message: string;
    stream_ids: string[];
    errors: Record<string, string[]>;
}

// ─── API Functions ─────────────────────────────────────────────────────────

/** Create streams for the active school. */
export async function createStreams(names: string[]): Promise<CreateStreamsResponse> {
    return api.post<CreateStreamsResponse>("/api/school/streams", names);
}
