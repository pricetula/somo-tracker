import { useState, useCallback } from "react";
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
import { useTimetableWizard } from "../hooks/use-timetable-wizard";
import { TimetableGrid } from "./timetable-grid";
import { useCreateTimetableTemplate } from "../hooks/use-timetable-templates";

const metadataSchema = z.object({
    name: z.string().min(1, "Template name is required"),
    description: z.string().optional(),
});

type MetadataForm = z.infer<typeof metadataSchema>;

export function TimetableTemplateWizard() {
    const [step, setStep] = useState<1 | 2>(1);

    const form = useForm<MetadataForm>({
        resolver: zodResolver(metadataSchema),
        defaultValues: { name: "", description: "" },
    });

    const { slots, updateSlot, addSlot, deleteSlot, slotsValid, hasGapsOrOverlap } =
        useTimetableWizard();

    const createMutation = useCreateTimetableTemplate();

    const canProceed = form.getValues().name.trim().length > 0;

    const handleNext = useCallback(() => {
        form.trigger().then((valid) => {
            if (valid) setStep(2);
        });
    }, [form]);

    const handleBack = useCallback(() => setStep(1), []);

    const handleSave = useCallback(() => {
        const values = form.getValues();
        if (!slotsValid || hasGapsOrOverlap) return;
        createMutation.mutate({
            name: values.name.trim(),
            description: values.description?.trim() || undefined,
            time_slots: slots.map((s) => ({
                name: s.name,
                start_time: s.start_time,
                end_time: s.end_time,
                is_instructional: s.is_instructional,
            })),
        });
    }, [form, slots, slotsValid, hasGapsOrOverlap, createMutation]);

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

    return (
        <div className="space-y-4">
            <div>
                <h2 className="text-xl font-semibold">Configure Time Slots & Weekly Matrix</h2>
                <p className="text-muted-foreground">
                    Define your daily bell schedule sequence. Duration and start times auto-cascade.
                </p>
            </div>

            <TimetableGrid
                slots={slots}
                editable={true}
                onSlotChange={(id, patch) => updateSlot(id, patch)}
                onSlotDelete={(id) => deleteSlot(id)}
                onAddSlot={addSlot}
            />

            <div className="flex gap-2">
                <Button variant="secondary" onClick={handleBack}>
                    Back
                </Button>
                <Button
                    onClick={handleSave}
                    disabled={createMutation.isPending || !slotsValid || hasGapsOrOverlap}
                >
                    {createMutation.isPending ? "Saving..." : "Save Template"}
                </Button>
            </div>
        </div>
    );
}
