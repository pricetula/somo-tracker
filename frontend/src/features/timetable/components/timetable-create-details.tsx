"use client";

import { Button } from "@/components/ui/button";
import { FormField, FormItem, FormLabel, FormControl, FormMessage } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { type Control } from "react-hook-form";
import type { CreateTimetableFormData } from "./timetable-create";

export interface DetailsStepProps {
    control: Control<CreateTimetableFormData>;
    onNext: () => void;
    onCancel: () => void;
}

export function TimetableCreateDetails({ control, onNext, onCancel }: DetailsStepProps) {
    return (
        <div className="space-y-4">
            <FormField
                control={control}
                name="name"
                render={({ field }) => (
                    <FormItem>
                        <FormLabel>Timetable Name</FormLabel>
                        <FormControl>
                            <Input {...field} placeholder="e.g., 2024 Academic Year" />
                        </FormControl>
                        <FormMessage />
                    </FormItem>
                )}
            />

            <FormField
                control={control}
                name="description"
                render={({ field }) => (
                    <FormItem>
                        <FormLabel>Description (Optional)</FormLabel>
                        <FormControl>
                            <Textarea
                                {...field}
                                placeholder="Brief description of this timetable"
                                rows={3}
                            />
                        </FormControl>
                        <p className="text-muted-foreground text-xs">Maximum 500 characters.</p>
                    </FormItem>
                )}
            />

            <div className="flex justify-end gap-2 pt-2">
                <Button type="button" variant="outline" onClick={onCancel}>
                    Cancel
                </Button>
                <Button type="button" onClick={onNext}>
                    Next
                </Button>
            </div>
        </div>
    );
}
