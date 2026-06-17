/**
 * Tenant API functions.
 *
 * Endpoints:
 *   POST /tenants  — create a new tenant (admin use)
 *
 * 🔄 AUTO-GENERATED TYPES: See src/lib/api/generated.ts (generated from backend swagger.json).
 *   Run `pnpm generate:api` to regenerate when the backend API changes.
 */

import { api } from "./client";
import type { definitions } from "./generated";

/** Request body for POST /tenants */
export type CreateTenantPayload = definitions["internal_tenant.CreateTenantPayload"];

/** Response from POST /tenants */
export type TenantResponse = definitions["internal_tenant.Tenant"];

/** Create a new tenant. Requires admin privileges. */
export async function createTenant(payload: CreateTenantPayload): Promise<TenantResponse> {
    return api.post<TenantResponse>("/tenants", payload);
}
