import { apiFetch } from '@/lib/api/client';
import type { DeactivateSchoolsRequest, DeactivateSchoolsResponse } from '../types';
import type { components } from '@/features/auth/api/generated/types';

type WireDeactivateSchoolsRequest = components['schemas']['DeactivateSchoolsRequest'];

function mapDeactivateSchoolsRequestToWire(
  req: DeactivateSchoolsRequest
): WireDeactivateSchoolsRequest {
  return {
    ids: req.ids,
  };
}

export async function deactivateSchools(req: DeactivateSchoolsRequest): Promise<DeactivateSchoolsResponse> {
  const wireReq = mapDeactivateSchoolsRequestToWire(req);
  const wireRes = await apiFetch<{ success?: boolean }>('/api/v1/schools', {
    method: 'DELETE',
    body: JSON.stringify(wireReq),
  });

  return {
    success: wireRes.success ?? false,
  };
}
