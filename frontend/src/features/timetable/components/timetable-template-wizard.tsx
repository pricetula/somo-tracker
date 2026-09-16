import { useState, useMemo, useCallback } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import {
    Form,
    FormControl,
    FormField,
    FormItem,
    FormLabel,
    FormMessage,
} from "@/components/ui/form";
import { toast } from "sonner";
import { getErrorMessage } from "@/lib/errors";
import { useTimetableWizard } from "../hooks/use-timetable-wizard";
import { TimeSlotRow } from "./time-slot-row";
import { createTimetableTemplate } from "../services/timetable-api";
import { useRouter } from "next/navigation";

const metadataSchema = z.object({
    name: z.string().min(1, "Template name is required"),
    description: z.string().optional(),
});

type MetadataForm = z.infer<typeof metadataSchema>;

export function TimetableTemplateWizard({ schoolId }: { schoolId: string }) {
    const [step, setStep] = useState<1 | 2>(1);
    const [isSaving, setIsSaving] = useState(false);
    const router = useRouter();

    const form = useForm<MetadataForm>({
        resolver: zodResolver(metadataSchema),
        defaultValues: { name: "", description: "" },
    });

    const { slots, updateSlot, addSlot, deleteSlot, slotsValid } = useTimetableWizard();

    const canProceed = useMemo(
        () => form.formState.isValid && form.getValues("name").trim().length > 0,
        [form]
    );

    const handleNext = useCallback(() => {
        form.trigger().then((valid) => {
            if (valid) setStep(2);
        });
    }, [form]);

    const handleBack = useCallback(() => setStep(1), []);

    const handleSave = useCallback(async () => {
        const values = form.getValues();
        if (!values.name.trim()) {
            toast.error("Template name is required");
            return;
        }
        if (!slotsValid) {
            toast.error("Check slot times");
            return;
        }
        setIsSaving(true);
        try {
            await createTimetableTemplate(schoolId, {
                name: values.name.trim(),
                description: values.description?.trim() || undefined,
                time_slots: slots.map((s) => ({
                    name: s.name,
                    start_time: s.start_time,
                    end_time: s.end_time,
                    is_instructional: s.is_instructional,
                })),
            });
            toast.success("Timetable template saved");
            router.push(`/dashboard/schools/${schoolId}/timetables`);
        } catch (err) {
            toast.error(getErrorMessage(err));
        } finally {
            setIsSaving(false);
        }
    }, [form, slots, slotsValid, schoolId, router]);

    if (step === 1) {
        return (
            <div className="max-w-xl space-y-5">
                <h2 className="text-xl font-semibold">Create Timetable Template</h2>
                <Form {...form}>
                    <form className="space-y-4" onSubmit={(e) => e.preventDefault()}>
                        <FormField
                            control={form.control}
                            name="name"
                            render={({ field }) => (
                                <FormItem>
                                    <FormLabel>Template Name</FormLabel>
                                    <FormControl>
                                        <Input placeholder="Standard 6-Period Day" {...field} />
                                    </FormControl>
                                    <FormMessage />
                                </FormItem>
                            )}
                        />
                        <FormField
                            control={form.control}
                            name="description"
                            render={({ field }) => (
                                <FormItem>
                                    <FormLabel>Description</FormLabel>
                                    <FormControl>
                                        <Textarea placeholder="Optional description" {...field} />
                                    </FormControl>
                                    <FormMessage />
                                </FormItem>
                            )}
                        />
                        <div className="flex justify-end">
                            <Button type="button" onClick={handleNext} disabled={!canProceed}>
                                Next
                            </Button>
                        </div>
                    </form>
                </Form>
            </div>
        );
    }

    const days = ["Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"];

    return (
        <div className="space-y-4">
            <div>
                <h2 className="text-xl font-semibold">Configure Time Slots & Weekly Matrix</h2>
                <p className="text-muted-foreground text-sm">
                    Define your daily bell schedule sequence. Duration and start times auto-cascade.
                </p>
            </div>

            <div className="overflow-x-auto rounded-md border">
                <table className="w-full text-sm">
                    <thead>
                        <tr className="border-b">
                            <th className="bg-background sticky top-0 left-0 z-30 w-96 border-r px-4 py-3 text-left font-medium">
                                Slot Configuration
                            </th>
                            {days.map((d) => (
                                <th
                                    key={d}
                                    className="bg-background sticky top-0 z-20 min-w-56 border-r px-4 py-3 text-left font-medium"
                                >
                                    {d}
                                </th>
                            ))}
                        </tr>
                    </thead>
                    <tbody>
                        {slots.map((slot) => (
                            <tr key={slot.id} className="border-b align-top">
                                <td className="bg-background sticky left-0 z-10 border-r">
                                    <TimeSlotRow
                                        slot={slot}
                                        onChange={(patch) => updateSlot(slot.id, patch)}
                                        onDelete={() => deleteSlot(slot.id)}
                                        canDelete={slots.length > 1}
                                    />
                                </td>
                                {days.map((d) => (
                                    <td
                                        key={d}
                                        className={`border-r px-4 align-top ${!slot.is_instructional ? "bg-row-disabled" : ""}`}
                                    />
                                ))}
                            </tr>
                        ))}
                    </tbody>
                </table>
            </div>

            <div className="flex items-center justify-between">
                <div>
                    <Button variant="outline" onClick={addSlot}>
                        + Add Time Slot
                    </Button>
                </div>
                <div className="flex gap-2">
                    <Button variant="secondary" onClick={handleBack}>
                        Back
                    </Button>
                    <Button onClick={handleSave} disabled={isSaving || !slotsValid}>
                        {isSaving ? "Saving..." : "Save Template"}
                    </Button>
                </div>
            </div>
        </div>
    );
}
