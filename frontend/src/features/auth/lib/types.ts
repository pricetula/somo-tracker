/**
 * Type matching the Go backend ``MeResult`` returned by ``GET /api/me``.
 * See backend/internal/services/me_service.go
 */
export interface MeResult {
    user_name: string;
    email: string;
    active_school_id: string | null;
    school_name: string | null;
    active_school_role: string | null;
    tenant_id: string;
}
