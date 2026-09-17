export type TimetableTemplate = {
    id: string;
    school_id: string;
    name: string;
    description?: string | null;
    created_at: string;
    updated_at: string;
};

export type TimeSlot = {
    id: string;
    timetable_template_id: string;
    name: string;
    start_time: string; // HH:mm
    end_time: string; // HH:mm
    sequence_index: number;
    is_instructional: boolean;
};

export type TimeSlotDraft = {
    id: string; // temp uuid
    name: string;
    start_time: string;
    end_time: string;
    is_instructional: boolean;
};

export type DraftTemplate = {
    name: string;
    description?: string;
    slots: TimeSlotDraft[];
};

export type CreateTimetableTemplatePayload = {
    name: string;
    description?: string;
    time_slots: Array<{
        name: string;
        start_time: string;
        end_time: string;
        is_instructional: boolean;
    }>;
};

export type ClassTimetableSlotWithDetails = {
    id: string;
    school_id: string;
    class_room_id: string;
    academic_term_id: string;
    day_of_week: number;
    time_slot_id: string;
    subject_id: string;
    teacher_membership_id: string;
    room_id?: string | null;
    created_at: string;
    updated_at: string;
    class_name: string;
    class_stream?: string | null;
    grade_name?: string | null;
    time_slot_name: string;
    start_time: string;
    end_time: string;
    sequence_index: number;
    is_instructional: boolean;
    subject_name?: string | null;
    teacher_name?: string | null;
    teacher_email?: string | null;
    room_name?: string | null;
};
