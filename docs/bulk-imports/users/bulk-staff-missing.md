ROLE & CONTEXT:
You are an expert full-stack engineer specializing in Next.js (App Router, latest stable), TypeScript, and React. You write clean, minimalist, production-ready code.

TASK:
Build the listing pages and add-staff pages for three staff categories — Admins, Nurses, and Finance — using Next.js App Router's Intercepting Routes and Parallel Routes features. The import form component already exists at `components/bulk-staff-import-dialog.tsx` and must be treated as a black box; your job is to wire it into the routing architecture described below.

EXISTING COMPONENT CONTRACT:
The existing `bulk-staff-import-dialog.tsx` must be refactored into a composable, mode-aware component before the page work begins. After refactoring it must satisfy this interface:

interface BulkStaffImportProps {
role: 'SCHOOL_ADMIN' | 'NURSE' | 'FINANCE';
mode: 'dialog' | 'page';
onSuccess?: () => void; // called after a successful import completes
onClose?: () => void; // called when the user actively dismisses (dialog mode only)
}

Rules the refactored component must follow:

- It must render correctly standalone (mode='page') or wrapped inside a modal shell (mode='dialog') without any internal routing knowledge — it must never call router.back() or router.push() itself.
- In mode='page': after a successful import, call onSuccess() if provided, then show an inline success state — do NOT redirect or close anything automatically.
- In mode='dialog': the modal shell is provided by the CALLER (the intercepted route), not by the component itself. The component just renders its form body.
- The role prop is passed directly to all API calls and IndexedDB namespace keys; the component never derives or infers role from the URL.
- The component is the single source of truth for import logic; the listing pages and add pages are pure routing/layout wrappers around it.

ROUTING ARCHITECTURE:
Use Next.js App Router intercepting routes (the (..) convention) combined with a @modal parallel route slot on each section's layout. The structure below must be followed exactly.

Desired URL behavior:

- Navigating to /admins/add directly (hard reload, shared link, opening in new tab) renders the FULL standalone page version.
- Navigating to /admins/add from within /admins (soft navigation, Link click) intercepts the route and renders the form inside a dialog overlaying the /admins listing page without unmounting it.
- Closing the dialog (Escape key, backdrop click, or an explicit close button) calls router.back(), returning to /admins with the listing page still mounted.
- The same pattern applies identically to /nurses → /nurses/add and /finance → /finance/add.

FILE STRUCTURE TO PRODUCE:
app/
(dashboard)/
layout.tsx ← existing root dashboard layout, DO NOT modify; shown for orientation only

    admins/
      layout.tsx                        ← renders {children} + {@modal} slot; imports and mounts the @modal parallel slot
      page.tsx                          ← /admins listing page (SCHOOL_ADMIN users)
      @modal/
        default.tsx                     ← null render (slot idle state, required by Next.js)
        (.)add/
          page.tsx                      ← intercepted /admins/add; renders BulkStaffImport in mode='dialog', onClose=router.back()
      add/
        page.tsx                        ← standalone /admins/add; renders BulkStaffImport in mode='page'

    nurses/
      layout.tsx
      page.tsx                          ← /nurses listing page (NURSE users)
      @modal/
        default.tsx
        (.)add/
          page.tsx                      ← intercepted /nurses/add; role='NURSE'
      add/
        page.tsx                        ← standalone /nurses/add; role='NURSE'

    finance/
      layout.tsx
      page.tsx                          ← /finance listing page (FINANCE users)
      @modal/
        default.tsx
        (.)add/
          page.tsx                      ← intercepted /finance/add; role='FINANCE'
      add/
        page.tsx                        ← standalone /finance/add; role='FINANCE'

INTERCEPTED ROUTE PAGES (dialog mode — e.g. app/admins/@modal/(.)add/page.tsx):

- Must be a Client Component ('use client').
- Renders a modal shell (backdrop + centered card) using whatever design system is already in the project (do not introduce a new modal library).
- Places <BulkStaffImport role='SCHOOL_ADMIN' mode='dialog' onClose={() => router.back()} /> inside the shell.
- Closes on backdrop click and Escape keydown, both calling router.back().
- Must NOT wrap the component in a <Suspense> boundary that would flash a skeleton on fast connections; only add Suspense if the component itself is dynamically imported with next/dynamic.

STANDALONE ADD PAGES (page mode — e.g. app/admins/add/page.tsx):

- Can be a Server Component wrapper that imports a thin Client Component child for the form.
- Renders a standard page layout (heading, breadcrumb back-link to /admins) above <BulkStaffImport role='SCHOOL_ADMIN' mode='page' />.
- onSuccess for the page version shows an inline success banner/state within the page — no redirect, no router.push().

LISTING PAGES (e.g. app/admins/page.tsx):
Each listing page renders two independent, paginated TanStack Table lists stacked vertically under a shared page heading and a primary action Link to ./add.

LIST 1 — Active Staff:

- Heading: e.g. "Active Admins"
- Data source: GET /api/v1/users?role=SCHOOL_ADMIN (tenant_id and school_id injected server-side by the auth layer)
- This list will be empty until invitations are accepted; render an appropriate empty state.
- Columns: first_name, last_name, email, phone_number, created_at. Actions per row: Edit, Deactivate (stubs — mark with TODO comments).

LIST 2 — Invitations:

- Heading: e.g. "Invited Admins"
- Data source: GET /api/v1/invitations?role=SCHOOL_ADMIN&status[]=pending&status[]=expired&status[]=revoked&status[]=invite_failed
- NEVER show status='accepted' invitations — those users already appear in List 1. The backend enforces this filter server-side; the frontend must also pass the explicit status array above rather than fetching all statuses and filtering client-side.
- Columns: first_name, last_name, email, status (badge), expires_at, created_at. Actions per row:
  - pending → Resend, Revoke (stubs — mark TODO)
  - expired → Resend (stub — mark TODO)
  - revoked → no actions
  - invite_failed → "Fix & Retry" button — opens the recovery import grid (the re-hydration flow defined in the bulk import spec). Stub with a TODO comment and console.log for now.
- Render an appropriate empty state when there are no non-accepted invitations.

Both lists manage their own loading, error, and empty states independently — a slow users query must not block the invitations list from rendering and vice versa. Use TanStack Query (useQuery) with separate query keys for each list.

ROLE MAPPING for listing page endpoints — pass these exact role values:

- /admins → role=SCHOOL_ADMIN
- /nurses → role=NURSE
- /finance → role=FINANCE

LAYOUT FILES (e.g. app/admins/layout.tsx):

- Must accept and render both the {children} slot and the {@modal} slot.
- Keep the layout thin — no data fetching, no providers — just slot composition.
- The layout is shared by /admins and /admins/add (standalone), so it must not add chrome that would look wrong on the full-page form view. If section-specific chrome is needed, put it in the listing page.tsx, not the layout.

ROLE MAPPING — pass these exact values:

- /admins and /admins/add → role='SCHOOL_ADMIN'
- /nurses and /nurses/add → role='NURSE'
- /finance and /finance/add → role='FINANCE'

BACKEND ENDPOINT SIGNATURES (to be implemented alongside the listing pages):
The following two endpoints must exist before the listing pages can render data. They are called from the frontend via TanStack Query; tenant_id and school_id are always resolved server-side from the session — never trusted from query params.

GET /api/v1/users
Query params: role (required: SCHOOL_ADMIN | NURSE | FINANCE), page, limit
Returns: paginated list of users matching role + tenant/school scope.

GET /api/v1/invitations
Query params: role (required), status[] (multi-value; backend must reject requests that include 'accepted' in this list — it is a disallowed filter value on this endpoint), page, limit
Returns: paginated list of invitations matching role + tenant/school scope, excluding accepted rows.
Backend query: WHERE role=$1 AND tenant_id=$2 AND school_id=$3 AND status != 'accepted' AND status = ANY($4)

The following partial index must exist in the database to keep this query fast as the invitations table grows:

CREATE INDEX idx_invitations_active_listing
ON invitations (tenant_id, school_id, role, status)
WHERE status != 'accepted';

CONSTRAINTS:

- Do not use the Pages Router, use App Router only.
- Do not use useRouter from next/router; use next/navigation throughout.
- Do not add react-modal, radix Dialog, or any new modal dependency; the modal shell is a plain div with Tailwind classes (or the project's existing utility classes).
- The @modal default.tsx files must exist and return null — Next.js requires them to avoid a build error when the slot has no active intercepted match.
- Do not fetch any data in this prompt's scope beyond the two listing endpoints defined above and what is already handled inside BulkStaffImport itself.
- TypeScript strict mode; no `any`.

OUT OF SCOPE FOR THIS PROMPT:

- The internal implementation of BulkStaffImport (IndexedDB, Web Worker, Asynq pipeline etc.) — covered by a separate spec.
- Authentication and authorization guards on these routes — handled by a separate coding agent.
- The Edit, Deactivate, Resend, Revoke, and Fix & Retry row actions — stubbed with TODOs only.
- The recovery import grid triggered by invite_failed rows — defined in the bulk import spec.
