export type SubstitutionStatus = "PENDING" | "ASSIGNED" | "COMPLETED" | "CANCELLED";

export interface TimetableSubstitution {
    id: string;
    schoolId: string;
    classTimetableSlotId: string;
    substitutionDate: string; // YYYY-MM-DD
    originalTeacherMembershipId: string;
    substituteTeacherMembershipId?: string;
    status: SubstitutionStatus;
    reason?: string;
    className: string;
    subjectName: string;
    originalTeacherName: string;
    substituteTeacherName?: string;
    timeSlotName: string;
    startTime: string; // HH:mm
    endTime: string; // HH:mm
    createdAt: string;
    updatedAt: string;
}

export interface ListSubstitutionsParams {
    page?: number;
    limit?: number;
    dateFrom?: string;
    dateTo?: string;
    status?: SubstitutionStatus | "";
}

export interface ListSubstitutionsResponse {
    items: TimetableSubstitution[];
    total: number;
    page: number;
    limit: number;
}

export interface CreateSubstitutionRequest {
    classTimetableSlotId: string;
    substitutionDate: string;
    originalTeacherMembershipId: string;
    substituteTeacherMembershipId?: string;
    status?: SubstitutionStatus;
    reason?: string;
}

export interface UpdateSubstitutionRequest {
    substituteTeacherMembershipId?: string;
    status?: SubstitutionStatus;
    reason?: string;
}
