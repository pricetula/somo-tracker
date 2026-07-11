/**
 * Streams API functions.
 *
 * Endpoint:
 *   GET /api/v1/streams — list streams for the active school
 */

import { api } from "./client";

// ─── Types ────────────────────────────────────────────────────────────────

export interface Stream {
    id: string;
    name: string;
}

export interface StreamListResult {
    items: Stream[];
    total: number;
}

// ─── API Functions ─────────────────────────────────────────────────────────

/** List streams for the active school. */
export async function listStreams(): Promise<StreamListResult> {
    return api.get<StreamListResult>("/api/v1/streams");
}
