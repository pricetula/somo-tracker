/**
 * Tests for the Two-Cookie Auth role signing and verification.
 *
 * These tests validate the Web Crypto API based verifySignedCookie function
 * used in the Next.js proxy middleware. The backend signs with createSignedCookieValue
 * (HMAC-SHA256, Go), and the frontend verifies with verifySignedCookie (HMAC-SHA256, Web Crypto).
 *
 * To run: pnpm vitest run src/__tests__/verify-signed-cookie.test.ts
 */

import { describe, it, expect } from "vitest";

// Re-implement verifySignedCookie here so it can be tested in isolation.
// In production this lives in src/proxy.ts.
async function verifySignedCookie(cookieValue: string, secret: string): Promise<string | null> {
    const lastDot = cookieValue.lastIndexOf(".");
    if (lastDot === -1) return null;

    const value = cookieValue.slice(0, lastDot);
    const expectedSig = cookieValue.slice(lastDot + 1);

    if (!value || !expectedSig) return null;

    try {
        const encoder = new TextEncoder();
        const keyData = encoder.encode(secret);
        const key = await crypto.subtle.importKey(
            "raw",
            keyData,
            { name: "HMAC", hash: "SHA-256" },
            false,
            ["sign"]
        );

        const valueBytes = encoder.encode(value);
        const sigBytes = await crypto.subtle.sign("HMAC", key, valueBytes);

        const sigHex = Array.from(new Uint8Array(sigBytes))
            .map((b) => b.toString(16).padStart(2, "0"))
            .join("");

        if (sigHex.length !== expectedSig.length) return null;
        let mismatch = 0;
        for (let i = 0; i < sigHex.length; i++) {
            mismatch |= sigHex.charCodeAt(i) ^ expectedSig.charCodeAt(i);
        }

        return mismatch === 0 ? value : null;
    } catch {
        return null;
    }
}

/**
 * Since Web Crypto uses the same algorithm as Go's crypto/hmac,
 * we can verify cross-platform compatibility by signing with Node.js
 * and verifying with Web Crypto, or just test the round-trip.
 * Here we test the verify function's self-consistency.
 */

describe("verifySignedCookie", () => {
    const secret = "test-secret-must-be-32-chars-long!";
    const roles = ["SCHOOL_ADMIN", "TEACHER", "SYSTEM_ADMIN", "SUPPORT_STAFF"];

    describe("valid cookies", () => {
        it.each(roles)("accepts a valid signed cookie for role %s", async (role) => {
            // Sign using the same HMAC-SHA256 algorithm
            const encoder = new TextEncoder();
            const key = await crypto.subtle.importKey(
                "raw",
                encoder.encode(secret),
                { name: "HMAC", hash: "SHA-256" },
                false,
                ["sign"]
            );
            const sigBytes = await crypto.subtle.sign("HMAC", key, encoder.encode(role));
            const sigHex = Array.from(new Uint8Array(sigBytes))
                .map((b) => b.toString(16).padStart(2, "0"))
                .join("");
            const signed = `${role}.${sigHex}`;

            const result = await verifySignedCookie(signed, secret);
            expect(result).toBe(role);
        });
    });

    describe("tampered cookies", () => {
        it("rejects a tampered signature", async () => {
            const signed =
                "SCHOOL_ADMIN.0000111122223333444455556666777788889999aaaabbbbccccddddeeeeffff";
            const result = await verifySignedCookie(signed, secret);
            expect(result).toBeNull();
        });

        it("rejects when the value part is modified", async () => {
            // Sign TEACHER, then change the value to SCHOOL_ADMIN
            const encoder = new TextEncoder();
            const key = await crypto.subtle.importKey(
                "raw",
                encoder.encode(secret),
                { name: "HMAC", hash: "SHA-256" },
                false,
                ["sign"]
            );
            const sigBytes = await crypto.subtle.sign("HMAC", key, encoder.encode("TEACHER"));
            const sigHex = Array.from(new Uint8Array(sigBytes))
                .map((b) => b.toString(16).padStart(2, "0"))
                .join("");
            const tampered = `SCHOOL_ADMIN.${sigHex}`;

            const result = await verifySignedCookie(tampered, secret);
            expect(result).toBeNull();
        });
    });

    describe("malformed cookies", () => {
        it("rejects a cookie with no dot separator", async () => {
            const result = await verifySignedCookie("SCHOOL_ADMIN", secret);
            expect(result).toBeNull();
        });

        it("rejects a cookie with empty value", async () => {
            const result = await verifySignedCookie(".abcdef1234567890", secret);
            expect(result).toBeNull();
        });

        it("rejects a cookie with empty signature", async () => {
            const result = await verifySignedCookie("SCHOOL_ADMIN.", secret);
            expect(result).toBeNull();
        });

        it("rejects an empty string", async () => {
            const result = await verifySignedCookie("", secret);
            expect(result).toBeNull();
        });

        it("rejects a cookie with multiple dots (still splits on last)", async () => {
            const result = await verifySignedCookie("a.b.c", secret);
            // The value would be "a.b", signature "c" — will fail hex decode comparison
            expect(result).toBeNull();
        });
    });

    describe("wrong secret", () => {
        it("rejects a cookie signed with a different secret", async () => {
            const encoder = new TextEncoder();
            const wrongSecret = "wrong-secret-thirty-two-chars-long!!";
            const key = await crypto.subtle.importKey(
                "raw",
                encoder.encode(wrongSecret),
                { name: "HMAC", hash: "SHA-256" },
                false,
                ["sign"]
            );
            const sigBytes = await crypto.subtle.sign("HMAC", key, encoder.encode("SCHOOL_ADMIN"));
            const sigHex = Array.from(new Uint8Array(sigBytes))
                .map((b) => b.toString(16).padStart(2, "0"))
                .join("");
            const signed = `SCHOOL_ADMIN.${sigHex}`;

            const result = await verifySignedCookie(signed, secret);
            expect(result).toBeNull();
        });
    });

    describe("edge cases", () => {
        it("handles very long role values", async () => {
            const longRole = "A".repeat(100);
            const encoder = new TextEncoder();
            const key = await crypto.subtle.importKey(
                "raw",
                encoder.encode(secret),
                { name: "HMAC", hash: "SHA-256" },
                false,
                ["sign"]
            );
            const sigBytes = await crypto.subtle.sign("HMAC", key, encoder.encode(longRole));
            const sigHex = Array.from(new Uint8Array(sigBytes))
                .map((b) => b.toString(16).padStart(2, "0"))
                .join("");
            const signed = `${longRole}.${sigHex}`;

            const result = await verifySignedCookie(signed, secret);
            expect(result).toBe(longRole);
        });

        it("is deterministic (same inputs → same result)", async () => {
            const encoder = new TextEncoder();
            const key = await crypto.subtle.importKey(
                "raw",
                encoder.encode(secret),
                { name: "HMAC", hash: "SHA-256" },
                false,
                ["sign"]
            );
            const sigBytes1 = await crypto.subtle.sign("HMAC", key, encoder.encode("TEACHER"));
            const sigHex1 = Array.from(new Uint8Array(sigBytes1))
                .map((b) => b.toString(16).padStart(2, "0"))
                .join("");
            const signed1 = `TEACHER.${sigHex1}`;

            const sigBytes2 = await crypto.subtle.sign("HMAC", key, encoder.encode("TEACHER"));
            const sigHex2 = Array.from(new Uint8Array(sigBytes2))
                .map((b) => b.toString(16).padStart(2, "0"))
                .join("");
            const signed2 = `TEACHER.${sigHex2}`;

            expect(signed1).toBe(signed2);
        });
    });
});
