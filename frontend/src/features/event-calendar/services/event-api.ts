import { api } from "@/lib/api/client";
import type { Event } from "@/features/event-calendar/types/event.types";

/**
 * Fetch school events for a given date range.
 * Backend endpoint: GET /api/events?from=YYYY-MM-DD&to=YYYY-MM-DD
 */
export async function listEvents(start: string, end: string): Promise<Event[]> {
    return api.get<Event[]>(`/api/events?from=${start}&to=${end}`);
}

/**
 * Create a new school event.
 * Backend endpoint: POST /api/events
 */
export async function createEvent(input: {
    title: string;
    event_type: string;
    start_date: string;
    end_date: string;
    requires_attendance: boolean;
}): Promise<Event> {
    return api.post<Event>(`/api/events`, input);
}

export const eventApi = {
    list: listEvents,
    create: createEvent,
};

export default eventApi;
