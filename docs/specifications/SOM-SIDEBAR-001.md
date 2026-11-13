---
id: SOM-SIDEBAR-001
title: App Sidebar — Nav Links, School Switcher, User Menu
status: approved
author: agent
date: 2026-11-13
updated: 2026-11-13
fr_ids: [FE-013, FE-007, FE-011, FE-021]
nf_ids: []
api_contract: yes
migrations: no
generated_files: []
---

## 1. Summary

The app shell sidebar (shadcn `sidebar-07`, FE-013) contains three sections:

1. **`NavMain`** — primary navigation links, role-gated, "Coming soon" for pages not yet implemented.
2. **`SchoolSwitcher`** — replaces the shadcn `TeamSwitcher` demo; lists schools in the tenant and switches the active school.
3. **`NavUser`** — user dropdown with real Logout; Profile/Account marked "Coming soon".

No dummy data is passed through props from `app-sidebar.tsx`. Each section owns its data or receives it via props from the root loader.

## 2. Nav Link Inventory (MVP)

`NavProjects` is dropped. `NavMain` is the only nav section. Links to pages not yet implemented render **"Coming soon"** (disabled state, tooltip, no navigation). Links outside the user's role set are **hidden entirely**, not shown as disabled.

| Link | Icon | Visibility | Status |
|------|------|------------|--------|
| Dashboard | `LayoutDashboard` | All roles | Coming soon |
| Students | `User` | All roles | Coming soon |
| Classes | `LayoutGrid` | Headteacher, Teacher | Coming soon |
| Learning Areas | `BookOpen` | Headteacher, Teacher | Coming soon |
| Timetable | `Calendar` | Headteacher, Teacher | Coming soon |
| Attendance / Roll Call | `CheckCircle` | Headteacher, Teacher | Coming soon |
| Fees / Payments | `CreditCard` | Headteacher, Accountant | Coming soon |
| Guardians | `UserCircle` | Headteacher, Teacher | Coming soon |
| Staff / Teachers | `Users` | Headteacher | Coming soon |
| Reports | `BarChart` | Headteacher | Coming soon |
| Logout | `LogOut` | All roles | Real (POST /api/v1/auth/logout) |
| Profile / Account | `User` | All roles | Coming soon |

## 3. Role Matrix Hook (`useSidebarNav`)

A pure, testable hook that takes `/me` data and returns the visible link set.

| Role | Visible links |
|------|---------------|
| Headteacher | All links |
| Teacher | Dashboard, Students, Classes, Learning Areas, Timetable, Attendance, Guardians, Profile, Logout |
| Guardian | Dashboard, Students, Profile, Logout |
| Accountant | Dashboard, Students, Fees / Payments, Profile, Logout |

Attempting to navigate to a page outside the user's role set renders the existing `/unauthorized` route.

## 4. Data Flow — `/me` Without Flicker

Backend endpoint `GET /api/v1/me` returns the authenticated user's profile. Frontend flow:

1. **Backend:** `GET /api/v1/me` returns `full_name`, `email`, `avatar_url`, `role`, `active_school { id, name }`, `schools [ { id, name } ]`.
2. **Server function:** `getMe` (`createServerFn`) reads the Cookie header and forwards it to `/api/v1/me`. Runs on TanStack Start server.
3. **Root route loader:** `routes/__root.tsx` loader calls `getMe()` during SSR; data is embedded in HTML.
4. **Client hook:** `useMe()` reads loader data via context; no client fetch on initial load (no skeleton, no flicker).
5. **Client-side navigation:** `beforeLoad` can invoke `getMe` as a server function RPC, or TanStack Query revalidates.
6. **Sidebar components** (`AppSidebar`, `NavMain`, `SchoolSwitcher`, `NavUser`) receive data via props — **never fetch** (FE-021).

## 5. School Switcher

Replaces `TeamSwitcher` in the sidebar header.

- `GET /api/v1/schools` — lists schools in the tenant (`{ id, name }[]`).
- `PUT`/`PATCH /api/v1/me/active-school` — switches active school membership.
- On switch: hard refresh the page.
- Sidebar MVP location; may move to a Settings page later.

## 6. `/me` Endpoint Payload

| Field | Type | Notes |
|-------|------|-------|
| `full_name` | string | User display name |
| `email` | string | User email |
| `avatar_url` | string \| null | Avatar image URL |
| `role` | string | User role (`headteacher`, `teacher`, `guardian`, `accountant`) |
| `active_school` | object \| null | `{ id, name }` — currently active school |
| `schools` | array | `[{ id, name }]` — all schools in tenant |

## 7. User Menu (Sidebar Footer)

- Logout — real endpoint (`POST /api/v1/auth/logout`).
- Profile / Account — Coming soon.

## 8. Implementation Notes

- `app-sidebar.tsx` renders `NavMain` + `SchoolSwitcher` + `NavUser` only; no dummy data passed via props.
- `NavMain` owns its link list internally (already refactored in prior cleanup).
- `useSidebarNav()` hook is unit-tested per role.
- Backend `/me`, `GET /api/v1/schools`, and switch active school endpoints are new API contract changes; OpenAPI spec must be updated and frontend types regenerated (`pnpm generate:api`).

## 9. Links

- [FE-013](../requirements/frontend.md) — app shell shadcn `sidebar-07`
- [FE-007](../requirements/frontend.md) — no school ID header; backend owns tenant context
- [FE-011](../requirements/frontend.md) — `useActiveSchool()` reads active school via API
- [FE-021](../requirements/frontend.md) — shared composites never fetch
