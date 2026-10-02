export type RoomType = "STANDARD" | "SCIENCE_LAB" | "COMPUTER_LAB" | "GYM";

export interface Room {
    id: string;
    schoolId: string;
    name: string;
    capacity: number;
    roomType: RoomType;
    createdAt: string;
    updatedAt: string;
}

export interface ListRoomsParams {
    page?: number;
    limit?: number;
}

export interface ListRoomsResponse {
    items: Room[];
    total: number;
    page: number;
    limit: number;
}

export interface CreateRoomRequest {
    name: string;
    capacity?: number;
    roomType: RoomType;
}

export interface UpdateRoomRequest {
    id: string;
    name?: string;
    capacity?: number;
    roomType?: RoomType;
}
