"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select";
import {
    Form,
    FormField,
    FormItem,
    FormLabel,
    FormControl,
    FormMessage,
} from "@/components/ui/form";
import { useCreateRoom, useUpdateRoom } from "../hooks/use-rooms";
import type { CreateRoomRequest } from "../types/room";

const schema = z.object({
    name: z.string().min(1, "Name is required"),
    capacity: z.coerce.number().int().positive("Capacity must be a positive integer").optional(),
    roomType: z.enum(["STANDARD", "SCIENCE_LAB", "COMPUTER_LAB", "GYM"]),
});

type FormValues = z.infer<typeof schema>;

interface RoomFormProps {
    onSuccess: () => void;
    defaultValues?: Partial<FormValues>;
    isEditing?: boolean;
    roomId?: string;
}

export function RoomForm({ onSuccess, defaultValues, isEditing = false, roomId }: RoomFormProps) {
    const createRoom = useCreateRoom();
    const updateRoom = useUpdateRoom();
    const isPending = isEditing ? updateRoom.isPending : createRoom.isPending;

    const form = useForm<FormValues>({
        resolver: zodResolver(schema),
        defaultValues: { name: "", roomType: "STANDARD", ...defaultValues },
    });

    const onSubmit = form.handleSubmit((values) => {
        if (isEditing && roomId) {
            updateRoom.mutate({ id: roomId, ...values }, { onSuccess });
        } else {
            createRoom.mutate(values as CreateRoomRequest, { onSuccess });
        }
    });

    return (
        <Form {...form}>
            <form onSubmit={onSubmit} className="space-y-4">
                <FormField
                    control={form.control}
                    name="name"
                    render={({ field }) => (
                        <FormItem>
                            <FormLabel>Name</FormLabel>
                            <FormControl>
                                <Input {...field} placeholder="e.g. Lab A, Room 204" />
                            </FormControl>
                            <FormMessage />
                        </FormItem>
                    )}
                />
                <FormField
                    control={form.control}
                    name="capacity"
                    render={({ field }) => (
                        <FormItem>
                            <FormLabel>Capacity</FormLabel>
                            <FormControl>
                                <Input {...field} type="number" placeholder="e.g. 30" />
                            </FormControl>
                            <FormMessage />
                        </FormItem>
                    )}
                />
                <FormField
                    control={form.control}
                    name="roomType"
                    render={({ field }) => (
                        <FormItem>
                            <FormLabel>Room Type</FormLabel>
                            <FormControl>
                                <Select onValueChange={field.onChange} defaultValue={field.value}>
                                    <SelectTrigger>
                                        <SelectValue placeholder="Select type" />
                                    </SelectTrigger>
                                    <SelectContent>
                                        <SelectItem value="STANDARD">Standard</SelectItem>
                                        <SelectItem value="SCIENCE_LAB">Science Lab</SelectItem>
                                        <SelectItem value="COMPUTER_LAB">Computer Lab</SelectItem>
                                        <SelectItem value="GYM">Gym</SelectItem>
                                    </SelectContent>
                                </Select>
                            </FormControl>
                            <FormMessage />
                        </FormItem>
                    )}
                />
                <Button type="submit" disabled={isPending} className="w-full">
                    {isPending
                        ? isEditing
                            ? "Updating..."
                            : "Creating..."
                        : isEditing
                          ? "Update"
                          : "Create"}
                </Button>
            </form>
        </Form>
    );
}
