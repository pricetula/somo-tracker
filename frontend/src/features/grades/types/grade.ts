export interface GradeLevel {
    id: string;
    education_system_id: string;
    local_label: string;
    tier_stage: string;
    sequence_index: number;
}

export interface GradesResponse {
    code: string;
    message: string;
    grades: GradeLevel[];
    errors: Record<string, string[]>;
}
