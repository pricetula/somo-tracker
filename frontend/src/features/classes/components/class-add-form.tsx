"use client";

import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import {
    Form,
    FormControl,
    FormField,
    FormItem,
    FormLabel,
    FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { StreamsCombobox } from "@/features/streams/components/streams-combobox";
import { GradesCombobox } from "@/features/grades/components/grades-combobox";
import { createClass } from "../services/api";

const schema = z.object({
    name: z.string().min(1, "Name is required"),
    gradeId: z.string().min(1, "Grade is required"),
    streamId: z.string().min(1, "Stream is required"),
});

type FormValues = z.infer<typeof schema>;

export function ClassAddForm() {
    const router = useRouter();
    const form = useForm<FormValues>({
        resolver: zodResolver(schema),
        defaultValues: { name: "", gradeId: "", streamId: "" },
    });

    const onSubmit = async (values: FormValues) => {
        await createClass(values);
        router.back();
    };

    return (
        <Form {...form}>
            <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
                <FormField
                    control={form.control}
                    name="name"
                    render={({ field }) => (
                        <FormItem>
                            <FormLabel>Name</FormLabel>
                            <FormControl>
                                <Input placeholder="Class name" {...field} />
                            </FormControl>
                            <FormMessage />
                        </FormItem>
                    )}
                />
                <FormField
                    control={form.control}
                    name="gradeId"
                    render={({ field }) => (
                        <FormItem>
                            <FormLabel>Grade</FormLabel>
                            <FormControl>
                                <GradesCombobox value={field.value} onChange={field.onChange} />
                            </FormControl>
                            <FormMessage />
                        </FormItem>
                    )}
                />
                <FormField
                    control={form.control}
                    name="streamId"
                    render={({ field }) => (
                        <FormItem>
                            <FormLabel>Stream</FormLabel>
                            <FormControl>
                                <StreamsCombobox value={field.value} onChange={field.onChange} />
                            </FormControl>
                            <FormMessage />
                        </FormItem>
                    )}
                />
                <div className="flex justify-end gap-2 pt-2">
                    <Button type="button" variant="outline" onClick={() => router.back()}>
                        Cancel
                    </Button>
                    <Button type="submit">Create</Button>
                </div>
            </form>
        </Form>
    );
}
