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
