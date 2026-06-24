/**
 * Comprehensive tests for all student import validation utilities.
 *
 * Covers: gender normalization, date parsing, UPI/KNEC validation,
 * class name normalization, parent name normalization, duplicate detection,
 * full record validation, and per-field validation.
 *
 * To run: pnpm vitest run src/features/student-import/__tests__/validation.test.ts
 */

import { describe, it, expect } from "vitest";
import {
    normalizeGender,
    parseDate,
    validateUPI,
    validateKNEC,
    normalizeClassName,
    normalizeParentName,
    detectDuplicates,
    validateRecord,
    validateField,
} from "../lib/validation";
import type { StagedStudentRecord, ParentRecord, ClassRecord, ExistingStudent } from "../types";

// =========================================================================
// GENDER NORMALIZATION
// =========================================================================

describe("normalizeGender", () => {
    describe("valid inputs", () => {
        it('returns "M" for "m"', () => {
            expect(normalizeGender("m")).toEqual({ gender: "M" });
        });

        it('returns "M" for "male"', () => {
            expect(normalizeGender("male")).toEqual({ gender: "M" });
        });

        it('returns "M" for "boy"', () => {
            expect(normalizeGender("boy")).toEqual({ gender: "M" });
        });

        it('returns "M" for "b"', () => {
            expect(normalizeGender("b")).toEqual({ gender: "M" });
        });

        it('returns "M" for "1"', () => {
            expect(normalizeGender("1")).toEqual({ gender: "M" });
        });

        it('returns "M" for Swahili "me" (mwanaume)', () => {
            expect(normalizeGender("me")).toEqual({ gender: "M" });
        });

        it('returns "M" for uppercase "MALE"', () => {
            expect(normalizeGender("MALE")).toEqual({ gender: "M" });
        });

        it('returns "M" for mixed case "BoY"', () => {
            expect(normalizeGender("BoY")).toEqual({ gender: "M" });
        });

        it('returns "F" for "f"', () => {
            expect(normalizeGender("f")).toEqual({ gender: "F" });
        });

        it('returns "F" for "female"', () => {
            expect(normalizeGender("female")).toEqual({ gender: "F" });
        });

        it('returns "F" for "girl"', () => {
            expect(normalizeGender("girl")).toEqual({ gender: "F" });
        });

        it('returns "F" for "g"', () => {
            expect(normalizeGender("g")).toEqual({ gender: "F" });
        });

        it('returns "F" for "2"', () => {
            expect(normalizeGender("2")).toEqual({ gender: "F" });
        });

        it('returns "F" for Swahili "ke" (kike)', () => {
            expect(normalizeGender("ke")).toEqual({ gender: "F" });
        });

        it('returns "F" for uppercase "GIRL"', () => {
            expect(normalizeGender("GIRL")).toEqual({ gender: "F" });
        });

        it("trims whitespace around input", () => {
            expect(normalizeGender("  male  ")).toEqual({ gender: "M" });
        });
    });

    describe("invalid inputs", () => {
        it("returns null + error for unrecognized value", () => {
            const result = normalizeGender("unknown");
            expect(result.gender).toBeNull();
            expect(result.error).toContain("Unrecognized gender value");
            expect(result.error).toContain("unknown");
        });

        it("returns null + error for empty string", () => {
            const result = normalizeGender("");
            expect(result.gender).toBeNull();
            expect(result.error).toBe("Gender is required");
        });

        it("returns null + error for whitespace-only string", () => {
            const result = normalizeGender("   ");
            expect(result.gender).toBeNull();
            expect(result.error).toBe("Gender is required");
        });

        it("returns null + error for null", () => {
            const result = normalizeGender(null);
            expect(result.gender).toBeNull();
            expect(result.error).toBe("Gender is required");
        });

        it("returns null + error for undefined", () => {
            const result = normalizeGender(undefined);
            expect(result.gender).toBeNull();
            expect(result.error).toBe("Gender is required");
        });
    });
});

// =========================================================================
// DATE PARSING
// =========================================================================

describe("parseDate", () => {
    describe("ISO format (YYYY-MM-DD)", () => {
        it("returns the date as-is for valid ISO", () => {
            const result = parseDate("2010-03-15");
            expect(result.date).toBe("2010-03-15");
            expect(result.error).toBeUndefined();
            expect(result.advisory).toBeUndefined();
        });
    });

    describe("DD/MM/YYYY format", () => {
        it("parses DD/MM/YYYY correctly", () => {
            const result = parseDate("15/03/2010");
            expect(result.date).toBe("2010-03-15");
            expect(result.error).toBeUndefined();
            expect(result.advisory).toBeUndefined();
        });

        it("parses D/M/YYYY (single digit) correctly", () => {
            const result = parseDate("5/3/2010");
            expect(result.date).toBe("2010-03-05");
        });

        it("adds advisory for ambiguous dates (DD <= 12 and MM <= 12)", () => {
            const result = parseDate("01/02/2010");
            expect(result.date).toBe("2010-02-01");
            expect(result.advisory).toContain("Ambiguous date");
            expect(result.advisory).toContain("DD/MM/YYYY");
        });

        it("does NOT add advisory when DD > 12 (unambiguous)", () => {
            const result = parseDate("15/02/2010");
            expect(result.date).toBe("2010-02-15");
            expect(result.advisory).toBeUndefined();
        });
    });

    describe("DD-MM-YYYY format", () => {
        it("parses DD-MM-YYYY correctly", () => {
            const result = parseDate("15-03-2010");
            expect(result.date).toBe("2010-03-15");
        });

        it("adds advisory for ambiguous dash-separated dates", () => {
            const result = parseDate("01-02-2010");
            expect(result.date).toBe("2010-02-01");
            expect(result.advisory).toContain("Ambiguous date");
        });
    });

    describe("error cases", () => {
        it("returns error for unrecognized format", () => {
            const result = parseDate("not-a-date");
            expect(result.date).toBeNull();
            expect(result.error).toContain("Unrecognized date format");
        });

        it("returns error for invalid month (>12)", () => {
            const result = parseDate("15/13/2010");
            expect(result.date).toBeNull();
            expect(result.error).toContain("Unrecognized date format");
        });

        it("returns error for invalid day (>31)", () => {
            const result = parseDate("32/03/2010");
            expect(result.date).toBeNull();
            expect(result.error).toContain("Unrecognized date format");
        });

        it("returns null for empty input", () => {
            const result = parseDate("");
            expect(result.date).toBeNull();
            expect(result.error).toBeUndefined();
        });

        it("returns null for null input", () => {
            const result = parseDate(null);
            expect(result.date).toBeNull();
            expect(result.error).toBeUndefined();
        });

        it("returns null for undefined input", () => {
            const result = parseDate(undefined);
            expect(result.date).toBeNull();
            expect(result.error).toBeUndefined();
        });

        it("rejects year outside 1900-2100", () => {
            const result = parseDate("15/03/1899");
            expect(result.date).toBeNull();
        });
    });
});

// =========================================================================
// UPI VALIDATION (Kenya NEMIS)
// =========================================================================

describe("validateUPI", () => {
    it("accepts valid UPI: KP1234567A", () => {
        const result = validateUPI("KP1234567A");
        expect(result.upi).toBe("KP1234567A");
        expect(result.error).toBeUndefined();
    });

    it("accepts valid UPI with lowercase letters, uppercases them", () => {
        const result = validateUPI("kp1234567a");
        expect(result.upi).toBe("KP1234567A");
    });

    it("trims whitespace", () => {
        const result = validateUPI("  KP1234567A  ");
        expect(result.upi).toBe("KP1234567A");
    });

    it("rejects UPI without KP prefix", () => {
        const result = validateUPI("12345678A");
        expect(result.upi).toBeNull();
        expect(result.error).toContain("Invalid UPI format");
    });

    it("rejects UPI with wrong digit count (6 digits)", () => {
        const result = validateUPI("KP123456A");
        expect(result.upi).toBeNull();
        expect(result.error).toContain("Invalid UPI format");
    });

    it("rejects UPI with wrong digit count (8 digits)", () => {
        const result = validateUPI("KP12345678A");
        expect(result.upi).toBeNull();
    });

    it("rejects UPI with special characters", () => {
        const result = validateUPI("KP1234567@");
        expect(result.upi).toBeNull();
    });

    it("returns null for empty input", () => {
        const result = validateUPI("");
        expect(result.upi).toBeNull();
        expect(result.error).toBeUndefined();
    });

    it("returns null for null input", () => {
        const result = validateUPI(null);
        expect(result.upi).toBeNull();
    });
});

// =========================================================================
// KNEC ASSESSMENT NUMBER VALIDATION
// =========================================================================

describe("validateKNEC", () => {
    it("accepts valid 8-character alphanumeric", () => {
        const result = validateKNEC("ABC12345");
        expect(result.knec).toBe("ABC12345");
    });

    it("accepts valid number with hyphens", () => {
        const result = validateKNEC("ABC-1234");
        expect(result.knec).toBe("ABC-1234");
    });

    it("uppercases lowercase input", () => {
        const result = validateKNEC("abc12345");
        expect(result.knec).toBe("ABC12345");
    });

    it("accepts minimum length (6)", () => {
        const result = validateKNEC("AB1234");
        expect(result.knec).toBe("AB1234");
    });

    it("accepts maximum length (14)", () => {
        const result = validateKNEC("ABCDEF12345678");
        expect(result.knec).toBe("ABCDEF12345678");
    });

    it("rejects too short (< 6)", () => {
        const result = validateKNEC("AB12");
        expect(result.knec).toBeNull();
        expect(result.error).toContain("Invalid KNEC assessment number format");
    });

    it("rejects too long (> 14)", () => {
        const result = validateKNEC("ABCDEF1234567890");
        expect(result.knec).toBeNull();
    });

    it("rejects special characters (not hyphen)", () => {
        const result = validateKNEC("ABC@12345");
        expect(result.knec).toBeNull();
    });

    it("returns null for empty input", () => {
        const result = validateKNEC("");
        expect(result.knec).toBeNull();
    });

    it("returns null for null input", () => {
        const result = validateKNEC(null);
        expect(result.knec).toBeNull();
    });
});

// =========================================================================
// CLASS NAME NORMALIZATION
// =========================================================================

describe("normalizeClassName", () => {
    it('strips "Class " prefix: "Class 4 West" → "4west"', () => {
        expect(normalizeClassName("Class 4 West")).toBe("4west");
    });

    it('strips "Grade " prefix: "Grade 4 West" → "4west"', () => {
        expect(normalizeClassName("Grade 4 West")).toBe("4west");
    });

    it('strips "Std " prefix: "Std 4 West" → "4west"', () => {
        expect(normalizeClassName("Std 4 West")).toBe("4west");
    });

    it('strips "Form " prefix: "Form 4 West" → "4west"', () => {
        expect(normalizeClassName("Form 4 West")).toBe("4west");
    });

    it('strips "g" prefix: "G4 West" → "4west"', () => {
        expect(normalizeClassName("G4 West")).toBe("4west");
    });

    it('strips "g." prefix: "G.4 West" → "4west"', () => {
        expect(normalizeClassName("G.4 West")).toBe("4west");
    });

    it("does NOT strip mid-word 'class': 'Geography' stays intact", () => {
        expect(normalizeClassName("Grade 10 Geography")).toBe("10geography");
    });

    it("does NOT strip mid-word 'grade': 'upgrade' stays intact", () => {
        expect(normalizeClassName("Form 2 Upgrade")).toBe("2upgrade");
    });

    it("does NOT strip 'Stage' prefix (not in the known prefix list)", () => {
        expect(normalizeClassName("Stage 2")).toBe("stage2");
    });

    it("does NOT strip mid-word 'form': 'platform' stays intact", () => {
        expect(normalizeClassName("Grade 5 Platform")).toBe("5platform");
    });

    it("handles lowercase input", () => {
        expect(normalizeClassName("class 4 west")).toBe("4west");
    });

    it("handles mixed case input", () => {
        expect(normalizeClassName("ClAsS 4 WeSt")).toBe("4west");
    });

    it("collapses multiple spaces", () => {
        expect(normalizeClassName("Class   4   West")).toBe("4west");
    });

    it("trims leading/trailing whitespace", () => {
        expect(normalizeClassName("  Class 4 West  ")).toBe("4west");
    });

    it("handles plain class name without prefix: '4 West'", () => {
        expect(normalizeClassName("4 West")).toBe("4west");
    });

    it("handles 'g' followed by whitespace", () => {
        expect(normalizeClassName("g 4 West")).toBe("4west");
    });
});

// =========================================================================
// PARENT NAME NORMALIZATION
// =========================================================================

describe("normalizeParentName", () => {
    it("lowercases and removes spaces", () => {
        expect(normalizeParentName("Nancy Onyinde")).toBe("nancyonyinde");
    });

    it("handles multiple spaces", () => {
        expect(normalizeParentName("John   Doe")).toBe("johndoe");
    });

    it("handles mixed case", () => {
        expect(normalizeParentName("JANE SMITH")).toBe("janesmith");
    });

    it("handles single name", () => {
        expect(normalizeParentName("Prince")).toBe("prince");
    });
});

// =========================================================================
// DUPLICATE DETECTION
// =========================================================================

describe("detectDuplicates", () => {
    const existingStudents: ExistingStudent[] = [
        { full_name: "Alice Wanjiku", date_of_birth: "2010-03-15", upi_number: "KP1234567A" },
        { full_name: "Bob Kimani", date_of_birth: "2011-07-22", upi_number: null },
    ];

    it("flags duplicate by UPI match", () => {
        const records: StagedStudentRecord[] = [
            createBaseRecord(0, {
                full_name: "Alice Wanjiku",
                upi_number: "KP1234567A",
                date_of_birth: "2010-03-15",
            }),
        ];
        const result = detectDuplicates(records, existingStudents);
        expect(result[0].isDuplicate).toBe(true);
    });

    it("flags duplicate by name + DOB match", () => {
        const records: StagedStudentRecord[] = [
            createBaseRecord(0, { full_name: "Bob Kimani", date_of_birth: "2011-07-22" }),
        ];
        const result = detectDuplicates(records, existingStudents);
        expect(result[0].isDuplicate).toBe(true);
    });

    it("does NOT flag record with same name but different DOB", () => {
        const records: StagedStudentRecord[] = [
            createBaseRecord(0, { full_name: "Bob Kimani", date_of_birth: "2012-01-10" }),
        ];
        const result = detectDuplicates(records, existingStudents);
        expect(result[0].isDuplicate).toBe(false);
    });

    it("does NOT flag entirely new record", () => {
        const records: StagedStudentRecord[] = [
            createBaseRecord(0, { full_name: "Carol Otieno", date_of_birth: "2012-05-10" }),
        ];
        const result = detectDuplicates(records, existingStudents);
        expect(result[0].isDuplicate).toBe(false);
    });

    it("preserves existing importAnyway flag", () => {
        const records: StagedStudentRecord[] = [
            createBaseRecord(0, {
                full_name: "Alice Wanjiku",
                upi_number: "KP1234567A",
                date_of_birth: "2010-03-15",
                importAnyway: true,
            }),
        ];
        const result = detectDuplicates(records, existingStudents);
        expect(result[0].isDuplicate).toBe(true);
        expect(result[0].importAnyway).toBe(true);
    });

    it("handles empty existing students list", () => {
        const records: StagedStudentRecord[] = [
            createBaseRecord(0, { full_name: "Alice Wanjiku", upi_number: "KP1234567A" }),
        ];
        const result = detectDuplicates(records, []);
        expect(result[0].isDuplicate).toBe(false);
    });

    it("is case-insensitive for name matching", () => {
        const records: StagedStudentRecord[] = [
            createBaseRecord(0, { full_name: "alice wanjiku", date_of_birth: "2010-03-15" }),
        ];
        const result = detectDuplicates(records, existingStudents);
        expect(result[0].isDuplicate).toBe(true);
    });
});

// =========================================================================
// FULL RECORD VALIDATION
// =========================================================================

describe("validateRecord", () => {
    const parentsMap = new Map<string, ParentRecord>();
    parentsMap.set("nancyonyinde", { id: "parent-uuid-1", full_name: "Nancy Onyinde" });

    const classesMap = new Map<string, ClassRecord>();
    classesMap.set("4west", { id: "class-uuid-1", name: "Class 4 West" });

    const defaultMapping = {
        nameColumns: ["full_name"],
        genderColumn: "gender",
        dobColumn: "date_of_birth",
        upiColumn: "upi_number",
        knecColumn: "knec_assessment_number",
        parentColumns: ["parent_name"],
        classColumns: ["class_name"],
    };

    const validRaw = {
        full_name: "John Kamau",
        gender: "male",
        date_of_birth: "15/03/2010",
        upi_number: "KP1234567A",
        knec_assessment_number: "ABC12345",
        parent_name: "Nancy Onyinde",
        class_name: "Class 4 West",
    };

    it("validates a fully valid record", () => {
        const result = validateRecord(0, validRaw, defaultMapping, parentsMap, classesMap);
        expect(result.isValid).toBe(true);
        expect(result.full_name).toBe("John Kamau");
        expect(result.gender).toBe("M");
        expect(result.date_of_birth).toBe("2010-03-15");
        expect(result.upi_number).toBe("KP1234567A");
        expect(result.knec_assessment_number).toBe("ABC12345");
        expect(result.cbc_student_parents_id).toBe("parent-uuid-1");
        expect(result.class_id).toBe("class-uuid-1");
        expect(result.errors).toEqual({});
    });

    it("errors when full_name is empty", () => {
        const result = validateRecord(0, { ...validRaw, full_name: "" }, defaultMapping);
        expect(result.isValid).toBe(false);
        expect(result.errors.full_name).toBe("Full name is required");
    });

    it("errors when full_name is whitespace-only", () => {
        const result = validateRecord(0, { ...validRaw, full_name: "   " }, defaultMapping);
        expect(result.isValid).toBe(false);
        expect(result.errors.full_name).toBe("Full name is required");
    });

    it("errors on unrecognized gender", () => {
        const result = validateRecord(0, { ...validRaw, gender: "alien" }, defaultMapping);
        expect(result.isValid).toBe(false);
        expect(result.errors.gender).toContain("Unrecognized gender value");
    });

    it("errors on invalid date format", () => {
        const result = validateRecord(
            0,
            { ...validRaw, date_of_birth: "not-a-date" },
            defaultMapping
        );
        expect(result.isValid).toBe(false);
        expect(result.errors.date_of_birth).toContain("Unrecognized date format");
    });

    it("errors on invalid UPI format", () => {
        const result = validateRecord(0, { ...validRaw, upi_number: "bad-upi" }, defaultMapping);
        expect(result.isValid).toBe(false);
        expect(result.errors.upi_number).toContain("Invalid UPI format");
    });

    it("errors on invalid KNEC format", () => {
        const result = validateRecord(
            0,
            { ...validRaw, knec_assessment_number: "@@" },
            defaultMapping
        );
        expect(result.isValid).toBe(false);
        expect(result.errors.knec_assessment_number).toContain(
            "Invalid KNEC assessment number format"
        );
    });

    it("adds advisory when parent not found in system", () => {
        const result = validateRecord(
            0,
            { ...validRaw, parent_name: "Unknown Parent" },
            defaultMapping,
            parentsMap,
            classesMap
        );
        expect(result.cbc_student_parents_id).toBeNull();
        expect(result.advisories.parent).toContain("Parent not found in system");
    });

    it("adds advisory when class not found in system", () => {
        const result = validateRecord(
            0,
            { ...validRaw, class_name: "Unknown Class" },
            defaultMapping,
            parentsMap,
            classesMap
        );
        expect(result.class_id).toBeNull();
        expect(result.advisories.class).toContain("Class not found in system");
    });

    it("handles missing parent and class lookup maps (degraded mode)", () => {
        const result = validateRecord(0, validRaw, defaultMapping, null, null);
        expect(result.cbc_student_parents_id).toBeNull();
        expect(result.class_id).toBeNull();
        expect(result.isValid).toBe(true); // parent/class are optional
    });

    it("handles omitted optional columns gracefully", () => {
        const minimalMapping = {
            nameColumns: ["full_name"],
            genderColumn: "gender",
            dobColumn: null,
            upiColumn: null,
            knecColumn: null,
            parentColumns: [],
            classColumns: [],
        };
        const minimalRaw = {
            full_name: "John Kamau",
            gender: "male",
        };
        const result = validateRecord(0, minimalRaw, minimalMapping);
        expect(result.isValid).toBe(true);
        expect(result.date_of_birth).toBeNull();
        expect(result.upi_number).toBeNull();
        expect(result.knec_assessment_number).toBeNull();
        expect(result.cbc_student_parents_id).toBeNull();
        expect(result.class_id).toBeNull();
    });

    it("adds advisory for ambiguous date", () => {
        const result = validateRecord(
            0,
            { ...validRaw, date_of_birth: "01/02/2010" },
            defaultMapping
        );
        expect(result.isValid).toBe(true);
        expect(result.advisories.date_of_birth).toContain("Ambiguous date");
    });

    it("handles multi-column name concatenation", () => {
        const multiNameMapping = { ...defaultMapping, nameColumns: ["first_name", "last_name"] };
        const raw = { ...validRaw, first_name: "John", last_name: "Kamau" };
        const result = validateRecord(0, raw, multiNameMapping);
        expect(result.full_name).toBe("John Kamau");
    });

    it("preserves original row index", () => {
        const result = validateRecord(42, validRaw, defaultMapping);
        expect(result._rowIndex).toBe(42);
    });
});

// =========================================================================
// PER-FIELD VALIDATION
// =========================================================================

describe("validateField", () => {
    const validRecord: StagedStudentRecord = createBaseRecord(0, {
        full_name: "John Kamau",
        gender: "M",
        date_of_birth: "2010-03-15",
        upi_number: "KP1234567A",
        knec_assessment_number: "ABC12345",
        isValid: true,
    });

    it("clears full_name error when value becomes valid", () => {
        const record = {
            ...validRecord,
            errors: { full_name: "Full name is required" },
            isValid: false,
        };
        const result = validateField(record, "full_name", "John Kamau");
        expect(result.errors).not.toHaveProperty("full_name");
        expect(result.isValid).toBe(true);
    });

    it("adds full_name error when value is cleared", () => {
        const result = validateField(validRecord, "full_name", "");
        expect(result.errors?.full_name).toBe("Full name is required");
        expect(result.isValid).toBe(false);
    });

    it("clears gender error when value becomes valid", () => {
        const record = {
            ...validRecord,
            errors: { gender: "Unrecognized gender value" },
            isValid: false,
            gender: null as "M" | "F" | null,
        };
        const result = validateField(record, "gender", "female");
        expect(result.errors).not.toHaveProperty("gender");
        expect(result.gender).toBe("F");
        expect(result.isValid).toBe(true);
    });

    it("adds gender error for invalid value", () => {
        const result = validateField(validRecord, "gender", "unknown");
        expect(result.errors?.gender).toContain("Unrecognized gender value");
        expect(result.isValid).toBe(false);
    });

    it("clears DOB error and adds advisory for ambiguous date", () => {
        const record = { ...validRecord, errors: { date_of_birth: "error" }, isValid: false };
        const result = validateField(record, "date_of_birth", "01/02/2010");
        expect(result.errors).not.toHaveProperty("date_of_birth");
        expect(result.date_of_birth).toBe("2010-02-01");
        expect(result.advisories?.date_of_birth).toContain("Ambiguous date");
        expect(result.isValid).toBe(true);
    });

    it("clears UPI error on valid input", () => {
        const record = { ...validRecord, errors: { upi_number: "Invalid UPI" }, isValid: false };
        const result = validateField(record, "upi_number", "KP1234567A");
        expect(result.errors).not.toHaveProperty("upi_number");
        expect(result.upi_number).toBe("KP1234567A");
    });

    it("clears KNEC error on valid input", () => {
        const record = {
            ...validRecord,
            errors: { knec_assessment_number: "Invalid KNEC" },
            isValid: false,
        };
        const result = validateField(record, "knec_assessment_number", "ABC12345");
        expect(result.errors).not.toHaveProperty("knec_assessment_number");
        expect(result.knec_assessment_number).toBe("ABC12345");
    });

    it("preserves existing errors from other fields", () => {
        const record = {
            ...validRecord,
            errors: { full_name: "Full name is required", gender: "Unrecognized gender" },
            isValid: false,
        };
        const result = validateField(record, "full_name", "John Kamau");
        expect(result.errors).not.toHaveProperty("full_name");
        expect(result.errors).toHaveProperty("gender");
        expect(result.isValid).toBe(false);
    });
});

// =========================================================================
// HELPERS
// =========================================================================

function createBaseRecord(
    rowIndex: number,
    overrides: Partial<StagedStudentRecord> = {}
): StagedStudentRecord {
    return {
        _rowIndex: rowIndex,
        full_name: "",
        gender: null,
        date_of_birth: null,
        upi_number: null,
        knec_assessment_number: null,
        cbc_student_parents_id: null,
        class_id: null,
        isValid: false,
        isDuplicate: false,
        importAnyway: false,
        errors: {},
        advisories: {},
        ...overrides,
    };
}
