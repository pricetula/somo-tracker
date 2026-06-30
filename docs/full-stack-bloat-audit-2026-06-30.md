# Full-Stack Bloat & Dead Code Audit

**Date:** 2026-06-30  
**Scope:** Next.js 16 Frontend (`frontend/`) + Go 1.26 Fiber Backend (`backend/`)  
**Auditor:** Automated static analysis (knip, deadcode, depcheck, staticcheck, go vet, grep)

---

## Stack Detection

| Attribute | Frontend | Backend |
|---|---|---|
| **Framework** | Next.js 16.2.9, App Router (`src/app/`) | Go 1.26, Fiber v2 |
| **Language** | TypeScript (strict mode) | Go (modules in `cmd/`+`internal/` layout) |
| **Package Manager** | pnpm ≥9.0.0 | Go modules |
| **State Management** | TanStack React Query v5 + URL/browser state | N/A |
| **Styling** | Tailwind CSS v4 + shadcn/ui Radix components | N/A |
| **Data Fetching** | TanStack React Query + native `fetch` via `client.ts` | pgx v5 (Postgres), asynq (Redis queues) |
| **ORM/DB** | N/A | Raw SQL + sqlc, golang-migrate |
| **Auth** | HttpOnly cookies + signed role cookies | Stytch B2B SSO + session tokens |
| **DI** | N/A | uber-go/fx |
| **Logging** | N/A | zap + slog |

---

## Prioritised Findings

### Quick Wins (High Confidence, XS/S Effort — Safe to Delete in One Pass)

| # | Layer | File(s) | Finding | Evidence | Recommendation | Confidence | Effort |
|---|---|---|---|---|---|---|---|
| 1 | FE | `src/components/error-boundary.tsx` | Unused ErrorBoundary component | knip: unused file; `grep -r "ErrorBoundary" src/` shows zero imports from any page/component | Delete file (77 lines). Pages rely on Next.js `error.tsx` | **High** | XS |
| 2 | FE | `src/components/ui/alert.tsx` | Unused shadcn alert component | knip: unused file; no imports from source code | Delete file | **High** | XS |
| 3 | FE | `src/components/ui/calendar.tsx` | Unused Calendar component (imports `react-day-picker`) | knip: unused file; no imports from source; `react-day-picker` also flagged as unused | Delete file + remove `react-day-picker` from deps | **High** | XS |
| 4 | FE | `src/components/ui/hover-card.tsx` | Unused HoverCard component | knip: unused file; no imports from source | Delete file | **High** | XS |
| 5 | FE | `src/components/ui/progress.tsx` | Unused Progress component | knip: unused file; no imports from source | Delete file | **High** | XS |
| 6 | FE | `src/components/layout/index.ts` | Dead barrel file | knip + `grep -r` shows no imports of this barrel; components are imported directly from their files | Delete file (2 re-exports) | **High** | XS |
| 7 | FE | `src/lib/redirect.ts` | Unused Open Redirect Guard | knip: unused; `grep -r "from.*redirect" src/` returns no results | Delete file | **High** | XS |
| 8 | FE | `src/features/home/components/home-page.tsx` | Unused trivial HomePage component | knip: unused file; never imported from any page | Delete file | **High** | XS |
| 9 | FE | `src/features/dashboard/components/support-staff-dashboard.tsx` | Dead SupportStaffDashboard component | knip: unused; not exported from barrel `dashboard/index.ts`; never imported | Delete file | **High** | XS |
| 10 | FE | `src/components/DocTooltipWrapper.tsx` | Dead wrapper (exports `FeatureHelp` but never imported) | knip: unused file; `grep -r "FeatureHelp" src/` returns only definition, no imports | Delete file | **High** | XS |
| 11 | FE | `src/features/staff/components/active-staff-table.tsx` | Dead ActiveStaffTable (254 lines, unused table) | knip: unused export; exported from barrel but never imported by any page | Delete file (254 lines). Role-specific tables (AdminsTable, TeachersTable, etc.) are used instead | **High** | S |
| 12 | FE | `src/features/staff/hooks/use-staff-users.ts` | Dead `useStaffUsers` hook | knip: unused export; barrel-re-exported but never imported | Delete file. `useStaffInvitations` is the only function used | **High** | XS |
| 13 | FE | `src/lib/api/auth.ts` re-exports `{ApiError, isApiError, getErrorMessage}` | Dead re-exports | knip flags these as unused exports; all consumers import directly from `@/lib/errors` | Remove the 3 re-exports at bottom of `auth.ts` | **High** | XS |
| 14 | FE | `src/lib/utils/breadcrumbs.ts:40,77` | `truncateId` and `resolveSegmentLabel` unused | knip: unused exports; `SmartBreadcrumb` only uses `buildBreadcrumbs` | Remove 2 unused helper functions | **High** | XS |
| 15 | FE | `src/lib/api/imports.ts` exports unused types | `StartImportRequest`, `ImportStaffRecord`, `ImportProgressEvent`, `FailedInvitation` re-exports | knip flags these as unused; they're re-exported from barrel but only used directly from the API file | Remove re-exports or confirm usage | **High** | XS |
| 16 | BE | `internal/auth/domain.go:348` | `ErrorToCode` function unreachable from main | `deadcode` tool: unreachable func; only used in test file | Delete or inline into test helper | **High** | XS |
| 17 | BE | `internal/slug/slug.go:41` | `slug.Validate` function unreachable from main | `deadcode` tool: unreachable func; only used in tests | Delete or keep only if external consumer planned | **High** | XS |

---

### Unused / Redundant Dependencies

| # | Layer | Finding | Evidence | Recommendation | Confidence | Effort |
|---|---|---|---|---|---|---|
| 18 | FE | `recharts` (~200KB) | `grep -r "recharts\|ResponsiveContainer\|BarChart\|LineChart\|PieChart" src/` returns zero results in source | Remove from `package.json`. No chart components exist | **High** | XS |
| 19 | FE | `input-otp` (~5KB) | `grep -r "input-otp\|InputOTP\|OTPInput" src/` returns zero results | Remove from `package.json` | **High** | XS |
| 20 | FE | `date-fns` (~30KB tree-shakeable) | knip + depcheck flag as unused. No imports found from any source file | Remove from `package.json` | **High** | XS |
| 21 | FE | `@radix-ui/react-dialog` | Redundant: shadcn v4 components (`dialog.tsx`, `popover.tsx`, `tooltip.tsx`) import from `radix-ui` meta-package. These individual packages are pulled in transitively | Remove 3 redundant `@radix-ui/react-*` deps | **High** | XS |
| 22 | FE | `@radix-ui/react-popover` | Same as above | Remove | **High** | XS |
| 23 | FE | `@radix-ui/react-tooltip` | Same as above | Remove | **High** | XS |
| 24 | FE | `@axe-core/react` (devDep) | knip: unused devDep. Never imported/configured | Remove | **High** | XS |
| 25 | FE | `lint-staged` (devDep) | knip: unused devDep. Husky hooks not configured to use lint-staged | Remove | **High** | XS |
| 26 | BE | `github.com/clipperhouse/uax29/v2` | Transitive dependency — check if anything actually uses it | Run `go mod why github.com/clipperhouse/uax29/v2` and remove if unnecessary | **Needs Confirmation** | XS |

---

### Structural Bloat / Duplication

| # | Layer | File(s) | Finding | Evidence | Recommendation | Confidence | Effort |
|---|---|---|---|---|---|---|---|
| 27 | FE | `src/features/staff-import/` + `src/features/student-import/` | Two separate import systems with duplicated patterns | Both have: `lib/validation.ts`, `hooks/use-*.ts`, file upload/parsing, progress tracking, indexeddb session recovery. Student import has 25+ components, staff import has 9 | Consider unifying under one `features/imports/` module | **High** | L |
| 28 | FE | `src/features/student-import/components/file-dropzone.tsx` + `src/workers/xlsx-parser.ts` | Duplicate CSV/XLSX parsing logic | Both `file-dropzone.tsx` and `xlsx-parser.ts` independently implement PapaParse + XLSX.read sheet parsing with near-identical code | Move parsing logic into a single shared utility | **High** | M |
| 29 | FE | `src/features/staff/` barrel exports | 5 table components, but `ActiveStaffTable` is dead — only role-specific tables (AdminsTable, TeachersTable, etc.) are actually used | `ActiveStaffTable` and `useStaffUsers` are dead; rest are used | Remove dead exports from barrel | **High** | XS |
| 30 | FE | `src/app/(dashboard)/page.tsx` + all `features/dashboard/` components | Requires `"use client"` on all 6 dashboard components, sending them all to client bundle | Each dashboard component starts with `"use client"` despite most being purely presentational (just rendering server data) | Extract data fetching from presentation; add Suspense boundaries | **Medium** | M |
| 31 | FE | `src/features/staff/components/status-toggle-cell.tsx` + `teacher-status-toggle-cell.tsx` | Two near-identical toggle cell components | Both render a checkbox/switch toggle; `teacher-status-toggle-cell.tsx` appears to have slightly different props | Merge into one configurable toggle component | **Medium** | S |
| 32 | BE | `internal/members/handler.go` + `internal/teachers/handler.go` | Duplicate handler patterns | Both implement identical List/ToggleActive patterns with independent repository interfaces and nearly identical domain errors (same sentinels with different prefix text) | Consolidate into member toggle / list that accepts role filter | **Medium** | M |
| 33 | BE | `internal/imports/domain.go:255` + `internal/auth/domain.go:279-280` | `GetTenantStytchOrgID` defined in 2 different repository interfaces | Duplicated interface method in both `imports.Repository` and `auth.SchoolCreator` interface | If one adapter pattern works, remove the duplicate | **Medium** | S |
| 34 | BE | `internal/middleware/security.go` + Next.js `next.config.ts` | Duplicate CSP headers | Backend sets CSP for `/api/` routes; Next.js sets CSP for all routes via `next.config.ts` headers. These overlap and may conflict | Only set CSP in one place (Next.js owns the full page) | **Medium** | S |
| 35 | BE | `internal/students/` + `internal/imports/` | Students import routes registered in `imports/handler.go`, but students also has its own handler | `/api/v1/students` (list) in imports handler, `/api/v1/students/list` in students handler — two different routes for listing students | Determine which is canonical, deprecate the other | **High** | S |

---

### `"use client"` Directive Concerns

| # | File | Issue | Impact |
|---|---|---|---|
| 36 | `src/features/dashboard/components/admin-dashboard.tsx` | `"use client"` but only renders static content from props | Forces ~10KB+ of React client runtime into bundle unnecessarily |
| 37 | `src/features/dashboard/components/teacher-dashboard.tsx` | `"use client"` but only renders static content from props | Same |
| 38 | `src/features/dashboard/components/nurse-dashboard.tsx` | `"use client"` but only renders static content from props | Same |
| 39 | `src/features/dashboard/components/finance-dashboard.tsx` | `"use client"` but only renders static content from props | Same |
| 40 | `src/features/dashboard/components/system-admin-dashboard.tsx` | `"use client"` but only renders static content from props | Same |
| 41 | `src/features/dashboard/components/school-admin-dashboard.tsx` | `"use client"` but only renders static content from props | Same |
| 42 | `src/features/staff/components/teachers-table.tsx` | `"use client"` — justified (uses interactive toggle) | OK — interactivity needed |
| 43 | `src/features/staff/components/admins-table.tsx` | `"use client"` — justified (uses interactive toggle) | OK — interactivity needed |

---

### Needs Confirmation

| # | Layer | File(s) | Finding | How to Verify |
|---|---|---|---|---|
| NC1 | FE | `src/app/(dashboard)/settings/page.tsx` | Page exists but may have no navigation links pointing to it | Check `nav-main.tsx` and `app-sidebar.tsx` for a "Settings" nav item |
| NC2 | FE | `scripts/audit-docs.js` | Script file exists, not imported by any page/config | Check if any npm script or CI pipeline calls `audit:docs` |
| NC3 | FE | `src/features/staff-import/lib/validation.ts` exports `isValidPhoneNumber`, `parsePhoneNumber` | knip flags as unused exports. May be used by internal components, not through barrel | Check if any staff-import component imports these directly (not through barrel) |
| NC4 | FE | `src/lib/db.ts` | Exists but not flagged by knip | Check for direct imports in any source file |
| NC5 | BE | `github.com/clipperhouse/uax29/v2` | Pulled in as transitive dep | Run `go mod why github.com/clipperhouse/uax29/v2` |
| NC6 | FE | `src/app/logout/page.tsx` | Page exists — confirm it's referenced from `client.ts` 401 handler (`window.location.href = "/logout"`) | Already confirmed — used by `client.ts` |

---

## Total Estimated Savings

### Bundle Size (Frontend)
- `recharts` removal: ~200KB (unminified, ~80KB gzipped)
- `date-fns` removal: ~30KB (tree-shakeable but unused)
- `react-day-picker` removal: ~15KB
- `input-otp` removal: ~5KB
- 6 dashboard components avoiding `"use client"`: ~60KB client bundle reduction
- **Total est. savings: ~250KB client bundle (~100KB gzipped)**

### Code Volume (Frontend)
- 10 files safe to delete: ~1,000 lines
- Dead re-exports/tiny helpers: ~60 lines
- ActiveStaffTable removal: 254 lines

### Code Volume (Backend)
- 2 unreachable functions: ~40 lines
- Potential teacher/members consolidation: ~300 lines saved
- Potential import merge: ~500 lines saved

### Dependency Count
- 7 npm packages removable (3 direct + 4 dev)
- 0 Go modules directly removable (1 transitive candidate)

---

## Appendix: Commands Used

```bash
# Frontend
npx knip --production
npx depcheck
grep -r "import.*from.*" src/ --include="*.ts" --include="*.tsx"

# Backend
deadcode ./...
go vet ./...
staticcheck ./...
go mod tidy -v
go mod graph | wc -l
```
