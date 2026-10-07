export interface Stream {
    id: string;
    school_id: string;
    name: string;
    color: string | null;
    created_at: string;
    updated_at: string;
}

export interface ListStreamsResponse {
    code: string;
    message: string;
    streams: Stream[];
    errors: Record<string, string[]>;
}

export interface CreateStreamsResponse {
    code: string;
    message: string;
    stream_ids: string[];
    errors: Record<string, string[]>;
}
