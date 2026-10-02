"use client";

import { useEffect, useState } from "react";
import { format } from "date-fns";
import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
    DialogFooter,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select";
import { Label } from "@/components/ui/label";
import { DatePicker } from "@/components/ui/date-picker";
import { useCreateEvent } from "../hooks/useCreateEvent";
import { useUpdateEvent } from "../hooks/useUpdateEvent";
import { useDeleteEvent } from "../hooks/useDeleteEvent";

import type { Event } from "../types/event.types";

export function EventCreateDialog({
    open,
    onOpenChange,
    event,
}: {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    event?: Event | null;
}) {
    const isEditing = !!event?.id;
    const [title, setTitle] = useState("");
    const [eventType, setEventType] = useState("SPORTS");
    const [startDate, setStartDate] = useState(format(new Date(), "yyyy-MM-dd"));
    const [endDate, setEndDate] = useState(format(new Date(), "yyyy-MM-dd"));
    const createEvent = useCreateEvent();
    const updateEvent = useUpdateEvent();
    const deleteEvent = useDeleteEvent();

    useEffect(() => {
        if (!open) return;
        const resetForm = () => {
            if (event) {
                setTitle(event.title ?? "");
                setEventType(event.event_type ?? "SPORTS");
                setStartDate(event.start_date ?? format(new Date(), "yyyy-MM-dd"));
                setEndDate(event.end_date ?? format(new Date(), "yyyy-MM-dd"));
            } else {
                setTitle("");
                setEventType("SPORTS");
                setStartDate(format(new Date(), "yyyy-MM-dd"));
                setEndDate(format(new Date(), "yyyy-MM-dd"));
            }
        };
        // Defer state updates to avoid synchronous cascading renders
        Promise.resolve().then(resetForm);
    }, [open, event]);

    const handleSave = () => {
        const payload = {
            title,
            event_type: eventType,
            start_date: startDate,
            end_date: endDate,
            requires_attendance: false,
        };
        if (isEditing && event?.id) {
            updateEvent.mutate(
                { id: event.id, input: payload },
                {
                    onSuccess: () => onOpenChange(false),
                }
            );
        } else {
            createEvent.mutate(payload, {
                onSuccess: () => onOpenChange(false),
            });
        }
    };

    return (
        <Dialog open={open} onOpenChange={onOpenChange}>
            <DialogContent className="md:max-w-lg">
                <DialogHeader>
                    <DialogTitle>{isEditing ? "Update Event" : "Add Event"}</DialogTitle>
                </DialogHeader>
                <div className="space-y-4 py-2">
                    <div className="grid grid-cols-2 gap-4">
                        <div className="space-y-2">
                            <Label htmlFor="title">Title</Label>
                            <Input
                                id="title"
                                placeholder="Event title"
                                value={title}
                                onChange={(e) => setTitle(e.target.value)}
                            />
                        </div>
                        <div className="w-full space-y-2">
                            <Label htmlFor="type">Event type</Label>
                            <Select value={eventType} onValueChange={(e) => setEventType(e ?? "")}>
                                <SelectTrigger id="type" className="w-full">
                                    <SelectValue placeholder="Select type" />
                                </SelectTrigger>
                                <SelectContent>
                                    <SelectItem value="SPORTS">SPORTS</SelectItem>
                                    <SelectItem value="EXAM">EXAM</SelectItem>
                                    <SelectItem value="ADMISSION">ADMISSION</SelectItem>
                                    <SelectItem value="MEETING">MEETING</SelectItem>
                                </SelectContent>
                            </Select>
                        </div>
                    </div>
                    <div className="grid grid-cols-2 gap-4">
                        <div className="space-y-2">
                            <Label htmlFor="start">Start date</Label>
                            <DatePicker
                                id="start"
                                value={startDate}
                                onChange={setStartDate}
                                placeholder="Pick start date"
                                className="w-full"
                            />
                        </div>
                        <div className="space-y-2">
                            <Label htmlFor="end">End date</Label>
                            <DatePicker
                                id="end"
                                value={endDate}
                                onChange={setEndDate}
                                placeholder="Pick end date"
                                className="w-full"
                            />
                        </div>
                    </div>
                </div>
                <DialogFooter>
                    <Button variant="outline" onClick={() => onOpenChange(false)}>
                        Cancel
                    </Button>
                    <Button
                        onClick={handleSave}
                        disabled={!title || createEvent.isPending || updateEvent.isPending}
                    >
                        {isEditing ? "Update" : "Save"}
                    </Button>
                    {isEditing && (
                        <Button
                            variant="destructive"
                            onClick={() => {
                                if (window.confirm("Are you sure you want to delete this event?")) {
                                    if (event?.id) {
                                        deleteEvent.mutate(event.id);
                                    }
                                    onOpenChange(false);
                                }
                            }}
                            className="ml-2"
                        >
                            Delete
                        </Button>
                    )}
                </DialogFooter>
            </DialogContent>
        </Dialog>
    );
}
