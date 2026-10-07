# Upload Components

Directory: `frontend/src/features/import/components/upload/`

This directory contains the complete upload-to-map pipeline for the **Admin Import** feature. It handles file ingestion (CSV/Excel), field auto-mapping, manual mapping UI, and validation — producing a `MappingResult` ready for the save/import step.

---

## Component Overview

| File                   | Export               | Responsibility                                                                     |
| ---------------------- | -------------------- | ---------------------------------------------------------------------------------- |
| `upload.tsx`           | `Upload`             | Top-level orchestration: file upload → field mapper → result callback              |
| `upload-file.tsx`      | `UploadFile`         | Drag-and-drop / click file input; parses CSV (PapaParse) & Excel (SheetJS)         |
| `field-mapper.tsx`     | `FieldMapper`, types | Auto-mapping logic, per-field dropdown UI, validation, preview, result computation |
| `field-mapper-row.tsx` | `FieldMapperRow`     | Single-field row: shows mapped columns, add/remove, preview badges                 |
| `index.ts`             | `Upload`             | Public barrel export                                                               |

---

## Data Flow

```
Upload (parent)
  │
  ├─► UploadFile.onUploaded(raw: Record<string, object>[])
  │       │
  │       └─► adds __id to each row → ExtractedRow[]
  │
  └─► FieldMapper
          │
          ├─► computeAutoMapping() → initial mapping
          ├─► User adjusts via FieldMapperRow dropdowns
          ├─► computeResult() → MappingResult { mapping, rows[], isValid }
          │
          └─► onMappingChange(result) → parent saves/imports
```

---

## Key Types (from `field-mapper.tsx`)

```ts
// Desired output field definition
interface FieldDef {
    key: string; // canonical key in final object
    label: string; // UI label
    aliases: string[]; // header aliases for auto-match
    required: boolean; // must be mapped for validity
    validator?: (value: unknown, row: Record<string, unknown>) => string | null;
}

// One row from parsed file (with synthetic __id)
interface ExtractedRow {
    __id: string;
    [column: string]: unknown;
}

// Result of mapping a single row
interface MappedRow {
    id: string;
    data: Record<string, unknown>;
    errors?: Record<string, string>;
}

// Final result passed to onMappingChange
interface MappingResult {
    mapping: Record<string, string[] | null>; // fieldKey → source column(s)
    rows: MappedRow[];
    isValid: boolean; // all required mapped + no validation errors
}
```

---

## Upload (Entry Point)

**File:** `upload.tsx`  
**Props:** `{ onCancel: () => void }`

- Owns state: `extractedRows`, `mappingResult`
- Renders `UploadFile` when no file uploaded
- Renders `FieldMapper` once rows exist
- Hard-coded `ADMIN_FIELDS` schema (full_name, email, role, school_slug)
- **Usage:** Used by the import modal/page; parent handles `onCancel` and `onMappingChange` (via FieldMapper's internal callback)

```tsx
<Upload onCancel={() => setShowImport(false)} />
```

---

## UploadFile (File Ingestion)

**File:** `upload-file.tsx`  
**Props:** `{ onUploaded: (rows: Record<string, object>[]) => void }`

### Features

- **Drag & drop** zone with visual feedback (`dragActive` state)
- **Click to select** (hidden `<input type="file" multiple>`)
- **Accepts:** `.csv`, `.xlsx`, `.xls`
- **Lazy-loads** parsers via dynamic `import()`:
    - `papaparse` for CSV
    - `xlsx` (SheetJS) for Excel
- **Multiple files:** merges rows from all accepted files
- **Error handling:** shows inline error banner; does not throw

### Parsing Logic

```ts
// CSV
Papa.parse(text, { header: true, skipEmptyLines: true });

// Excel (first sheet only)
XLSX.read(data, { type: "array" });
XLSX.utils.sheet_to_json(ws);
```

---

## FieldMapper (Mapping Orchestration)

**File:** `field-mapper.tsx`  
**Props:** `FieldMapperProps { extractedRows, desiredFields, onMappingChange }`

### Auto-Mapping Algorithm (`computeAutoMapping`)

1. Extract source columns from first row (excludes `__id`)
2. For each desired field, normalize `(key, label, aliases)`
3. Match against normalized source columns:
    - Exact match
    - Substring inclusion (either direction)
4. First match wins → `mapping[fieldKey] = [matchedColumn]`

### Result Computation (`computeResult`)

For each row & field:

- Pull values from mapped source columns
- Join multiple columns with space
- Run validator (if provided) → collect errors
- Build `MappedRow { id, data, errors? }`

### Validity (`isValid`)

- All required fields have at least one mapped column
- Zero validation errors across all rows

### Memoization

- `sourceCols` derived from `extractedRows[0]`
- `result` memoized on `[extractedRows, desiredFields, mapping]`
- Auto-recompute mapping when rows or fields change (via refs + `useEffect`)

### UI Sections

1. **Header bar:** mapped count / total, validation error count, "Auto-match" button
2. **Field rows:** one `FieldMapperRow` per desired field
3. **Footer:** status badge (green/amber), "Save mapping" button (disabled if `!isValid`)

---

## FieldMapperRow (Per-Field UI)

**File:** `field-mapper-row.tsx`  
**Props:** `FieldMapperRowProps { field, mapping, sourceCols, extractedRows, onChange, onRemove }`

### States

| State                   | Rendered                                                       |
| ----------------------- | -------------------------------------------------------------- |
| **Unmapped** (required) | Red "Required" badge + "Missing required mapping" text         |
| **Unmapped** (optional) | Dropdown trigger "Choose column"                               |
| **Mapped**              | Badge per mapped column (with ✕ remove), "Add column" dropdown |

### Dropdown Options

- **When unmapped:** "Unmapped" + all source columns
- **When mapped:** only _available_ (not yet used for this field) columns

### Preview Badges

- Shows up to 3 sample values from first rows
- Truncates at 28 chars with ellipsis
- Uses `Badge variant="outline"`

---

## Extending for Other Import Types

To reuse this pipeline for a different entity (e.g., "students", "teachers"):

1. **Define a new `FieldDef[]` schema** (replace `ADMIN_FIELDS` in `upload.tsx` or pass as prop)
2. **Add validators** per field (email format, enum check, etc.)
3. **Handle `onMappingChange`** in parent to POST the `MappingResult.rows` to your API

Example schema:

```ts
const STUDENT_FIELDS: FieldDef[] = [
    { key: "student_id", label: "Student ID", aliases: ["id", "sid"], required: true },
    { key: "full_name", label: "Full name", aliases: ["name"], required: true },
    {
        key: "grade",
        label: "Grade",
        aliases: ["year", "level"],
        required: true,
        validator: (v) => (/^\d+$/.test(String(v)) ? null : "Must be numeric"),
    },
    {
        key: "email",
        label: "Email",
        aliases: ["mail"],
        required: false,
        validator: (v) => (v && !String(v).includes("@") ? "Invalid email" : null),
    },
];
```

---

## Dependencies

| Package             | Purpose              | Load Strategy             |
| ------------------- | -------------------- | ------------------------- |
| `papaparse`         | CSV parsing          | Dynamic `import()` (lazy) |
| `xlsx` (SheetJS)    | Excel parsing        | Dynamic `import()` (lazy) |
| `@/components/ui/*` | Shadcn UI primitives | Static import             |
| `lucide-react`      | Icons                | Static import             |

---

## Error Handling Conventions

- **No empty catches:** All `catch` blocks set user-facing error state
- **No log-and-return:** Errors surfaced to UI via `setError()`
- **Validation errors:** Collected per-field in `MappedRow.errors`, displayed in preview badges (future: could show inline per-row)
- **Parse errors:** First error per file shown in red banner

---

## Accessibility Notes

- Drop zone: `role="button" tabIndex={0} aria-label`
- Dropdowns: Radix-based `DropdownMenu` (keyboard navigable)
- Buttons: `aria-label` on icon-only buttons
- Status badge: `aria-hidden="true"` (decorative), text label adjacent

---

## Testing Checklist

- [ ] CSV with headers matching aliases → auto-maps correctly
- [ ] Excel (.xlsx) single sheet → parses first sheet only
- [ ] Multiple files → rows concatenated
- [ ] Required field unmapped → `isValid = false`, red badge shown
- [ ] Validator returns string → error appears in row preview
- [ ] "Auto-match" button → re-runs auto-mapping
- [ ] Add/remove columns via dropdowns → mapping updates reactively
- [ ] Drag-and-drop + click both work
- [ ] Error banner appears for unsupported file type
- [ ] Error banner appears for parse failure

---

## Future Improvements

- [ ] Per-row validation error expansion (click to see all)
- [ ] Column type inference (date, number, boolean)
- [ ] Mapping persistence (localStorage / server)
- [ ] Large file streaming parse (web worker)
- [ ] Unit tests for `computeAutoMapping` / `computeResult`
- [ ] E2E test for full upload→map→save flow
