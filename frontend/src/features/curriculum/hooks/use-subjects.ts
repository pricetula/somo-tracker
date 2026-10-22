import { useQuery } from "@tanstack/react-query";
import { listSubjects, getSubject, listTopics, listSubTopics, getSubTopic } from "../services/api";

export const subjectsKeys = {
    list: (params?: { page?: number; limit?: number }) => ["subjects", params] as const,
    detail: (id: string) => ["subjects", id] as const,
    topics: (subjectId: string) => ["subjects", subjectId, "topics"] as const,
    subtopics: (topicId: string) => ["subjects", topicId, "subtopics"] as const,
    subtopic: (id: string) => ["subjects", "subtopic", id] as const,
};

export function subjectsListQueryFn() {
    return (params: {
        page?: number;
        limit?: number;
        search?: string;
        filters?: Record<string, string | string[]>;
    }) => listSubjects(params);
}

export function useSubject(id: string) {
    return useQuery({
        queryKey: subjectsKeys.detail(id),
        queryFn: () => getSubject(id),
        enabled: !!id,
    });
}

export function useTopics(subjectId: string) {
    return useQuery({
        queryKey: subjectsKeys.topics(subjectId),
        queryFn: () => listTopics(subjectId),
        enabled: !!subjectId,
    });
}

export function useSubTopics(topicId: string) {
    return useQuery({
        queryKey: subjectsKeys.subtopics(topicId),
        queryFn: () => listSubTopics(topicId),
        enabled: !!topicId,
    });
}

export function useSubTopic(id: string) {
    return useQuery({
        queryKey: subjectsKeys.subtopic(id),
        queryFn: () => getSubTopic("", id),
        enabled: !!id,
    });
}

export function useSubjects(params: { page?: number; limit?: number } = { page: 1, limit: 100 }) {
    return useQuery({
        queryKey: subjectsKeys.list(params),
        queryFn: () => listSubjects(params),
        staleTime: 5 * 60 * 1000,
    });
}
