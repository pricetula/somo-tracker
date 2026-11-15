export type { School, SchoolCreateRequest, SchoolUpdateRequest } from './types';
export { getSchools, type GetSchoolsParams } from './api/get-schools';
export { createSchool } from './api/create-school';
export { getSchool } from './api/get-school';
export { updateSchool } from './api/update-school';
export { deactivateSchools } from './api/deactivate-schools';
export { useSchools } from './hooks/use-schools';
export { schoolKeys } from './query-keys';
