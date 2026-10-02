import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { listRooms, getRoom, createRoom, updateRoom, deleteRoom } from "../services/api";
import type { ListRoomsParams, CreateRoomRequest, UpdateRoomRequest } from "../types/room";
import { getErrorMessage } from "@/lib/errors";
import { toast } from "sonner";

export const roomKeys = {
    list: (params: ListRoomsParams) => ["rooms", "list", params] as const,
    detail: (id: string) => ["rooms", "detail", id] as const,
};

export function useRooms(params: ListRoomsParams = {}) {
    return useQuery({
        queryKey: roomKeys.list(params),
        queryFn: () => listRooms(params),
        staleTime: 30_000,
    });
}

export function useRoom(id: string) {
    return useQuery({
        queryKey: roomKeys.detail(id),
        queryFn: () => getRoom(id),
        enabled: !!id,
    });
}

export function useCreateRoom() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (data: CreateRoomRequest) => createRoom(data),
        onSuccess: () => {
            toast.success("Room created");
            queryClient.invalidateQueries({ queryKey: ["rooms", "list"] });
        },
        onError: (err) => toast.error(getErrorMessage(err)),
    });
}

export function useUpdateRoom() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (data: UpdateRoomRequest) => updateRoom(data),
        onSuccess: (_data, variables) => {
            toast.success("Room updated");
            queryClient.invalidateQueries({ queryKey: ["rooms", "list"] });
            queryClient.invalidateQueries({ queryKey: roomKeys.detail(variables.id) });
        },
        onError: (err) => toast.error(getErrorMessage(err)),
    });
}

export function useDeleteRoom() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (id: string) => deleteRoom(id),
        onSuccess: () => {
            toast.success("Room deleted");
            queryClient.invalidateQueries({ queryKey: ["rooms", "list"] });
        },
        onError: (err) => toast.error(getErrorMessage(err)),
    });
}
