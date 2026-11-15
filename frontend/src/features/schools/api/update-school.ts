import { apiFetch } from '@/lib/api/client';
import type { School, SchoolUpdateRequest } from '../types';
import type { components } from '@/features/auth/api/generated/types';

type WireSchool = components['schemas']['School'];
type WireSchoolUpdateRequest = components['schemas']['SchoolUpdateRequest'];

function mapSchoolUpdateRequestToWire(req: SchoolUpdateRequest): WireSchoolUpdateRequest {
  const wireReq: WireSchoolUpdateRequest = {};
  if (req.name !== undefined) wireReq.name = req.name;
  if (req.code !== undefined) wireReq.code = req.code;
  if (req.countryId !== undefined) wireReq.country_id = req.countryId;
  if (req.educationSystemId !== undefined) wireReq.education_system_id = req.educationSystemId;
  return wireReq;
}

function mapSchoolFromWire(school: WireSchool): School {
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

export async function updateSchool(id: string, req: SchoolUpdateRequest): Promise<School> {
  const wireReq = mapSchoolUpdateRequestToWire(req);
  const wireSchool = await apiFetch<WireSchool>(`/api/v1/schools/${id}`, {
    method: 'PATCH',
    body: JSON.stringify(wireReq),
  });

  return mapSchoolFromWire(wireSchool);
}
