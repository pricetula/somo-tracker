# Student Bulk Import — Test Scenarios & Coverage Map

This document catalogs all test scenarios for the student bulk import feature,
organized by phase and component. Use this as a reference when writing or
reviewing tests.

---

## 1. Validation — Pure Function Tests

| # | Scenario | Expected | Test File |
|---|----------|----------|-----------|
| 1.1 | `normalizeGender("male")` | → `{ gender: "M" }` | `validation.test.ts` |
| 1.2 | `normalizeGender("female")` | → `{ gender: "F" }` | ↑ |
| 1.3 | `normalizeGender("ME")` (Swahili mwanaume) | → `{ gender: "M" }` | ↑ |
| 1.4 | `normalizeGender("KE")` (Swahili kike) | → `{ gender: "F" }` | ↑ |
| 1.5 | `normalizeGender("b")` / `"boy"` / `"1"` | → `{ gender: "M" }` | ↑ |
| 1.6 | `normalizeGender("g")` / `"girl"` / `"2"` | → `{ gender: "F" }` | ↑ |
| 1.7 | `normalizeGender("unknown")` | → `{ gender: null, error }` | ↑ |
| 1.8 | `normalizeGender("")` / `null` / `undefined` | → `{ gender: null, error }` | ↑ |
| 1.9 | `normalizeGender("  male  ")` | → trimmed to `"M"` | ↑ |
| 1.10 | `parseDate("2010-03-15")` (ISO) | → `{ date: "2010-03-15" }` | ↑ |
| 1.11 | `parseDate("15/03/2010")` (DD/MM/YYYY) | → `{ date: "2010-03-15" }` | ↑ |
| 1.12 | `parseDate("01/02/2010")` (ambiguous) | → advisory added, date = `2010-02-01` | ↑ |
| 1.13 | `parseDate("15-03-2010")` (dash-separated) | → `{ date: "2010-03-15" }` | ↑ |
| 1.14 | `parseDate("not-a-date")` | → `{ date: null, error }` | ↑ |
| 1.15 | `parseDate("")` / `null` / `undefined` | → `{ date: null }` (no error) | ↑ |
| 1.16 | `parseDate("32/03/2010")` (invalid day) | → `{ date: null, error }` | ↑ |
| 1.17 | `parseDate("15/13/2010")` (invalid month) | → `{ date: null, error }` | ↑ |
| 1.18 | `validateUPI("KP1234567A")` | → `{ upi: "KP1234567A" }` | ↑ |
| 1.19 | `validateUPI("kp1234567a")` (lowercase) | → `{ upi: "KP1234567A" }` (uppercased) | ↑ |
| 1.20 | `validateUPI("12345678")` (no KP prefix) | → `{ upi: null, error }` | ↑ |
| 1.21 | `validateUPI("KP123456A")` (wrong length) | → `{ upi: null, error }` | ↑ |
| 1.22 | `validateUPI("")` / `null` | → `{ upi: null }` | ↑ |
| 1.23 | `validateKNEC("ABC12345")` | → `{ knec: "ABC12345" }` | ↑ |
| 1.24 | `validateKNEC("ABC-1234")` (with hyphen) | → valid | ↑ |
| 1.25 | `validateKNEC("AB12")` (too short) | → `{ knec: null, error }` | ↑ |
| 1.26 | `validateKNEC("@#invalid")` (special chars) | → `{ knec: null, error }` | ↑ |
| 1.27 | `normalizeClassName("Class 4 West")` | → `"4west"` | ↑ |
| 1.28 | `normalizeClassName("Grade 10 Geography")` | → `"10geography"` (mid-word intact) | ↑ |
| 1.29 | `normalizeClassName("Stage 2")` | → `"stage2"` (prefix not stripped) | ↑ |
| 1.30 | `normalizeClassName("G4 West")` / `"g.4 west"` | → `"4west"` | ↑ |
| 1.31 | `normalizeParentName("Nancy Onyinde")` | → `"nancyonyinde"` | ↑ |
| 1.32 | `detectDuplicates` — UPI match | → `isDuplicate: true` | ↑ |
| 1.33 | `detectDuplicates` — name + DOB match | → `isDuplicate: true` | ↑ |
| 1.34 | `detectDuplicates` — no match | → `isDuplicate: false` | ↑ |
| 1.35 | `detectDuplicates` — same name, different DOB | → `isDuplicate: false` | ↑ |
| 1.36 | `detectDuplicates` — preserves `importAnyway` | → flag kept | ↑ |
| 1.37 | `validateRecord` — fully valid record | → `isValid: true`, all fields parsed | ↑ |
| 1.38 | `validateRecord` — missing name | → `isValid: false`, `errors.full_name` | ↑ |
| 1.39 | `validateRecord` — invalid gender | → `isValid: false`, `errors.gender` | ↑ |
| 1.40 | `validateRecord` — invalid date | → `isValid: false`, `errors.date_of_birth` | ↑ |
| 1.41 | `validateRecord` — invalid UPI | → `isValid: false`, `errors.upi_number` | ↑ |
| 1.42 | `validateRecord` — invalid KNEC | → `isValid: false`, `errors.knec_assessment_number` | ↑ |
| 1.43 | `validateRecord` — parent not in system | → advisory, `parent_id = null` | ↑ |
| 1.44 | `validateRecord` — class not in system | → advisory, `class_id = null` | ↑ |
| 1.45 | `validateRecord` — degraded mode (null maps) | → parent/class null, no crash | ↑ |
| 1.46 | `validateRecord` — multi-column name concat | → name joined with space | ↑ |
| 1.47 | `validateRecord` — optional columns omitted | → optional fields null, no error | ↑ |
| 1.48 | `validateRecord` — ambiguous date | → advisory, not error | ↑ |
| 1.49 | `validateField` — clear error on valid edit | → error removed, isValid recalculated | ↑ |
| 1.50 | `validateField` — add error on invalid edit | → error added, isValid recalculated | ↑ |
| 1.51 | `validateField` — preserve other field errors | → other errors unchanged | ↑ |

---

## 2. IndexedDB — Persistence Tests

| # | Scenario | Expected | Test File |
|---|----------|----------|-----------|
| 2.1 | Save session, load it back | → same data returned | `indexeddb.test.ts` |
| 2.2 | Load when no session exists | → `null` | ↑ |
| 2.3 | Clear session data | → load returns `null` | ↑ |
| 2.4 | Update session step | → step changed, timestamp updated | ↑ |
| 2.5 | Update step when no session exists | → no crash, no-op | ↑ |
| 2.6 | Overwrite existing session | → new values persisted | ↑ |
| 2.7 | `hasStoredSession` → true when session exists | → `true` | ↑ |
| 2.8 | `hasStoredSession` → false when empty | → `false` | ↑ |
| 2.9 | Save records, load them back | → same array | ↑ |
| 2.10 | Load records when none stored | → `[]` | ↑ |
| 2.11 | Update single record by rowIndex | → that record updated | ↑ |
| 2.12 | Update record with non-existent rowIndex | → no-op, other records unchanged | ↑ |
| 2.13 | Update record when no records table exists | → `[]` returned | ↑ |
| 2.14 | Batch update multiple records | → all updated | ↑ |
| 2.15 | Batch update with non-existent index | → that update skipped | ↑ |
| 2.16 | Overwrite all records | → old records replaced | ↑ |
| 2.17 | Save parsed file meta, load back | → same meta | ↑ |
| 2.18 | Load parsed file meta when none exists | → `null` | ↑ |
| 2.19 | Clear parsed file meta | → `null` after clear | ↑ |

---

## 3. Session Recovery — Hook Tests

| # | Scenario | Expected | Test File |
|---|----------|----------|-----------|
| 3.1 | Hook mounts → loading state | → `action: "loading"` | `use-session-recovery.test.ts` |
| 3.2 | No session in IndexedDB → clear | → `action: "clear"`, `session: null` | ↑ |
| 3.3 | Session found in IndexedDB → prompt | → `action: "prompt"`, session loaded | ↑ |
| 3.4 | Resume action → transitions to clear | → `action: "clear"` | ↑ |
| 3.5 | Discard action → clears IndexedDB, goes clear | → IndexedDB empty, `action: "clear"` | ↑ |
| 3.6 | IndexedDB read error (throw) → falls to clear | → `action: "clear"`, no crash | ↑ |

---

## 4. Lookup Hooks — HTTP & Retry Tests

| # | Scenario | Expected | Test File |
|---|----------|----------|-----------|
| 4.1 | `useParentLookup` → loading state | → `parentsLoading: true` | `use-lookups.test.ts` |
| 4.2 | `useParentLookup` → success | → map keyed by normalized name | ↑ |
| 4.3 | `useParentLookup` → API error | → `parentsError` set, map empty | ↑ |
| 4.4 | `useParentLookup` → retry after error | → retry succeeds, error cleared | ↑ |
| 4.5 | `useClassLookup` → loading state | → `classesLoading: true` | ↑ |
| 4.6 | `useClassLookup` → success | → map keyed by normalized name | ↑ |
| 4.7 | `useClassLookup` → API error | → `classesError` set, map empty | ↑ |
| 4.8 | `useClassLookup` → retry | → retry succeeds, error cleared | ↑ |
| 4.9 | `useExistingStudents` → loading state | → `existingStudentsLoading: true` | ↑ |
| 4.10 | `useExistingStudents` → success | → array returned | ↑ |
| 4.11 | `useExistingStudents` → API error | → `error` set, `[]` returned | ↑ |
| 4.12 | `useLookups` (combined) → all three load | → maps populated + students loaded | ↑ |
| 4.13 | `useLookups` → parents fail, classes succeed | → degraded mode for parents only | ↑ |
| 4.14 | `useLookups` → retry parents after combined failure | → parents recovered | ↑ |

---

## 5. Component Tests (Rendering & Interaction)

### 5.1 SessionRecoveryBanner

| # | Scenario | Expected | Test File |
|---|----------|----------|-----------|
| 5.1.1 | Renders with session timestamp | → timestamp displayed in message | `session-recovery-banner.test.tsx` |
| 5.1.2 | Renders Resume Session button | → button present | ↑ |
| 5.1.3 | Renders Discard & Start New button | → button present | ↑ |
| 5.1.4 | Click Resume → calls onResume | → callback invoked | ↑ |
| 5.1.5 | Click Discard → calls onDiscard | → callback invoked | ↑ |
| 5.1.6 | Manual ingestion pattern session | → renders without error | ↑ |

### 5.2 LookupWarningBanner

| # | Scenario | Expected | Test File |
|---|----------|----------|-----------|
| 5.2.1 | parents type → shows "Parent linking unavailable" | → label visible | `lookup-warning-banner.test.tsx` |
| 5.2.2 | classes type → shows "Class linking unavailable" | → label visible | ↑ |
| 5.2.3 | Error message displayed | → message text visible | ↑ |
| 5.2.4 | Retry Lookup button rendered | → button present | ↑ |
| 5.2.5 | Click Retry → calls onRetry | → callback invoked | ↑ |

### 5.3 IngestionSelector

| # | Scenario | Expected | Test File |
|---|----------|----------|-----------|
| 5.3.1 | Renders title "Import Students" | → title visible | `ingestion-selector.test.tsx` |
| 5.3.2 | Renders Manual Entry option | → card visible | ↑ |
| 5.3.3 | Renders Upload File option | → card visible | ↑ |
| 5.3.4 | Click Manual Entry → calls `onSelect("manual")` | → callback with "manual" | ↑ |
| 5.3.5 | Click Upload File → calls `onSelect("csv")` | → callback with "csv" | ↑ |
| 5.3.6 | File size (10MB) and row limit (5K) shown | → limits visible in aside | ↑ |

### 5.4 ManualEntryGrid

| # | Scenario | Expected | Test File |
|---|----------|----------|-----------|
| 5.4.1 | Column headers present (Name, Gender, DOB, UPI, KNEC, Parent, Class) | → all visible | `manual-entry-grid.test.tsx` |
| 5.4.2 | Default single empty row rendered | → one name input visible | ↑ |
| 5.4.3 | Add Row button visible | → button present | ↑ |
| 5.4.4 | Click Add Row → calls onAddRow | → callback invoked | ↑ |
| 5.4.5 | Validate & Review button disabled when no filled rows | → button disabled | ↑ |
| 5.4.6 | Validate & Review enabled when name filled | → button enabled | ↑ |
| 5.4.7 | Click Validate → calls onProceed | → callback invoked | ↑ |
| 5.4.8 | Remove row button → calls onRemoveRow with index | → callback invoked | ↑ |
| 5.4.9 | Remove button disabled when single row | → button disabled | ↑ |
| 5.4.10 | Name input change → calls onUpdateRow | → callback with field, value | ↑ |
| 5.4.11 | Gender select change → calls onUpdateRow | → callback with "M"/"F" | ↑ |
| 5.4.12 | DOB input change → calls onUpdateRow | → callback with value | ↑ |
| 5.4.13 | Filled count displayed | → "N filled" text visible | ↑ |

### 5.5 FileDropzone

| # | Scenario | Expected | Test File |
|---|----------|----------|-----------|
| 5.5.1 | Title and format info rendered | → CSV, XLSX, 10MB, 5K info visible | `file-dropzone.test.tsx` |
| 5.5.2 | Dropzone area with instructions | → "Drop a CSV or Excel file here" visible | ↑ |
| 5.5.3 | Hidden file input with .csv/.xlsx/.xls accept | → input present, class "hidden" | ↑ |
| 5.5.4 | Back button rendered | → button present | ↑ |
| 5.5.5 | Click Back → calls onBack | → callback invoked | ↑ |

### 5.6 ResultsSummary

| # | Scenario | Expected | Test File |
|---|----------|----------|-----------|
| 5.6.1 | Success state → "Import Complete" + count | → success message visible | `results-summary.test.tsx` |
| 5.6.2 | Success with 1 record → singular "student" | → "1 student" visible | ↑ |
| 5.6.3 | Success → "Start New Import" button | → button present | ↑ |
| 5.6.4 | Partial (207) → "Partial Success" + counts | → header + numbers visible | ↑ |
| 5.6.5 | Partial → failed records listed with errors | → each failure + error displayed | ↑ |
| 5.6.6 | Partial → field-level errors shown | → field error text visible | ↑ |
| 5.6.7 | Partial → "Retry Failed" and "Start New" buttons | → both buttons present | ↑ |
| 5.6.8 | Error state → "Import Failed" + message | → failure text visible | ↑ |
| 5.6.9 | Error state without message → fallback text | → "unexpected error occurred" shown | ↑ |
| 5.6.10 | Error state → "Retry" and "Start New" buttons | → both buttons present | ↑ |
| 5.6.11 | Click Retry in partial → calls onRetry | → callback invoked | ↑ |
| 5.6.12 | Click Start New in success → calls onStartNew | → callback invoked | ↑ |

### 5.7 ValidationMotor (Phase 4)

| # | Scenario | Expected | Test File |
|---|----------|----------|-----------|
| 5.7.1 | Errors & Warnings / All Records toggle visible | → both toggles present | `validation-motor.test.tsx` |
| 5.7.2 | Click All Records → calls onViewFilterChange("all") | → callback invoked | ↑ |
| 5.7.3 | Active toggle highlighted with bg-background | → active class present | ↑ |
| 5.7.4 | Error count badge displayed | → "N errors" visible | ↑ |
| 5.7.5 | Duplicate warning count displayed | → "N duplicates" visible | ↑ |
| 5.7.6 | Total record count displayed | → "N records" visible | ↑ |
| 5.7.7 | Submit Import button with record count | → button + count visible | ↑ |
| 5.7.8 | Submit disabled when errors > 0 | → button disabled | ↑ |
| 5.7.9 | Submit enabled when errors === 0 | → button enabled | ↑ |
| 5.7.10 | Submit disabled + shows "Submitting…" when submitting | → button disabled, text changed | ↑ |
| 5.7.11 | Click Submit → calls onSubmit | → callback invoked | ↑ |
| 5.7.12 | Back button present | → button visible | ↑ |
| 5.7.13 | Click Back → calls onBack | → callback invoked | ↑ |
| 5.7.14 | Duplicate rows show "Possible duplicate" text | → advisory text visible | ↑ |
| 5.7.15 | Duplicate rows have import-anyway checkbox | → checkbox present | ↑ |
| 5.7.16 | Click checkbox → calls onToggleImportAnyway | → callback with rowIndex | ↑ |
| 5.7.17 | Non-duplicate rows hide checkbox | → checkbox not present | ↑ |
| 5.7.18 | Parent-not-found advisory displayed inline | → advisory text visible | ↑ |
| 5.7.19 | Class-not-found advisory displayed inline | → advisory text visible | ↑ |
| 5.7.20 | Ambiguous date advisory displayed inline | → advisory text visible | ↑ |
| 5.7.21 | All column headers rendered | → Name, Gender, DOB, UPI, KNEC, Parent, Class | ↑ |

---

## 6. SSE Progress Tracking Tests

| # | Scenario | Expected | Test File |
|---|----------|----------|-----------|
| 6.1 | Component mounts → EventSource connects to correct URL | → URL contains `/students/track/:id/sse` | `StudentImportProgress.test.tsx` |
| 6.2 | Initial state before any event → "Processing your import…" | → pending indicator visible | ↑ |
| 6.3 | Progress event at 50% → shows counts + progress bar | → 50/100 records, counts displayed | ↑ |
| 6.4 | Multiple sequential progress events (25% → 50% → 75%) | → each step updates display | ↑ |
| 6.5 | Progress includes failed count | → "N failed" displayed | ↑ |
| 6.6 | `import_finished` with no errors → "Import complete" | → success message visible | ↑ |
| 6.7 | `import_finished` with errors → "Completed with errors" | → error message visible | ↑ |
| 6.8 | `import_finished` → EventSource.close() called | → close() invoked | ↑ |
| 6.9 | `import_finished` → onDone callback called | → callback invoked | ↑ |
| 6.10 | `import_finished` with zero records → graceful display | → no crash | ↑ |
| 6.11 | Component unmount → EventSource.close() called | → close() invoked | ↑ |
| 6.12 | Component unmount → onClose callback called | → callback invoked | ↑ |
| 6.13 | Malformed event data → no crash | → component stays in pending state | ↑ |
| 6.14 | Progress bar has role="progressbar" + aria attributes | → accessible progressbar | ↑ |
| 6.15 | Progress bar at 0% when totalRecords is 0 | → aria-valuenow = 0 | ↑ |
| 6.16 | No reconnection after import_finished (even on error) | → EventSource count unchanged | ↑ |

---

## 7. Integration / End-to-End Workflow Scenarios (Manual / Smoke Tests)

These scenarios should be verified by running the application end-to-end,
either manually or via a Playwright/Cypress test suite.

### 7.1 Pattern A: Manual Entry → Validation → Success

```
1. Open /students/import
2. Verify no existing session → selector screen shown
3. Click Manual Entry
4. Grid displayed with one empty row
5. Fill: "John Kamau", Gender=M, DOB=15/03/2010, UPI=KP1234567A
6. Click Add Row → second row appears
7. Fill: "Jane", Gender=F (no other fields)
8. Click "Validate & Review"
9. Validation screen shows:
   - Row 0: green (valid)
   - Row 1: amber (error — missing name)
10. Fix row 1: add "Jane Wanjiku"
11. Error count → 0
12. Click Submit Import
13. Success screen: "Successfully imported 2 students"
14. Verify IndexedDB session cleared
```

### 7.2 Pattern B: CSV File Upload → Column Mapping → Validation → Partial Success

```
1. Open /students/import
2. Click Upload File
3. Drop a CSV with columns: [First Name, Last Name, Gender, DOB, UPI]
4. Verify file parsed, headers detected
5. Wizard Step 1 (Name): map "First Name" + "Last Name" → confirm
6. Wizard Step 2 (Gender): map "Gender" column → confirm
7. Wizard Step 3 (DOB): map "DOB" column → confirm
8. Wizard Step 4 (UPI): map "UPI" column → confirm
9. Wizard Step 5 (KNEC): skip → confirm
10. Wizard Step 6 (Parent): skip → confirm
11. Wizard Step 7 (Class): skip → confirm
12. "Processing records…" → validation screen
13. Verify some rows have errors (e.g., bad UPI format)
14. Click All Records → see all rows
15. Fix errors inline
16. Submit
17. Partial success screen (207):
    - 8 imported, 2 failed
    - Failed rows listed with reasons
18. Click "Retry Failed"
    - Only failed rows resubmitted
```

### 7.3 Session Recovery on Refresh

```
1. Start a manual entry with 5 partially filled rows
2. Refresh the browser tab
3. Recovery banner appears: "You have an unfinished import session from [time]"
4. Click "Resume Session"
5. Verify wizard rehydrates to previous step with all data intact
6. Click "Discard & Start New"
7. Selector screen shown, IndexedDB cleared
```

### 7.4 Degraded Mode (Lookup Failure)

```
1. Mock the parents API to return 500
2. Open /students/import
3. Warning banner appears: "Parent linking unavailable — …"
4. Proceed with manual entry anyway
5. In validation screen, parent column shows "Not found" for all rows
6. Click "Retry Lookup" on the banner
7. Parents load successfully → parent field auto-matches
```

### 7.5 Duplicate Detection During Validation

```
1. Mock existing students to have "Alice Wanjiku, 2010-03-15"
2. Upload CSV with matching row
3. In validation screen:
   - Duplicate row has amber left border
   - "Possible duplicate: matches existing student" text
   - "Import anyway" checkbox unchecked
4. Check "Import anyway"
5. Row fades to resolved styling
6. Submit Import includes the row
```

### 7.6 File Too Large / Too Many Rows

```
1. Drop a 12MB CSV file → toast error "File too large (12MB). Maximum is 10MB."
2. Drop a CSV with 6,000 rows → toast error "… maximum supported is 5,000"
```

### 7.7 Submission Timeout

```
1. Start an import with 100 rows
2. Mock network to hang for > 30 seconds
3. After 30s, toast error "Request timed out after 30 seconds"
4. Staged data preserved in IndexedDB
5. Click Retry → resubmits without re-uploading
```

### 7.8 Column Conflict Guard (Pattern B)

```
1. Upload CSV with columns: [Name, Gender, DOB, UPI, Name] (duplicate header)
2. In Name step, select "Name"
3. In Gender step, "Name" is dimmed with "(mapped to Student Name)"
4. Selecting "Name" in Gender step shows warning:
   "This column is already mapped to Student Name. Re-mapping will remove it from that step."
5. Confirm → removes from Name step
```

### 7.9 SSE Progress Tracking

```
1. Start a large import (500+ records)
2. SSE event stream delivers progress updates:
   - "Processing…" → "45/500 records" → "230/500" → "500/500"
3. Progress bar updates smoothly at each event
4. On completion → "Import complete" displayed
5. EventSource connection closed
6. If connection drops mid-stream, polling fallback picks up
```

---

## Coverage Summary

| Area | Scenarios |
|------|-----------|
| Validation pure functions | 51 |
| IndexedDB persistence | 19 |
| Session recovery hook | 6 |
| Lookup hooks | 14 |
| Component rendering + interaction | 62 |
| SSE progress tracking | 16 |
| Integration / E2E smoke tests | 9 |
| **Total** | **177 scenarios** |
