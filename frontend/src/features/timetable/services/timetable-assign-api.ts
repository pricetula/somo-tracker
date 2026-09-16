import { api } from "@/lib/api/client";

export type SetupTimetableSlotPayload = {
    class_room_id: string;
    day_of_week: number;
    time_slot_id: string;
    subject_id: string;
    teacher_membership_id: string;
    room_id?: string;
};

export async function setupTimetableSlot(
    payload: SetupTimetableSlotPayload
): Promise<{ code: string; message: string }> {
    return api.post<{ code: string; message: string }>(`/api/timetable/setup`, payload);
}
