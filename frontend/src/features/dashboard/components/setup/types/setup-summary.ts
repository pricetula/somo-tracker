export type AcademicPeriod = {
    exists: boolean;
    status: string;
    yearId?: string;
    yearName?: string;
    termId?: string;
    termName?: string;
    termStart?: string;
    termEnd?: string;
};

export type ClassroomItem = {
    id: string;
    name: string;
    grade: string;
    stream: string;
};

export type Classrooms = {
    count: number;
    items: ClassroomItem[];
};

export type EnrollmentItem = {
    classId: string;
    className: string;
    grade: string;
    stream: string;
    studentCount: number;
};

export type Enrollments = {
    items: EnrollmentItem[];
};

export type StreamsInfo = {
    count: number;
};

export type SetupSummary = {
    academicPeriod: AcademicPeriod;
    streams: StreamsInfo | null;
    classrooms: Classrooms | null;
    enrollments: Enrollments | null;
};
