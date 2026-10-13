import { api } from "@/lib/api/client";
import type { SetupSummary } from "../types/setup-summary";

export async function getSetupSummary(): Promise<SetupSummary> {
    const res = await api.get<{ data: SetupSummary }>("/api/dashboard/setup-summary");
    return res.data;
}
