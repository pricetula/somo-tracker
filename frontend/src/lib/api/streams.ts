/**
 * Streams API functions.
 *
 * Endpoints:
 *   GET    /api/v1/streams      — list streams for the active school
 *   POST   /api/v1/streams      — create a new stream
 *   PUT    /api/v1/streams/:id   — update a stream name
 *   DELETE /api/v1/streams/:id   — delete a stream
 */

import { api } from "./client";

// ─── Types ────────────────────────────────────────────────────────────────

export interface Stream {
    id: string;
    name: string;
    color: string;
}

export interface StreamListResult {
    items: Stream[];
    total: number;
}

export interface CreateStreamPayload {
    name: string;
    color: string;
}

export interface UpdateStreamPayload {
    name: string;
    color: string;
}

// ─── API Functions ─────────────────────────────────────────────────────────

/** List streams for the active school. */
export async function listStreams(): Promise<StreamListResult> {
    return api.get<StreamListResult>("/api/v1/streams");
}

/** Create a new stream. */
export async function createStream(payload: CreateStreamPayload): Promise<Stream> {
    return api.post<Stream>("/api/v1/streams", payload);
}

/** Update an existing stream's name. */
export async function updateStream(id: string, payload: UpdateStreamPayload): Promise<Stream> {
    return api.put<Stream>(`/api/v1/streams/${id}`, payload);
}

/** Delete a stream by ID. */
export async function deleteStream(id: string): Promise<void> {
    return api.delete<void>(`/api/v1/streams/${id}`);
}
