/**
 * Tests for the phone normalization utility.
 *
 * Uses the real libphonenumber-js module (not mocked) for proper
 * E.164 normalization testing with the KE default country.
 */

import { describe, it, expect } from "vitest";
import { normalizePhone } from "@/features/staff-import/lib/validation";

// ─── Phone normalization ────────────────────────────────────────────────

describe("normalizePhone", () => {
    it("'0712345678' with country 'KE' → '+254712345678' (E.164)", () => {
        const result = normalizePhone("0712345678");
        expect(result).toBe("+254712345678");
    });

    it("'+254712345678' (already E.164) → '+254712345678' (unchanged)", () => {
        const result = normalizePhone("+254712345678");
        expect(result).toBe("+254712345678");
    });

    it("'+1 (415) 555-0172' (US number, explicit country code) → '+14155550172'", () => {
        const result = normalizePhone("+1 (415) 555-0172");
        expect(result).toBe("+14155550172");
    });

    it("'N/A' → null + wasCleared: true", () => {
        // normalizePhone returns null for unparseable. The "wasCleared" semantic
        // is handled by the calling component: if input was non-empty and normalizePhone
        // returned null, the component sets phone to "" and marks it as a warning.
        const result = normalizePhone("N/A");
        expect(result).toBeNull();
    });

    it("'abc' → null (unparseable, treated as cleared)", () => {
        const result = normalizePhone("abc");
        expect(result).toBeNull();
    });

    it("'' (empty) → null, no warning (intentionally blank)", () => {
        const result = normalizePhone("");
        expect(result).toBeNull();
    });

    it("'   ' (whitespace) → null, no warning", () => {
        const result = normalizePhone("   ");
        expect(result).toBeNull();
    });

    it("'0712345678 ext 101' → libphonenumber parses it as a KE number (drops ext)", () => {
        // libphonenumber may strip the extension and parse the base number
        const result = normalizePhone("0712345678 ext 101");
        // It is treated as a valid KE number with leading 0
        expect(result).toBe("+254712345678");
    });

    it("'712345678' (missing leading 0, KE) → '+254712345678'", () => {
        // libphonenumber handles 712345678 as a valid KE number
        const result = normalizePhone("712345678");
        expect(result).toBe("+254712345678");
    });
});

// ─── Edge cases ─────────────────────────────────────────────────────────

describe("normalizePhone edge cases", () => {
    it("handles KE landline format — Nairobi area code", () => {
        const result = normalizePhone("020 1234567");
        // libphonenumber recognizes this as a valid KE landline
        expect(result).toBe("+254201234567");
    });

    it("handles phone with leading plus and spaces", () => {
        const result = normalizePhone("+254 712 345 678");
        expect(result).toBe("+254712345678");
    });

    it("handles phone with dashes", () => {
        const result = normalizePhone("+254-712-345-678");
        expect(result).toBe("+254712345678");
    });

    it("returns null for completely invalid string", () => {
        const result = normalizePhone("not-a-phone-number-at-all");
        expect(result).toBeNull();
    });
});

// ─── wasCleared flag (integration-style) ────────────────────────────────

describe("normalizePhone — wasCleared semantic", () => {
    /**
     * The wasCleared flag is computed by the calling UI component:
     *   if input was non-empty AND normalizePhone returns null → wasCleared = true
     *   if input was empty/whitespace → wasCleared = false (intentionally blank)
     */
    it("non-empty unparseable input implies wasCleared: true", () => {
        const input = "N/A";
        const result = normalizePhone(input);
        expect(result).toBeNull();
        // Component logic: if (input.trim() && !result) -> wasCleared = true
        const wasCleared = input.trim().length > 0 && result === null;
        expect(wasCleared).toBe(true);
    });

    it("empty whitespace input implies wasCleared: false", () => {
        const input = "   ";
        const result = normalizePhone(input);
        expect(result).toBeNull();
        // Component logic: if (!input.trim()) -> wasCleared = false
        const wasCleared = input.trim().length > 0 && result === null;
        expect(wasCleared).toBe(false);
    });
});
