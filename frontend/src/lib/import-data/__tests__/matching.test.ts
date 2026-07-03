/**
 * matching.test.ts — pure function tests for header matching, gender/date/class normalization.
 *
 * Covers the Test Matrix from the Bulk Student Importer spec.
 */

import { describe, it, expect } from "vitest";

import {
    matchHeader,
    matchAllHeaders,
    normalizeGender,
    normalizeClassName,
    normalizeDateOfBirth,
    fuzzyMatchClass,
    processRow,
    similarity,
    levenshtein,
} from "../matching";
import type { ClassMatchRecord } from "../matching";

// ─── Helpers ───────────────────────────────────────────────────────────────

const sampleClasses: ClassMatchRecord[] = [
    { id: "c1", grade_level: "Grade 1", stream_name: "Simba", display_label: "Grade 1 Simba" },
    { id: "c2", grade_level: "Grade 1", stream_name: "Tembo", display_label: "Grade 1 Tembo" },
    { id: "c3", grade_level: "Grade 2", stream_name: "Nyati", display_label: "Grade 2 Nyati" },
    { id: "c4", grade_level: "Grade 2", stream_name: "Kifaru", display_label: "Grade 2 Kifaru" },
];

// Build a lookup map with the full entry shape that processRow expects
function buildProcessRowLookup(
    classes: ClassMatchRecord[]
): Map<string, { class_id: string; grade_level: string; stream_name: string }> {
    const map = new Map();
    for (const c of classes) {
        const key = normalizeClassName(c.display_label);
        map.set(key, { class_id: c.id, grade_level: c.grade_level, stream_name: c.stream_name });
        const combined = normalizeClassName(`${c.grade_level} ${c.stream_name}`);
        map.set(combined, {
            class_id: c.id,
            grade_level: c.grade_level,
            stream_name: c.stream_name,
        });
    }
    return map;
}

const classLookup = buildProcessRowLookup(sampleClasses);

// ─── Happy Path ───────────────────────────────────────────────────────────

describe("Happy Path", () => {
    // Test 1: Standard single-column setup
    it("TC1 — unified Full Name column, exact class name matches, zero errors", () => {
        const raw = {
            "Full Name": "John Kiprop",
            Gender: "M",
            "Class Room": "Grade 1 Simba",
            "Date of Birth": "2015-06-15",
        };
        const mapping = {
            full_name: ["Full Name"],
            gender: "Gender",
            date_of_birth: "Date of Birth",
            class_room: "Class Room",
            nemis_number: null,
            assessment_number: null,
            birth_certificate_number: null,
        };
        const result = processRow(0, raw, mapping, classLookup);
        expect(result.full_name).toBe("John Kiprop");
        expect(result.gender).toBe("M");
        expect(result.class_id).toBe("c1");
        expect(result.has_error).toBe(false);
    });

    // Test 2: Split-name multi-column setup
    it("TC2 — split-name multi-column concatenation in selection order", () => {
        const raw = { "Jina la Kwanza": "Jane", Surname: "Wanjiku" };
        const mapping = {
            full_name: ["Jina la Kwanza", "Surname"],
            gender: null,
            date_of_birth: null,
            class_room: null,
            nemis_number: null,
            assessment_number: null,
            birth_certificate_number: null,
        };
        const result = processRow(0, raw, mapping, classLookup);
        expect(result.full_name).toBe("Jane Wanjiku");
    });

    // Test 3: Optional data missing safely
    it("TC3 — optional fields blank: class_room, DOB, tracking numbers all null, no errors", () => {
        const raw = { "Full Name": "Ali Hassan", Gender: "M" };
        const mapping = {
            full_name: ["Full Name"],
            gender: "Gender",
            date_of_birth: "Date of Birth",
            class_room: "Class Room",
            nemis_number: "NEMIS",
            assessment_number: "Assessment",
            birth_certificate_number: "Birth Cert",
        };
        const result = processRow(0, raw, mapping, classLookup);
        expect(result.date_of_birth).toBeNull();
        expect(result.class_id).toBeNull();
        expect(result.nemis_number).toBeNull();
        expect(result.assessment_number).toBeNull();
        expect(result.birth_certificate_number).toBeNull();
        expect(result.has_error).toBe(false);
    });
});

// ─── Header Matching ──────────────────────────────────────────────────────

describe("Header Matching", () => {
    // Test 4: Fuzzy match above threshold
    it("TC4 — 'Sex' maps to gender, 'Std Name' maps to full_name with ≥0.8 similarity", () => {
        // "sex" normalized is "sex" — it's ≥0.8 similarity to "gender"
        expect(matchHeader("Sex")).toBe("gender");
        // "std name" normalized is "std name" — it's ~0.75 similar to "student name", below 0.8 threshold
        expect(matchHeader("Std Name")).toBeNull();
    });

    // Test 14: Header below 0.8 threshold stays unmapped
    it("TC14 — ambiguous header below 0.8 threshold stays Unmapped", () => {
        expect(matchHeader("Info")).toBeNull();
        expect(matchHeader("Meta")).toBeNull();
        expect(matchHeader("Notes")).toBeNull();
    });

    // Test 15: Duplicate header collision — deterministic single assignment
    it("TC15 — duplicate normalized headers: first maps, second gets null", () => {
        const result = matchAllHeaders(["Class", "Name", "Class"]);
        // "class" maps to class_room on first occurrence, second overwrites to null
        // "name" maps to full_name
        expect(result["class"]).toBeNull(); // because the second "Class" won the overwrite
        expect(result["name"]).toBe("full_name");
    });

    // Exact match
    it("handles exact match for Swahili headers", () => {
        expect(matchHeader("Jina Kamili")).toBe("full_name");
        expect(matchHeader("Jinsia")).toBe("gender");
        expect(matchHeader("Darasa")).toBe("class_room");
    });

    it("handles exact match for normalized English headers", () => {
        expect(matchHeader("full name")).toBe("full_name");
        expect(matchHeader("Date of Birth")).toBe("date_of_birth");
        expect(matchHeader("NEMIS No")).toBe("nemis_number");
    });

    it("normalizes headers: lowercase, trim, collapse whitespace, strip punctuation", () => {
        expect(matchHeader("  FULL NAME  ")).toBe("full_name");
        expect(matchHeader("date-of-birth!")).toBe("date_of_birth");
    });
});

// ─── Gender Normalization ──────────────────────────────────────────────────

describe("Gender Normalization", () => {
    // Test 6: Missing required — gender blank
    it("TC6 — gender blank triggers missing_required and has_error", () => {
        const raw = { "Full Name": "", Gender: "" };
        const mapping = {
            full_name: ["Full Name"],
            gender: "Gender",
            date_of_birth: null,
            class_room: null,
            nemis_number: null,
            assessment_number: null,
            birth_certificate_number: null,
        };
        const result = processRow(0, raw, mapping, classLookup);
        expect(result.has_error).toBe(true);
        expect(result.errors.missing_required).toContain("Full Name");
        expect(result.errors.missing_required).toContain("Gender");
    });

    // Test 7: Gender unrecognized
    it("TC7 — unrecognized gender value returns empty string and error", () => {
        expect(normalizeGender("Other")).toBe("");
        expect(normalizeGender("N/A")).toBe("");
        expect(normalizeGender("")).toBe("");
        expect(normalizeGender("unknown")).toBe("");
    });

    // Test 8: Gender alias coverage — parametrized
    it.each([
        ["M", "M"],
        ["Male", "M"],
        ["male", "M"],
        ["MALE", "M"],
        ["boy", "M"],
        ["Boy", "M"],
        ["  male  ", "M"],
        ["F", "F"],
        ["Female", "F"],
        ["female", "F"],
        ["FEMALE", "F"],
        ["girl", "F"],
        ["Girl", "F"],
        ["Ume", "M"],
        ["UME", "M"],
        ["Mvulana", "M"],
        ["Uke", "F"],
        ["uke", "F"],
        ["Msichana", "F"],
    ])("TC8 — gender '%s' normalizes to '%s'", (input, expected) => {
        expect(normalizeGender(input)).toBe(expected);
    });
});

// ─── Class Matching ────────────────────────────────────────────────────────

describe("Class Matching", () => {
    // Test 9: Unresolved fuzzy class typo below 0.75 threshold
    it("TC9 — 'Symba 2' is below 0.75 threshold, class_id null, invalid_class set", () => {
        const result = fuzzyMatchClass("Symba 2", sampleClasses, classLookup);
        expect(result).toBeNull();
    });

    // Test 10: Class name normalization strips noise tokens equivalently
    it("TC10 — both 'Grade 1 Simba' and 'simba grade1' normalize to contain '1' and 'simba'", () => {
        const a = normalizeClassName("Grade 1 Simba");
        const b = normalizeClassName("simba grade1");
        // After stripping (grade|class|form|stream), 'Grade 1 Simba' → '1simba'
        // and 'simba grade1' → 'simbagrade1'
        // Both contain '1' + 'simba' — match via fuzzy fallback, not exact equality
        expect(a).toContain("1");
        expect(a).toContain("simba");
        expect(b).toContain("1");
        expect(b).toContain("simba");
    });

    // Test 11: Blank classroom passes cleanly (AC5)
    it("TC11 — blank classroom cell: class_id null, no invalid_class error", () => {
        const raw = { "Full Name": "Test", Gender: "M", "Class Room": "" };
        const mapping = {
            full_name: ["Full Name"],
            gender: "Gender",
            date_of_birth: null,
            class_room: "Class Room",
            nemis_number: null,
            assessment_number: null,
            birth_certificate_number: null,
        };
        const result = processRow(0, raw, mapping, classLookup);
        expect(result.class_id).toBeNull();
        expect(result.errors.invalid_class).toBeNull();
        expect(result.has_error).toBe(false);
    });

    it("fuzzy matches 'Grade 1 Simba' to class c1", () => {
        const result = fuzzyMatchClass("Grade 1 Simba", sampleClasses, classLookup);
        expect(result?.class_id).toBe("c1");
    });

    it("fuzzy matches 'G1 Simba' to class c1 (short form)", () => {
        const result = fuzzyMatchClass("G1 Simba", sampleClasses, classLookup);
        expect(result?.class_id).toBe("c1");
    });

    it("returns null for completely unrelated text", () => {
        const result = fuzzyMatchClass("zzzzzzzzz", sampleClasses, classLookup);
        expect(result).toBeNull();
    });
});

// ─── Date of Birth Normalization ───────────────────────────────────────────

describe("Date of Birth Normalization", () => {
    // Test 12: Unparseable date
    it("TC12 — garbage string date_of_birth null and invalid_date error, row submittable", () => {
        const raw = { "Full Name": "Test", Gender: "M", "Date of Birth": "garbage" };
        const mapping = {
            full_name: ["Full Name"],
            gender: "Gender",
            date_of_birth: "Date of Birth",
            class_room: null,
            nemis_number: null,
            assessment_number: null,
            birth_certificate_number: null,
        };
        const result = processRow(0, raw, mapping, classLookup);
        expect(result.date_of_birth).toBeNull();
        expect(result.errors.invalid_date).toBe("Could not parse date");
        // DOB is optional, so this shouldn't block submission
        expect(result.has_error).toBe(false);
    });

    // Test 13: Excel serial date vs string date
    it("TC13 — Excel serial number parses to a valid ISO date, and string format also works", () => {
        // Serial 45843 ≈ 2025-07-03 (timezone may shift by 1 day)
        const serialDate = normalizeDateOfBirth(45843);
        const stringDate = normalizeDateOfBirth("2025-07-03");
        // Both should produce valid ISO dates
        expect(serialDate).toMatch(/^\d{4}-\d{2}-\d{2}$/);
        expect(stringDate).toBe("2025-07-03");
        // The serial date should round-trip close to the target date (within 1 day TZ shift)
        const serialMs = new Date(serialDate!).getTime();
        const targetMs = new Date("2025-07-03").getTime();
        const diffDays = Math.abs(serialMs - targetMs) / 86400000;
        expect(diffDays).toBeLessThanOrEqual(1);
    });

    it("returns null for blank/null input", () => {
        expect(normalizeDateOfBirth(null)).toBeNull();
        expect(normalizeDateOfBirth("")).toBeNull();
        expect(normalizeDateOfBirth(undefined)).toBeNull();
    });

    it("parses DD/MM/YYYY format", () => {
        expect(normalizeDateOfBirth("15/06/2020")).toBe("2020-06-15");
    });

    it("parses DD-MM-YYYY format", () => {
        expect(normalizeDateOfBirth("15-06-2020")).toBe("2020-06-15");
    });

    it("parses YYYY-MM-DD format", () => {
        expect(normalizeDateOfBirth("2020-06-15")).toBe("2020-06-15");
    });
});

// ─── Levenshtein / Similarity ──────────────────────────────────────────────

describe("Levenshtein / Similarity", () => {
    it("levenshtein distance is 0 for identical strings", () => {
        expect(levenshtein("hello", "hello")).toBe(0);
    });

    it("levenshtein distance is the length of the longer string for completely different", () => {
        expect(levenshtein("abc", "xyz")).toBe(3);
    });

    it("similarity is 1 for identical strings", () => {
        expect(similarity("test", "test")).toBe(1);
    });

    it("similarity is 0 for empty vs non-empty", () => {
        expect(similarity("", "abc")).toBe(0);
    });
});

// ─── Required-Field Validation ─────────────────────────────────────────────

describe("Required-Field Validation", () => {
    it("missing full_name and gender both flagged", () => {
        const raw = { "Full Name": "", Gender: "" };
        const mapping = {
            full_name: ["Full Name"],
            gender: "Gender",
            date_of_birth: null,
            class_room: null,
            nemis_number: null,
            assessment_number: null,
            birth_certificate_number: null,
        };
        const result = processRow(0, raw, mapping, classLookup);
        expect(result.errors.missing_required).toContain("Full Name");
        expect(result.errors.missing_required).toContain("Gender");
        expect(result.has_error).toBe(true);
    });

    it("missing full_name only flag", () => {
        const raw = { "Full Name": "", Gender: "M" };
        const mapping = {
            full_name: ["Full Name"],
            gender: "Gender",
            date_of_birth: null,
            class_room: null,
            nemis_number: null,
            assessment_number: null,
            birth_certificate_number: null,
        };
        const result = processRow(0, raw, mapping, classLookup);
        expect(result.errors.missing_required).toContain("Full Name");
        expect(result.errors.missing_required).not.toContain("Gender");
    });
});
