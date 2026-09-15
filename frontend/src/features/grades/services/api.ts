import { api } from "@/lib/api/client";
import type { GradesResponse } from "../types/grade";

export async function listGrades(): Promise<GradesResponse> {
    return api.get<GradesResponse>("/api/school/grades");
}
