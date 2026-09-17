import type { SubjectListItem, TopicItem, SubTopicItem } from "../types/curriculum";
import { api } from "@/lib/api/client";

export async function listSubjects(params: {
    page?: number;
    limit?: number;
    search?: string;
    filters?: Record<string, string | string[]>;
}): Promise<{ items: SubjectListItem[]; total: number }> {
    const searchParams = new URLSearchParams();
    if (params.page) searchParams.set("page", String(params.page));
    if (params.limit) searchParams.set("limit", String(params.limit));
    if (params.search) searchParams.set("search", params.search);
    if (params.filters?.grade) searchParams.set("grade", String(params.filters.grade));
    const query = searchParams.toString() ? `?${searchParams.toString()}` : "";
    return api.get<{ items: SubjectListItem[]; total: number }>(`/api/curriculum/subjects${query}`);
}

export async function getSubject(id: string): Promise<SubjectListItem | null> {
    return api.get<SubjectListItem>(`/api/curriculum/subjects/${id}`);
}

export async function listTopics(subjectId: string): Promise<TopicItem[]> {
    return api.get<TopicItem[]>(`/api/curriculum/subjects/${subjectId}/topics`);
}

export async function listSubTopics(topicId: string): Promise<SubTopicItem[]> {
    return api.get<SubTopicItem[]>(`/api/curriculum/topics/${topicId}/subtopics`);
}

export async function getSubTopic(
    topicId: string,
    subTopicId: string
): Promise<SubTopicItem | null> {
    return api.get<SubTopicItem>(`/api/curriculum/topics/${topicId}/subtopics/${subTopicId}`);
}
