import type { components } from "@/lib/generated/api";

/**
 * Generated wire-format types (snake_case) from the OpenAPI contract.
 * Do not edit by hand — regenerated from backend/docs/swagger.json.
 */
export type EducationSystemListItemResponse =
  components["schemas"]["service.EducationSystemListItemResponse"];
export type EducationSystemListResponse =
  components["schemas"]["service.EducationSystemListResponse"];

/**
 * Frontend-facing education system type (camelCase) used by forms and components.
 */
export interface EducationSystem {
  id: string;
  name: string;
  countryId: string;
  countryName: string;
  countryCode: string;
}
