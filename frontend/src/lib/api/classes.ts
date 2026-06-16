/**
 * Classes API client.
 *
 * Endpoints:
 *   GET  /api/v1/schools/classes    → Class[]
 *   POST /api/v1/schools/classes/generate → GenerateResult
 */

import { api } from "@/lib/api/client";
import type { ClassItem, GeneratePayload, GenerateResult } from "@/features/classes/types";

/** Fetch all active classes for the current school and academic year. */
export async function fetchClasses(): Promise<ClassItem[]> {
  try {
    return await api.get<ClassItem[]>("/api/v1/schools/classes");
  } catch (err) {
    // On 404, return empty list (no classes configured)
    if (
      err &&
      typeof err === "object" &&
      "status" in err &&
      (err as { status: number }).status === 404
    ) {
      return [];
    }
    throw err;
  }
}

/** Generate (bulk-create) classrooms from stream names × grade levels. */
export async function generateClasses(
  payload: GeneratePayload,
): Promise<GenerateResult> {
  return await api.post<GenerateResult>(
    "/api/v1/schools/classes/generate",
    payload,
  );
}
