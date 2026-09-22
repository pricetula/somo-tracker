"use client";

import React, { useMemo } from "react";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Plus, Trash2 } from "lucide-react";
import { Form, FormControl, FormField, FormItem, FormMessage } from "@/components/ui/form";
import { useCreateStreams } from "@/features/streams/hooks/use-streams";
import { useGrades } from "@/features/grades/hooks/use-grades";
import { useMeSession } from "@/features/auth/hooks/use-me-session";

const schema = z.object({
    name: z.string().min(1, "Stream name is required"),
    color: z.string().min(1, "Color is required"),
});

type FormValues = z.infer<typeof schema>;

interface CreateStreamsProps {
    onSuccess: () => void;
}

export function CreateStreams({ onSuccess }: CreateStreamsProps) {
    const { data: me } = useMeSession();
    const { data: gradesData } = useGrades();
    const { mutate: createStreams, isPending } = useCreateStreams();
    const [names, setNames] = React.useState<string[]>([]);

    const grades = gradesData?.grades ?? [];
    const firstGrade = grades[0];

    const preview = useMemo(() => {
        if (names.length === 0 || !firstGrade) return "";
        const gradeLabel = firstGrade.local_label ?? "Grade 1";
        return `${gradeLabel} ${names[0]}`;
    }, [names, firstGrade]);

    const form = useForm<FormValues>({
        resolver: zodResolver(schema),
        defaultValues: { name: "", color: "#0050d1" },
    });

    const addName = (values: FormValues) => {
        const trimmed = values.name.trim();
        if (!trimmed) return;
        if (!names.includes(trimmed)) {
            setNames((prev) => [...prev, trimmed]);
        }
        form.reset({ name: "", color: values.color || "#0050d1" });
    };

    const removeName = (n: string) => {
        setNames((prev) => prev.filter((x) => x !== n));
    };

    const handleSubmit = () => {
        if (names.length === 0) return;
        const items = names.map((n) => ({ name: n, color: form.getValues().color || "#0050d1" }));
        createStreams(items, { onSuccess });
    };

    return (
        <div className="space-y-6">
            <h2 className="text-xl font-semibold">Create Streams</h2>

            <Form {...form}>
                <form onSubmit={form.handleSubmit(addName)} className="flex items-end gap-2">
                    <FormField
                        control={form.control}
                        name="name"
                        render={({ field }) => (
                            <FormItem className="flex-1">
                                <FormControl>
                                    <Input
                                        {...field}
                                        placeholder="Stream name, e.g. Blue"
                                        className="flex-1"
                                        onKeyDown={(e) => {
                                            if (e.key === "Enter") {
                                                e.preventDefault();
                                            }
                                        }}
                                    />
                                </FormControl>
                                <FormMessage />
                            </FormItem>
                        )}
                    />
                    <Button type="submit" variant="outline" size="icon">
                        <Plus className="h-4 w-4" />
                    </Button>
                </form>
            </Form>

            <div className="text-muted-foreground text-xs">
                Examples: Blue · Red · Green · Yellow · Purple
            </div>

            <div className="flex flex-wrap gap-2">
                {names.map((n) => (
                    <Badge key={n} variant="secondary" className="gap-1 py-1 pr-1 pl-2">
                        {n}
                        <button
                            onClick={() => removeName(n)}
                            className="hover:text-destructive ml-0.5 inline-flex items-center rounded-full"
                            aria-label={`Remove ${n}`}
                        >
                            <Trash2 className="h-3 w-3" />
                        </button>
                    </Badge>
                ))}
            </div>

            {names.length > 0 && (
                <div className="bg-muted text-muted-foreground rounded-md p-3">
                    Preview: <span className="text-foreground font-medium">{preview}</span>
                </div>
            )}

            <div className="pt-2">
                <Button onClick={handleSubmit} disabled={names.length === 0 || isPending}>
                    {isPending ? "Creating..." : "Create Streams"}
                </Button>
            </div>

            {me?.active_school_id && (
                <p className="text-muted-foreground text-xs">
                    Active school: {me.active_school_id}
                </p>
            )}
        </div>
    );
}
