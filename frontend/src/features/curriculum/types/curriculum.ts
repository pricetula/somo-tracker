export interface SubjectListItem {
    id: string;
    name: string;
    code: string;
    grade: string;
}

export interface TopicItem {
    id: string;
    subjectId: string;
    name: string;
    code: string;
    description?: string;
}

export interface SubTopicItem {
    id: string;
    topicId: string;
    subjectId: string;
    name: string;
    code: string;
}
