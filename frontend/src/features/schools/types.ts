import type { components } from '@/features/auth/api/generated/types';

// Internal camelCase types (mapped from snake_case wire types at API layer)
export interface School {
  id: string;
  tenantId: string;
  name: string;
  code: string;
  countryId: string;
  educationSystemId: string;
  createdAt: string;
  updatedAt: string;
  deletedAt: string | null;
}

export interface SchoolCreateRequest {
  name: string;
  code: string;
  countryId: string;
  educationSystemId: string;
}

export interface SchoolUpdateRequest {
  name?: string;
  code?: string;
  countryId?: string;
  educationSystemId?: string;
}

export interface DeactivateSchoolsRequest {
  ids: string[];
}

export interface DeactivateSchoolsResponse {
  success: boolean;
}

// Map from generated snake_case wire type to internal camelCase type
export function mapSchoolFromWire(
  school: components['schemas']['School']
): School {
  return {
    id: school.id!,
    tenantId: school.tenant_id!,
    name: school.name!,
    code: school.code!,
    countryId: school.country_id!,
    educationSystemId: school.education_system_id!,
    createdAt: school.created_at!,
    updatedAt: school.updated_at!,
    deletedAt: school.deleted_at ?? null,
  };
}
