/** Streams feature — API clients */

import { api } from "@/lib/api/client";
import type { Stream, ListStreamsResponse, CreateStreamsResponse } from "../types/stream";

export async function listStreams(): Promise<Stream[]> {
    const res = await api.get<ListStreamsResponse>("/api/school/streams");
    return res?.streams ?? [];
}

export async function createStreams(names: string[]): Promise<CreateStreamsResponse> {
    return api.post<CreateStreamsResponse>("/api/school/streams", names);
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
    const res = await api.get<GetStreamResponse>(`/api/school/streams/${id}`);
    return res.stream;
}

export async function updateStream(
    id: string,
    data: { name?: string; color?: string | null }
): Promise<Stream> {
    const res = await api.patch<UpdateStreamResponse>(`/api/school/streams/${id}`, data);
    return res.stream;
}

export interface DeleteStreamsResponse {
    code: string;
    message: string;
    errors: Record<string, string[]>;
}

export async function deleteStreams(ids: string[]): Promise<void> {
    await api.post<DeleteStreamsResponse>("/api/school/streams/delete", { ids });
}
