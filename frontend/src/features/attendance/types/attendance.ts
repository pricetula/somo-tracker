export type AttendanceStatus = "SUBMITTED" | "IN_PROGRESS" | "MISSED" | "NO_STUDENTS";

export interface AttendanceSession {
    slotId: string;
    classId: string;
    className: string;
    grade: string;
    stream: string;
    subject: string;
    teacherName: string;
    teacherId: string;
    dayOfWeek: number;
    timeSlotName: string;
    startTime: string; // "08:00"
    endTime: string; // "09:00"
    attendanceDate: string; // "2025-01-15"
    status: AttendanceStatus;
    totalStudents: number;
    recordedStudents: number;
}

export interface ListAttendanceSessionsParams {
    page?: number;
    limit?: number;
    search?: string;
    status?: AttendanceStatus | "";
    dateFrom?: string;
    dateTo?: string;
    grades?: string[];
    streams?: string[];
    subjects?: string[];
    teachers?: string[];
}

export interface ListAttendanceSessionsResponse {
    items: AttendanceSession[];
    total: number;
    page: number;
    limit: number;
}
