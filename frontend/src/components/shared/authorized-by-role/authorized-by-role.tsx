/**
 * AuthorizedByRole — DEPRECATED: Backend no longer provides role cookie or /me endpoint.
 *
 * This component previously verified the signed `somo_role` cookie server-side.
 * The current backend auth flow uses only the HttpOnly `session_token` cookie with
 * fingerprint validation. Role information is not available client-side without
 * a /me endpoint.
 *
 * TODO: Re-implement when backend adds /me endpoint with role/tenant info.
 * For now, this component passes through children unconditionally.
 * Actual authorization is enforced by the backend session middleware on API calls.
 */

interface AuthorizedByRoleProps {
    children: React.ReactNode;
    allowedRoles?: string[];
}

export async function AuthorizedByRole({
    children,
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    allowedRoles,
}: AuthorizedByRoleProps) {
    // Role verification not available without backend /me endpoint.
    // Backend enforces authorization on all API endpoints via session middleware.
    return children;
}
