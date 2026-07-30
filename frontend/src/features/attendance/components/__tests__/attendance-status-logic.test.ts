/**
 * Pure function tests that mirror the backend's ComputeDayStatus logic.
 * This verifies the frontend's understanding of the status mapping contract.
 */

// The backend's status-mapping logic (reproduced here for testing clarity).
// This duplicates the Go function in service.go:
//
//   func ComputeDayStatus(expectedCount, handledCount int) DayStatus {
//       switch {
//       case expectedCount == 0:            return DayStatusNone
//       case handledCount == expectedCount: return DayStatusGreen
//       case handledCount == 0:             return DayStatusRed
//       default:                           return DayStatusYellow
//       }
//   }

type DayStatus = "none" | "green" | "yellow" | "red";

function computeDayStatus(expectedCount: number, handledCount: number): DayStatus {
    if (expectedCount === 0) return "none";
    if (handledCount === expectedCount) return "green";
    if (handledCount === 0) return "red";
    return "yellow";
}

describe("computeDayStatus (mirrors backend ComputeDayStatus)", () => {
    it("returns 'none' when expected_count is 0", () => {
        expect(computeDayStatus(0, 0)).toBe("none");
    });

    it("returns 'green' when all expected slots are handled", () => {
        expect(computeDayStatus(6, 6)).toBe("green");
        expect(computeDayStatus(1, 1)).toBe("green");
        expect(computeDayStatus(8, 8)).toBe("green");
    });

    it("returns 'red' when no expected slots are handled", () => {
        expect(computeDayStatus(6, 0)).toBe("red");
        expect(computeDayStatus(1, 0)).toBe("red");
        expect(computeDayStatus(8, 0)).toBe("red");
    });

    it("returns 'yellow' when some but not all slots are handled", () => {
        expect(computeDayStatus(6, 3)).toBe("yellow");
        expect(computeDayStatus(6, 1)).toBe("yellow");
        expect(computeDayStatus(6, 5)).toBe("yellow");
    });

    it("treats SKIPPED sessions as handled (handled_count includes them)", () => {
        // 6 expected, 4 handled (2 marked + 2 SKIPPED) → yellow
        expect(computeDayStatus(6, 4)).toBe("yellow");
        // 6 expected, 6 handled (all marked or SKIPPED) → green
        expect(computeDayStatus(6, 6)).toBe("green");
    });
});
