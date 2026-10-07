import type { SubjectListItem, TopicItem, SubTopicItem } from "../types/curriculum";
import { api } from "@/lib/api/client";

export async function listSubjects(params?: {
    page?: number;
    limit?: number;
    search?: string;
    filters?: Record<string, string | string[]>;
}): Promise<{ items: SubjectListItem[]; total: number }> {
    const searchParams = new URLSearchParams();
    if (params?.page != null) searchParams.set("page", String(params.page));
    if (params?.limit != null) searchParams.set("limit", String(params.limit));
    if (params?.search) searchParams.set("search", params.search);
    if (params?.filters?.grade) searchParams.set("grade", String(params.filters.grade));
    const query = searchParams.toString() ? `?${searchParams.toString()}` : "";
    return api.get<{ items: SubjectListItem[]; total: number }>(`/api/subjects${query}`);
}

export async function getSubject(id: string): Promise<SubjectListItem | null> {
    return api.get<SubjectListItem>(`/api/subjects/${id}`);
}

export async function listTopics(subjectId: string): Promise<TopicItem[]> {
    const res = await api.get<{ items: TopicItem[]; total: number }>(
        `/api/topics?subject_id=${subjectId}`
    );
    return res?.items ?? [];
}

export async function listSubTopics(topicId: string): Promise<SubTopicItem[]> {
    const res = await api.get<{ items: SubTopicItem[]; total: number }>(
        `/api/sub-topics?topic_id=${topicId}`
    );
    return res?.items ?? [];
}

export async function getSubTopic(
    topicId: string,
    subTopicId: string
): Promise<SubTopicItem | null> {
    return api.get<SubTopicItem>(`/api/sub-topics/${subTopicId}`);
}
