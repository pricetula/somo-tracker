import type { components } from "@/lib/generated/api";

/**
 * Generated wire-format types (snake_case) from the OpenAPI contract.
 * Do not edit by hand — regenerated from backend/docs/swagger.json.
 */
export type CountryResponse = components["schemas"]["service.CountryResponse"];
export type CountryListResponse = components["schemas"]["service.CountryListResponse"];

/**
 * Frontend-facing country type (camelCase) used by forms and components.
 */
export interface Country {
  id: string;
  name: string;
  code: string;
}
