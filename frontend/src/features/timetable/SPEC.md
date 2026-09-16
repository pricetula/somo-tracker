# Timetable Template Configuration UI — Step 1 Specification

**Feature:** `frontend/src/features/timetable`
**Workflow Step:** 1 of Timetable Setup
**Owner:** Platform Team
**Version:** 1.0.0

## Purpose

Implement the Timetable Template Configuration UI for Somotracker SIS. This is Step 1 of the timetable setup workflow, allowing school admins to create a bell schedule template with a two-step flow: metadata definition followed by time slot matrix configuration with a sticky slot column and Monday–Sunday weekly matrix.

## Database Context

### `timetable_templates`

Parent container for a school's bell schedule configuration.

- `id` UUID
- `school_id` UUID
- `name` VARCHAR — required, unique per school
- `description` TEXT — optional

### `time_slots`

Individual periods or breaks belonging to a template.

- `id` UUID
- `timetable_template_id` UUID FK → `timetable_templates(id)` cascade
- `name` VARCHAR — e.g., "Period 1", "Morning Break"
- `start_time` TIME
- `end_time` TIME
- `sequence_index` INT — implicitly managed by row array index
- `is_instructional` BOOLEAN — true for classes, false for breaks/recess

Grid Architecture:

- Horizontal axis: Monday to Sunday
- Vertical axis: Time ranges/slots
- First column sticky for slot identity visibility during horizontal scroll

## User Flow

### Part 1: Template Metadata Form

1. Render form with:
    - Template Name input (required, VARCHAR)
    - Description textarea (TEXT, optional)
2. **Next** button disabled until `name` is valid (non-empty, trimmed, passes backend validation)
3. On valid Next, transition to Part 2 with slide/fade animation. Preserve form state.

### Part 2: Time Slot Configuration & Weekly Matrix Table

Header: "Create Timetable Template > Step 2: Configure Time Slots & Weekly Matrix"
Subtext: "Define your daily bell schedule sequence. Duration and start times auto-cascade."

Matrix layout:

- Sticky Column 1: Slot configuration stacked vertically per row
- Columns 2-8: Monday – Sunday (future assignment cells)

**Sticky Column 1 Anatomy — stacked per row:**

1. **Slot Name Input** — editable text, pre-seeded with smart patterns: "Period 1", "Period 2", "Morning Break", etc.
2. **Time Pickers** — two inputs side-by-side: Start Time, End Time. Uses native time input or shadcn `TimePicker` if available. Automated duration cascading:
    - When a slot's end time changes, the next slot's start time auto-seeds to that end time.
    - Clicking "+ Add Row" auto-seeds start = previous end, end = previous end + default 1h.
3. **Instructional Checkbox** — toggle for `is_instructional`
4. **Delete Button** — removes row. Disabled if only one row remains.

Columns 2-8:

- Placeholder cells for future class/teacher/subject assignment. Non-instructional slots should visually span as breaks.
- For now render as empty, disabled, or placeholder text: "(Future Slot Assignment Cell)"

**Actions Footer:**

- `[ + Add Time Slot ]` — prepends new row at end with cascaded times
- `[ Back ]` — returns to Part 1, preserving data
- `[ Save Template ]` — validates and persists

## Component Architecture

Feature module structure follows `src/features/timetable/`:

- `components/` — presentational UI
- `hooks/` — data fetching/local state
- `services/` — API clients
- `types/` — TypeScript interfaces
- `index.ts` — public API exports

Public exports:

```ts
export { TimetableTemplateWizard } from "./components/timetable-template-wizard";
export { TimeSlotRow } from "./components/time-slot-row";
export { WeeklyMatrix } from "./components/weekly-matrix";
```

## UI Implementation Guidelines

- Use unadulterated minimalist Shadcn components. No custom Tailwind colors, no nested div wrappers.
- Follow Frontend AGENTS.md: no back buttons, no headers in feature components, no tabs.
- Sticky first column: `position: sticky; left: 0; z-index: 10; background: inherit`
- Horizontal scroll container for days columns
- Form implementation: shadcn `Form`, `FormField`, `FormItem`, `FormLabel`, `FormControl`, `FormMessage` with `zod` + `zodResolver`
- Flat table styling, no excessive borders/cards/shadows
- Vertical stacking with `space-y-*`

## State Management

Client-side state for draft template before save:

```ts
type DraftTemplate = {
    name: string;
    description?: string;
    slots: TimeSlotDraft[];
};

type TimeSlotDraft = {
    id: string; // temp uuid
    name: string;
    start_time: string; // "HH:mm"
    end_time: string; // "HH:mm"
    is_instructional: boolean;
};
```

`sequence_index` derived from array position on save.

Auto-cascade logic:

- On `end_time` change → update next slot `start_time` to new end_time if user has not manually edited it
- On add row → `start_time = lastSlot.end_time`, `end_time = add 60min`
- On delete row → re-index implicitly

## API Contract

Backend endpoints follow canonical error response contract.

Create template with slots:

```
POST /api/v1/schools/:schoolId/timetable-templates
Body: {
  name: string;
  description?: string;
  time_slots: Array<{
    name: string;
    start_time: string; // ISO time
    end_time: string;
    is_instructional: boolean;
  }>
}
Response: 201 { id, name, ... }
Error: { code, message, errors: { field_name: [...] } }
```

Fetch template for edit:

```
GET /api/v1/schools/:schoolId/timetable-templates/:id
```

## Validation

- Template name required, max length per DB
- Start time < End time per slot
- No overlapping slots in sequence
- At least one slot required
- Field errors surfaced via `form.setError`; generic errors via toast

Error handling per Frontend AGENTS.md:

- Use `ApiError` from `src/lib/api/client.ts`
- Use `getErrorMessage` for catches
- Mutation `onError` must call `toast.error(getErrorMessage(err))`
- 400 errors map to field errors, not toast

## Accessibility

- All inputs labelled
- Keyboard navigation through rows
- Delete button has aria-label "Delete time slot"
- Checkbox labelled "Instructional"

## Future Iterations

- Columns 2-8 will host class/teacher/subject assignment UI
- Drag-and-drop reorder of slots
- Copy template to another term
- Bulk import/export

## Implementation Checklist

- [ ] Create `types/timetable-template.ts` with TypeScript interfaces
- [ ] Create `services/timetable-api.ts` with `createTemplate`, `getTemplate`
- [ ] Create `hooks/use-timetable-wizard.ts` for draft state and cascade logic
- [ ] Create `components/timetable-template-wizard.tsx` — two-step stepper
- [ ] Create `components/time-slot-row.tsx` — sticky column stacked inputs
- [ ] Create `components/weekly-matrix.tsx` — scrollable days grid
- [ ] Wire zod schema for metadata and slot validation
- [ ] Ensure Next button disabled until name valid
- [ ] Implement auto-cascade on end time change and add row
- [ ] Ensure Save Template persists via API and redirects on success
- [ ] Add `FeatureHelp` tooltip integration from `content/docs`
- [ ] Run `pnpm lint` and `pnpm run audit:docs`

## References

- Backend error middleware: `backend/internal/middleware/errors.go`
- Frontend API client: `src/lib/api/client.ts`
- Frontend AGENTS.md for architecture constraints
- Database schema: `backend/db/migrations/*`
