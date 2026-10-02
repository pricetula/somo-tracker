import { api } from "@/lib/api/client";
import type {
    Room,
    RoomType,
    ListRoomsParams,
    ListRoomsResponse,
    CreateRoomRequest,
    UpdateRoomRequest,
} from "../types/room";

interface BackendRoom {
    id: string;
    school_id: string;
    name: string;
    capacity: number;
    room_type: string;
    created_at: string;
    updated_at: string;
}

interface BackendListResponse {
    items: BackendRoom[];
    total: number;
    page: number;
    limit: number;
}

function transformRoom(item: BackendRoom): Room {
    return {
        id: item.id,
        schoolId: item.school_id,
        name: item.name,
        capacity: item.capacity,
        roomType: item.room_type as RoomType,
        createdAt: item.created_at,
        updatedAt: item.updated_at,
    };
}

export async function listRooms(params: ListRoomsParams = {}): Promise<ListRoomsResponse> {
    const searchParams = new URLSearchParams();
    Object.entries(params).forEach(([key, value]) => {
        if (value === undefined || value === null || value === "") return;
        searchParams.set(key, String(value));
    });
    const response = await api.get<BackendListResponse>(
        `/api/school/rooms?${searchParams.toString()}`
    );
    return {
        items: response.items.map(transformRoom),
        total: response.total,
        page: response.page,
        limit: response.limit,
    };
}

export async function getRoom(id: string): Promise<Room> {
    const response = await api.get<{ room: BackendRoom }>(`/api/school/rooms/${id}`);
    return transformRoom(response.room);
}

export async function createRoom(data: CreateRoomRequest): Promise<Room> {
    const response = await api.post<{ room: BackendRoom }>("/api/school/rooms", data);
    return transformRoom(response.room);
}

export async function updateRoom(data: UpdateRoomRequest): Promise<Room> {
    const { id, ...payload } = data;
    const response = await api.patch<{ room: BackendRoom }>(`/api/school/rooms`, {
        id,
        ...payload,
    });
    return transformRoom(response.room);
}

export async function deleteRoom(id: string): Promise<void> {
    await api.delete(`/api/school/rooms/${id}`);
}
