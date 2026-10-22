/** Streams feature — API clients */

import { api } from "@/lib/api/client";
import type { Stream, ListStreamsResponse, CreateStreamsResponse } from "../types/stream";

export async function listStreams(): Promise<Stream[]> {
    const res = await api.get<ListStreamsResponse>("/api/streams");
    return res?.streams ?? [];
}

export async function createStreams(
    items: { name: string; color?: string }[]
): Promise<CreateStreamsResponse> {
    return api.post<CreateStreamsResponse>("/api/streams", items);
}

export interface GetStreamResponse {
    code: string;
    message: string;
    stream: Stream;
    errors: Record<string, string[]>;
}

export interface UpdateStreamResponse {
    code: string;
    message: string;
    stream: Stream;
    errors: Record<string, string[]>;
}

export async function getStream(id: string): Promise<Stream> {
    const res = await api.get<GetStreamResponse>(`/api/streams/${id}`);
    return res.stream;
}

export async function updateStream(
    id: string,
    data: { name?: string; color?: string | null }
): Promise<Stream> {
    const res = await api.patch<UpdateStreamResponse>(`/api/streams/${id}`, data);
    return res.stream;
}

export interface DeleteStreamsResponse {
    code: string;
    message: string;
    errors: Record<string, string[]>;
}

export async function deleteStreams(ids: string[]): Promise<void> {
    await api.post<DeleteStreamsResponse>("/api/streams/delete", { ids });
}
