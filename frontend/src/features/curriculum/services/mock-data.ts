import type { SubjectListItem, TopicItem, SubTopicItem } from "../types/curriculum";

export const mockSubjects: SubjectListItem[] = [
    { id: "1", name: "Primary Mathematics", code: "MATH-PP1", grade: "PP1" },
    { id: "2", name: "Primary English", code: "ENG-PP1", grade: "PP1" },
    { id: "3", name: "Primary Science", code: "SCI-PP1", grade: "PP1" },
    { id: "4", name: "Grade 1 Mathematics", code: "MATH-G1", grade: "Grade 1" },
    { id: "5", name: "Grade 1 English", code: "ENG-G1", grade: "Grade 1" },
    { id: "6", name: "Grade 10 Mathematics", code: "MATH-G10", grade: "Grade 10" },
    { id: "7", name: "Grade 10 Physics", code: "PHY-G10", grade: "Grade 10" },
    { id: "8", name: "Grade 12 Biology", code: "BIO-G12", grade: "Grade 12" },
];

export const mockTopics: Record<string, TopicItem[]> = {
    "1": [
        {
            id: "s1",
            subjectId: "1",
            name: "Numbers",
            code: "NUM-01",
            description: "Counting and operations",
        },
        {
            id: "s2",
            subjectId: "1",
            name: "Shapes",
            code: "SHAPE-01",
            description: "Basic shapes recognition",
        },
    ],
    "4": [
        { id: "s3", subjectId: "4", name: "Addition", code: "ADD-01", description: "" },
        { id: "s4", subjectId: "4", name: "Subtraction", code: "SUB-01", description: "" },
    ],
};

export const mockSubTopics: Record<string, SubTopicItem[]> = {
    s1: [
        { id: "t1", topicId: "s1", subjectId: "1", name: "Counting 1-10", code: "CNT-01" },
        { id: "t2", topicId: "s1", subjectId: "1", name: "Number recognition", code: "NR-01" },
    ],
    s3: [
        { id: "t3", topicId: "s3", subjectId: "4", name: "Single digit addition", code: "ADD-SD" },
    ],
};
