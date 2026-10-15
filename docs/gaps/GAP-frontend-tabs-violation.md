# Gap: Frontend Tabs Violation – Student Detail Uses Tabs

**Area:** frontend
**Severity:** Medium
**Status:** Open
**Owner:** 
**Created:** 2026-10-15
**Related Issues:** #

## Description
`frontend/src/features/students/components/student-detail.tsx` uses shadcn `Tabs`, `TabsList`, `TabsTrigger`, `TabsContent`. `frontend/AGENTS.md` §8 Page Layout explicitly forbids tabs in pages/feature containers and requires vertical stacking with `space-y-*` and `<h2>/<h3>` headings.

## Current Behavior
Student detail renders:
```tsx
<Tabs defaultValue="overview">
  <TabsList>
    <TabsTrigger value="overview">Overview</TabsTrigger>
    ...
  </TabsList>
  <TabsContent ...>...</TabsContent>
</Tabs>
```
Feature spec `FEATURE-enhanced-student-profile.md` also shows tabs in wireframes.

## Expected Behavior
Page content is vertically stacked sections. No `<Tabs>` component in `src/app/` pages or feature containers. Navigation belongs in shell only.

## Impact
Violates architecture contract, creates inconsistent UX across the app, and will fail `pnpm lint` / design review. Sets precedent for other features.

## Reproduction Steps
1. Open `frontend/src/features/students/components/student-detail.tsx`
2. Observe import of `Tabs` from `@/components/ui/tabs`
3. Compare with `frontend/AGENTS.md` §8

## Proposed Solutions
* Refactor `StudentDetail` to vertical sections with `<h2>` headings for Overview, Academics, Guardians, Stats & KPIs, Notes.
* Update `FEATURE-enhanced-student-profile.md` wireframe to remove tabs.
* Audit other features for similar Tabs usage.

## Decision
Refactor to vertical stacking per AGENTS contract.

## Acceptance Criteria
- [ ] `student-detail.tsx` contains no `Tabs` import or usage
- [ ] Sections render as stacked `Card` blocks with `h2/h3` headings
- [ ] No dumb back buttons added
- [ ] Feature spec wireframe updated

## Links
* Component: `frontend/src/features/students/components/student-detail.tsx`
* Contract: `frontend/AGENTS.md` §8 Page Layout — no tabs
* Feature: `docs/features/FEATURE-enhanced-student-profile.md`
