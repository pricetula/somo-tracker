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
