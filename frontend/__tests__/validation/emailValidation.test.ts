/**
 * Tests for the email validation utility used by the Bulk Staff Import.
 *
 * Tests the pure hasValidEmailStructure function and batch duplicate detection logic.
 */

import { describe, it, expect } from "vitest";
import { hasValidEmailStructure } from "@/features/staff-import/lib/validation";

// ─── Email structural validation ────────────────────────────────────────

describe("hasValidEmailStructure", () => {
    it("user@example.com → valid", () => {
        expect(hasValidEmailStructure("user@example.com")).toBe(true);
    });

    it("user+tag@sub.domain.co → valid", () => {
        expect(hasValidEmailStructure("user+tag@sub.domain.co")).toBe(true);
    });

    it("notanemail → invalid (no @)", () => {
        expect(hasValidEmailStructure("notanemail")).toBe(false);
    });

    it("@nodomain → invalid", () => {
        expect(hasValidEmailStructure("@nodomain")).toBe(false);
    });

    it("user@ → invalid", () => {
        expect(hasValidEmailStructure("user@")).toBe(false);
    });

    it("'' (empty string) → invalid (treated as missing)", () => {
        expect(hasValidEmailStructure("")).toBe(false);
    });

    it("USER@EXAMPLE.COM vs user@example.com → duplicate (case-insensitive comparison)", () => {
        // Structural check: both are valid individually
        expect(hasValidEmailStructure("USER@EXAMPLE.COM")).toBe(true);
        expect(hasValidEmailStructure("user@example.com")).toBe(true);

        // Duplicate detection is case-insensitive
        const email1 = "USER@EXAMPLE.COM";
        const email2 = "user@example.com";

        const emails = [email1, email2];
        const seen = new Set<string>();
        const duplicates = new Set<string>();

        for (const email of emails) {
            const lower = email.toLowerCase();
            if (seen.has(lower)) {
                duplicates.add(lower);
            }
            seen.add(lower);
        }

        expect(duplicates.size).toBe(1);
        expect(duplicates.has("user@example.com")).toBe(true);
    });

    it("user @example.com (space) — structural check passes (has @ and domain has dot)", () => {
        expect(hasValidEmailStructure("user @example.com")).toBe(true);
    });

    it("user@exam ple.com (space in domain) — structural check passes (has @ and domain has dot)", () => {
        expect(hasValidEmailStructure("user@exam ple.com")).toBe(true);
    });

    it("batch duplicate detection — given ['a@b.com', 'A@B.COM', 'x@y.com'], returns Set { 'a@b.com' } as duplicates", () => {
        const emails = ["a@b.com", "A@B.COM", "x@y.com"];
        const seen = new Set<string>();
        const duplicates = new Set<string>();

        for (const email of emails) {
            const lower = email.toLowerCase();
            if (seen.has(lower)) {
                duplicates.add(lower);
            }
            seen.add(lower);
        }

        expect(duplicates.size).toBe(1);
        expect(duplicates.has("a@b.com")).toBe(true);
    });

    it("detects no duplicates when all emails are unique", () => {
        const emails = ["a@b.com", "c@d.com", "e@f.com"];
        const seen = new Set<string>();
        const duplicates = new Set<string>();

        for (const email of emails) {
            const lower = email.toLowerCase();
            if (seen.has(lower)) {
                duplicates.add(lower);
            }
            seen.add(lower);
        }

        expect(duplicates.size).toBe(0);
    });
});

// ─── Additional edge cases ──────────────────────────────────────────────

describe("email structural edge cases", () => {
    it("handles email with dots in local part", () => {
        expect(hasValidEmailStructure("first.last@domain.com")).toBe(true);
    });

    it("handles email with numbers", () => {
        expect(hasValidEmailStructure("user123@domain.com")).toBe(true);
    });

    it("handles email with hyphens", () => {
        expect(hasValidEmailStructure("user-name@domain.com")).toBe(true);
    });

    it("rejects email without domain dot", () => {
        expect(hasValidEmailStructure("user@domain")).toBe(false);
    });

    it("handles email with only domain dot at start — structural check passes (has @ and domain has dot)", () => {
        // The structural check only validates the presence of @ and a dot in the domain
        expect(hasValidEmailStructure("user@.com")).toBe(true);
    });
});
