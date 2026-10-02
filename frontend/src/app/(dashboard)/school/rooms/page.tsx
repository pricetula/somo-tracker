"use client";

import { RoomsTable } from "@/features/rooms";

export default function RoomsPage() {
    return (
        <div className="space-y-4 p-6">
            <h1 className="text-2xl font-semibold">Rooms</h1>
            <RoomsTable />
        </div>
    );
}
