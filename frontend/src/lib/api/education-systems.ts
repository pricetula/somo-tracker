/**
 * Education System API functions.
 *
 * Endpoints:
 *   GET /education-systems  — list all available education systems
 */

import { api } from "./client";

/** A single education system (curriculum framework). */
export interface EducationSystem {
  id: string;
  name: string;
  country_code: string;
}

/** Fetch all education systems. */
export async function listEducationSystems(): Promise<EducationSystem[]> {
  return api.get<EducationSystem[]>("/education-systems");
}
