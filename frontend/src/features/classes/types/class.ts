export interface ClassListItem {
    id: string;
    name: string;
    grade: string;
    stream: string;
    academicYear: string;
    teacherName?: string;
}

export interface ClassDetail extends ClassListItem {
    description?: string;
    studentsCount: number;
}
