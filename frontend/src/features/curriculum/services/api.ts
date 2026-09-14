import type { SubjectListItem, TopicItem, SubTopicItem } from "../types/curriculum";
import { mockSubjects, mockTopics, mockSubTopics } from "./mock-data";

const delay = (ms: number) => new Promise((res) => setTimeout(res, ms));

export async function listSubjects(params: {
    page?: number;
    limit?: number;
    search?: string;
    filters?: Record<string, string | string[]>;
}): Promise<{ items: SubjectListItem[]; total: number }> {
    await delay(150);
    let items = [...mockSubjects];
    if (params.search) {
        const q = params.search.toLowerCase();
        items = items.filter(
            (c) => c.name.toLowerCase().includes(q) || c.code.toLowerCase().includes(q)
        );
    }
    if (params.filters?.grade) {
        const grade = String(params.filters.grade);
        if (grade !== "all") {
            items = items.filter((c) => c.grade === grade);
        }
    }
    const page = params.page ?? 1;
    const limit = params.limit ?? 50;
    const total = items.length;
    const start = (page - 1) * limit;
    items = items.slice(start, start + limit);
    return { items, total };
}

export async function getSubject(id: string): Promise<SubjectListItem | null> {
    await delay(100);
    return mockSubjects.find((c) => c.id === id) ?? null;
}

export async function listTopics(subjectId: string): Promise<TopicItem[]> {
    await delay(100);
    return mockTopics[subjectId] ?? [];
}

export async function listSubTopics(topicId: string): Promise<SubTopicItem[]> {
    await delay(100);
    return mockSubTopics[topicId] ?? [];
}

export async function getSubTopic(
    topicId: string,
    subTopicId: string
): Promise<SubTopicItem | null> {
    await delay(100);
    return mockSubTopics[topicId]?.find((s) => s.id === subTopicId) ?? null;
}
