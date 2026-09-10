import { api } from "./client";
import type {
    MagicLinkRequest,
    MagicLinkResponse,
    CallbackResponse,
    LogoutResponse,
} from "./generated";

export interface MeResult {
    user_name: string;
    email: string;
    active_school_id: string | null;
    school_name: string | null;
    active_school_role: string | null;
    tenant_id: string;
}

export type {
    MagicLinkRequest,
    MagicLinkResponse,
    CallbackResponse,
    LogoutResponse,
} from "./generated";

export async function sendMagicLink(email: string, orgId?: string): Promise<MagicLinkResponse> {
    const body: MagicLinkRequest = { email };
    if (orgId) body.org_id = orgId;
    return api.post<MagicLinkResponse>("/api/auth/magic-link/send", body);
}

export async function handleCallback(token: string): Promise<CallbackResponse> {
    return api.get<CallbackResponse>(`/api/auth/callback?token=${encodeURIComponent(token)}`);
}

export async function getMe(): Promise<MeResult> {
    return api.get<MeResult>("/api/me");
}

export async function logout(): Promise<LogoutResponse> {
    return api.post<LogoutResponse>("/api/auth/logout");
}
