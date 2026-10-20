"use client";

import { useRoom } from "../hooks/use-rooms";

interface RoomDetailProps {
    id: string;
}

export function RoomDetail({ id }: RoomDetailProps) {
    const { data: room, isLoading } = useRoom(id);

    if (isLoading || !room) {
        return (
            <div className="space-y-6 p-6">
                <p className="text-muted-foreground">Loading…</p>
            </div>
        );
    }

    return (
        <div className="space-y-6 p-6">
            <div className="space-y-1">
                <h1 className="text-2xl font-semibold">{room.name}</h1>
                <p className="text-muted-foreground">ID: {room.id}</p>
            </div>
            <div className="space-y-2">
                <div className="grid grid-cols-2 gap-4">
                    <div>
                        <span className="font-medium">Type:</span> {room.roomType.replace("_", " ")}
                    </div>
                    <div>
                        <span className="font-medium">Capacity:</span>{" "}
                        {room.capacity?.toString() ?? "—"}
                    </div>
                    <div>
                        <span className="font-medium">Created:</span> {room.createdAt}
                    </div>
                    <div>
                        <span className="font-medium">Updated:</span> {room.updatedAt}
                    </div>
                </div>
            </div>
        </div>
    );
}
