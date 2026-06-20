/**
 * Tenant API functions.
 *
 * Endpoints:
 *   POST /tenants  — create a new tenant (admin use)
 */

import { api } from "./client";

/** Request body for POST /tenants */
export interface CreateTenantPayload {
    name: string;
    slug?: string;
}

/** Response from POST /tenants */
export interface TenantResponse {
    id: string;
    name: string;
    slug: string;
    created_at: string;
}

/** Create a new tenant. Requires admin privileges. */
export async function createTenant(payload: CreateTenantPayload): Promise<TenantResponse> {
    return api.post<TenantResponse>("/tenants", payload);
}
