/**
 * Server-side auth utilities for Node.js runtime (pages, layouts, server actions).
 *
 * Unlike proxy.ts which runs in the Edge runtime and uses Web Crypto,
 * this module uses Node.js `crypto` for HMAC verification.
 */
import { cookies } from "next/headers";
import crypto from "node:crypto";
import { ROLE_COOKIE_NAME } from "@/lib/auth";

/** All valid roles in the system. Must match proxy.ts VALID_ROLES. */
const VALID_ROLES = new Set([
    "SYSTEM_ADMIN",
    "SCHOOL_ADMIN",
    "TEACHER",
    "NURSE",
    "FINANCE",
    "PARENT",
]);

export type UserRole = "SYSTEM_ADMIN" | "SCHOOL_ADMIN" | "TEACHER" | "NURSE" | "FINANCE" | "PARENT";

export interface AuthUser {
    role: UserRole;
}

/**
 * Reads the signed `somo_role` cookie and verifies its HMAC-SHA256 signature.
 *
 * Returns the verified role string on success, or null if:
 * - The cookie is missing
 * - COOKIE_SECRET is not set
 * - The signature is invalid/tampered
 * - The role is not in the known valid set
 */
export async function getVerifiedRole(): Promise<UserRole | null> {
    const cookieStore = await cookies();
    const roleCookie = cookieStore.get(ROLE_COOKIE_NAME);
    console.log("1:---------", roleCookie?.value);
    if (!roleCookie?.value) return null;

    const secret = process.env.COOKIE_SECRET;
    console.log("2:---------", secret);
    if (!secret) return null;

    const lastDot = roleCookie.value.lastIndexOf(".");
    console.log("3:---------", lastDot);
    if (lastDot === -1) return null;

    const value = roleCookie.value.slice(0, lastDot);
    const expectedSig = roleCookie.value.slice(lastDot + 1);
    console.log("4:---------", value, expectedSig);
    if (!value || !expectedSig) return null;

    try {
        const hmac = crypto.createHmac("sha256", secret);
        hmac.update(value);
        const computedSig = hmac.digest("hex");
        console.log("5:---------");
        // Constant-time comparison to prevent timing attacks
        if (!crypto.timingSafeEqual(Buffer.from(computedSig), Buffer.from(expectedSig))) {
            return null;
        }
        console.log("6:---------", value);
        if (!VALID_ROLES.has(value)) return null;

        return value as UserRole;
    } catch (err) {
        console.log("7:---------", err);
        return null;
    }
}

/**
 * Returns the verified user from the signed role cookie.
 * Redirects to /login if not authenticated or role is invalid.
 */
export async function getAuthUser(): Promise<AuthUser | null> {
    const role = await getVerifiedRole();
    if (!role) return null;
    return { role };
}
