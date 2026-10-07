import { useMemo, useState } from "react";
import type { TimeSlotDraft } from "../types/timetable-template";

const DEFAULT_DURATION_MIN = 60;

function addMinutes(time: string, minutes: number): string {
    const [h, m] = time.split(":").map(Number);
    const date = new Date();
    date.setHours(h, m, 0, 0);
    date.setMinutes(date.getMinutes() + minutes);
    const hh = String(date.getHours()).padStart(2, "0");
    const mm = String(date.getMinutes()).padStart(2, "0");
    return `${hh}:${mm}`;
}

function formatSlotName(index: number, isInstructional: boolean): string {
    if (!isInstructional && index > 0) return "Break";
    return `Period ${index + 1}`;
}

export function useTimetableWizard(initialSlots?: TimeSlotDraft[]) {
    const [slots, setSlots] = useState<TimeSlotDraft[]>(() => {
        if (initialSlots && initialSlots.length) return initialSlots;
        return [
            {
                id: crypto.randomUUID(),
                name: "Period 1",
                start_time: "08:00",
                end_time: "09:00",
                is_instructional: true,
            },
        ];
    });

    const updateSlot = (id: string, patch: Partial<TimeSlotDraft>) => {
        setSlots((prev) => {
            const idx = prev.findIndex((s) => s.id === id);
            if (idx === -1) return prev;
            const next = [...prev];
            const updated = { ...next[idx], ...patch };
            next[idx] = updated;
            // Cascade start time to next slot if end_time changed
            if (patch.end_time && idx < next.length - 1) {
                const nextSlot = next[idx + 1];
                // Only cascade if next slot start is still equal to previous end (i.e., not manually edited)
                // For simplicity, always cascade
                next[idx + 1] = { ...nextSlot, start_time: patch.end_time };
            }
            return next;
        });
    };

    const addSlot = () => {
        setSlots((prev) => {
            const last = prev[prev.length - 1];
            const start = last?.end_time ?? "09:00";
            const end = addMinutes(start, DEFAULT_DURATION_MIN);
            const newSlot: TimeSlotDraft = {
                id: crypto.randomUUID(),
                name: formatSlotName(prev.length, true),
                start_time: start,
                end_time: end,
                is_instructional: true,
            };
            return [...prev, newSlot];
        });
    };

    const deleteSlot = (id: string) => {
        setSlots((prev) => {
            if (prev.length <= 1) return prev;
            return prev.filter((s) => s.id !== id);
        });
    };

    const slotsValid = useMemo(() => {
        return slots.every((s) => {
            const start = s.start_time;
            const end = s.end_time;
            if (!start || !end) return false;
            return start < end;
        });
    }, [slots]);

    const hasGapsOrOverlap = useMemo(() => {
        for (let i = 0; i < slots.length - 1; i++) {
            const curr = slots[i];
            const next = slots[i + 1];
            if (curr.end_time !== next.start_time) return true;
        }
        return false;
    }, [slots]);

    return {
        slots,
        updateSlot,
        addSlot,
        deleteSlot,
        slotsValid,
        hasGapsOrOverlap,
        setSlots,
    };
}
