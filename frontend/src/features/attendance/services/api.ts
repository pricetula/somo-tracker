import { api } from "@/lib/api/client";
import type {
    AttendanceSession,
    ListAttendanceSessionsParams,
    ListAttendanceSessionsResponse,
} from "../types/attendance";

interface BackendAttendanceSession {
    slot_id: string;
    class_id: string;
    class_name: string;
    grade: string;
    stream: string;
    subject: string;
    teacher_name: string;
    teacher_id: string;
    day_of_week: number;
    time_slot_name: string;
    start_time: string;
    end_time: string;
    attendance_date: string;
    status: string;
    total_students: number;
    recorded_students: number;
}

interface BackendResponse {
    items: BackendAttendanceSession[];
    total: number;
    page: number;
    limit: number;
}

function transformSession(item: BackendAttendanceSession): AttendanceSession {
    return {
        slotId: item.slot_id,
        classId: item.class_id,
        className: item.class_name,
        grade: item.grade,
        stream: item.stream,
        subject: item.subject,
        teacherName: item.teacher_name,
        teacherId: item.teacher_id,
        dayOfWeek: item.day_of_week,
        timeSlotName: item.time_slot_name,
        startTime: item.start_time,
        endTime: item.end_time,
        attendanceDate: item.attendance_date,
        status: item.status as AttendanceSession["status"],
        totalStudents: item.total_students,
        recordedStudents: item.recorded_students,
    };
}

export async function listAttendanceSessions(
    params: ListAttendanceSessionsParams = {}
): Promise<ListAttendanceSessionsResponse> {
    const searchParams = new URLSearchParams();
    Object.entries(params).forEach(([key, value]) => {
        if (value === undefined || value === null || value === "") return;
        if (Array.isArray(value)) {
            value.forEach((v) => searchParams.append(key, v));
        } else {
            searchParams.set(key, String(value));
        }
    });
    const response = await api.get<BackendResponse>(
        `/api/attendance/sessions?${searchParams.toString()}`
    );
    return {
        items: response.items.map(transformSession),
        total: response.total,
        page: response.page,
        limit: response.limit,
    };
}
