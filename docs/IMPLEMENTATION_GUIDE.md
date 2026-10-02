# Implementation Guide — Somotracker Frontend Patterns

> Extracted from existing codebase. Use as reference for all new features.
> This documents HOW we implement, not WHAT to implement.

---

## 1. Route Architecture

### Route Groups
```
src/app/
├── (auth)/           # Public auth pages (login, register, logout)
├── (dashboard)/      # Protected dashboard pages
│   ├── @modal/       # Parallel intercepted routes for side sheets/dialogs
│   ├── attendance/
│   ├── classes/
│   ├── settings/
│   └── ...
```

### Layout Structure
**`src/app/(dashboard)/layout.tsx`**
```tsx
export default function DashboardLayout({
    children,
    modal,
}: {
    children: React.ReactNode;
    modal?: React.ReactNode;
}) {
    return (
        <DashboardAuthLayout>
            <AppLayout>
                {children}
                {modal}
            </AppLayout>
        </DashboardAuthLayout>
    );
}
```
- `children` = main page content (full page)
- `modal` = `@modal` parallel route slot (renders Sheet/Dialog overlay)

### @modal Parallel Route Convention
```
@modal/
├── default.tsx       # Returns null (required for unmatched slots)
├── (.)feature/       # Intercepts `feature/` route
│   ├── [id]/page.tsx # Detail view → Sheet (side panel)
│   └── add/page.tsx  # Create view → Dialog (centered modal)
```

---

## 2. Detail View Patterns

### Pattern A: Side Sheet (Right Panel) — Most Common
**Use for:** Viewing/editing a single record from a list

**Route:** `@modal/(.)feature/[id]/page.tsx`
```tsx
"use client";

import { useParams, useRouter } from "next/navigation";
import { Sheet, SheetContent, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import { FeatureDetail } from "@/features/feature";

export default function FeatureDetailSheet() {
    const params = useParams();
    const id = params?.id as string;
    const router = useRouter();

    const handleOpenChange = (open: boolean) => {
        if (!open) {
            router.back(); // Navigate back on close
        }
    };

    return (
        <Sheet open onOpenChange={handleOpenChange}>
            <SheetContent side="right" className="w-full sm:max-w-md">
                <SheetHeader>
                    <SheetTitle>Feature Name</SheetTitle>
                </SheetHeader>
                <FeatureDetail id={id} />
            </SheetContent>
        </Sheet>
    );
}
```

**Link from list:** Regular `<Link href={`/feature/${id}`}>` — Next.js intercepts to @modal

**Component:** `FeatureDetail` receives `id` prop, fetches own data

---

### Pattern B: Full Page Detail
**Use for:** Complex detail views with sub-navigation, multiple tabs, or when URL should be shareable/bookmarkable

**Route:** `feature/[id]/page.tsx` (regular page, NOT in @modal)
```tsx
"use client";

import { useParams } from "next/navigation";
import { FeatureDetail } from "@/features/feature";

export default function FeatureDetailPage() {
    const params = useParams();
    const id = params.id as string;

    return <FeatureDetail id={id} />;
}
```

**Examples:** `/classes/[id]`, `/curriculum/[id]/[subjectId]/[topicId]`

---

### Pattern C: Dialog (Centered Modal) — For Create/Import
**Use for:** Create flows, import wizards, quick actions

**Route:** `@modal/(.)feature/add/page.tsx`
```tsx
"use client";

import { useRouter } from "next/navigation";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { FeatureCreateForm } from "@/features/feature";

export default function FeatureAddModalPage() {
    const router = useRouter();

    const handleOpenChange = (open: boolean) => {
        if (!open) {
            router.back();
        }
    };

    return (
        <Dialog open onOpenChange={handleOpenChange}>
            <DialogContent className="max-h-[85vh] overflow-y-auto md:max-w-2xl">
                <DialogHeader>
                    <DialogTitle>Add Feature</DialogTitle>
                </DialogHeader>
                <FeatureCreateForm onSuccess={() => router.push("/feature")} />
            </DialogContent>
        </Dialog>
    );
}
```

**Examples:** Student import, Teacher/Finance/Guardian/Admin invites

---

## 3. List Page Pattern

**Route:** `feature/page.tsx`
```tsx
"use client";

import { FeatureTable } from "@/features/feature";

export default function FeaturePage() {
    return (
        <div className="space-y-4 p-6">
            <h1 className="text-2xl font-semibold">Feature Name</h1>
            <FeatureTable />
        </div>
    );
}
```

**No headers, no back buttons** — app shell handles navigation

---

## 4. Data Table Component Pattern

**Location:** `src/features/feature/components/feature-table.tsx`

```tsx
"use client";
import { DataTable } from "@/components/shared/data-table";
import { listFeatures } from "../services/api";
import { useFeatures } from "../hooks/use-features";
import type { Feature, ListFeaturesParams } from "../types/feature";

function listWithFilters(params: ListFeaturesParams) {
    return listFeatures(params);
}

export function FeatureTable() {
    const { data: relatedData = [] } = useRelatedData(); // for filter options

    const filterGroups = useMemo(() => [...], [relatedData]);
    const columns = useMemo(() => [...], []);

    return (
        <DataTable<
            Feature,
            ListFeaturesParams,
            { items: Feature[]; total: number; page: number; limit: number }
        >
            queryKey={["feature", "list"]}
            queryFn={listWithFilters}
            getRowId={(row) => row.id}
            columns={columns}
            isSearchable
            searchPlaceholder="Search..."
            filterGroups={filterGroups}
            pageSize={50}
            height={500}
            addHref="/feature/add"        // Opens @modal/(.)feature/add
            // deleteFn={async (ids) => await deleteFeatures(ids)} // Optional
        />
    );
}
```

### Column Definition Pattern
```tsx
const columns = useMemo<DataTableColumn<Feature>[]>(
    () => [
        {
            id: "name",
            header: "Name",
            cell: (row: Feature) => (
                <Link
                    href={`/feature/${row.id}`}  // Opens side sheet
                    className="underline underline-offset-4 hover:no-underline"
                >
                    {row.name}
                </Link>
            ),
            width: "2fr",
        },
        {
            id: "actions",
            header: "",
            cell: (row: Feature) => (
                <DropdownMenu>
                    <DropdownMenuTrigger
                        render={<Button variant="ghost" size="icon"><MoreVertical className="size-4" /></Button>}
                    />
                    <DropdownMenuContent align="end">
                        <DropdownMenuItem
                            render={<Link href={`/feature/${row.id}`}>Edit</Link>}
                        />
                        <DropdownMenuItem
                            onSelect={() => handleDelete([row.id])}
                            className="text-destructive"
                        >
                            Delete
                        </DropdownMenuItem>
                    </DropdownMenuContent>
                </DropdownMenu>
            ),
            width: "50px",
            align: "right" as const,
        },
    ],
    []
);
```

---

## 5. Data Fetching Patterns

### API Service (`services/api.ts`)
```ts
import { api } from "@/lib/api/client";
import type { Feature, ListFeaturesParams, ListFeaturesResponse } from "../types/feature";

export async function listFeatures(params: ListFeaturesParams = {}): Promise<ListFeaturesResponse> {
    const searchParams = new URLSearchParams();
    Object.entries(params).forEach(([key, value]) => {
        if (value === undefined || value === null || value === "") return;
        if (Array.isArray(value)) {
            value.forEach((v) => searchParams.append(key, v));
        } else {
            searchParams.set(key, String(value));
        }
    });
    return api.get<ListFeaturesResponse>(`/api/feature?${searchParams.toString()}`);
}

export async function getFeature(id: string): Promise<Feature> {
    return api.get<Feature>(`/api/feature/${id}`);
}

export async function createFeature(data: CreateFeatureRequest): Promise<Feature> {
    return api.post<Feature>("/api/feature", data);
}

export async function updateFeature(id: string, data: UpdateFeatureRequest): Promise<Feature> {
    return api.patch<Feature>(`/api/feature/${id}`, data);
}

export async function deleteFeature(ids: string[]): Promise<void> {
    await api.delete(`/api/feature`, { ids });
}
```

### React Query Hooks (`hooks/use-features.ts`)
```ts
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { listFeatures, getFeature, createFeature, updateFeature, deleteFeature } from "../services/api";
import type { Feature, ListFeaturesParams } from "../types/feature";
import { getErrorMessage } from "@/lib/errors";
import { toast } from "sonner";

export const featureKeys = {
    list: (params: ListFeaturesParams) => ["feature", "list", params] as const,
    detail: (id: string) => ["feature", "detail", id] as const,
};

export function useFeatures(params: ListFeaturesParams = {}) {
    return useQuery({
        queryKey: featureKeys.list(params),
        queryFn: () => listFeatures(params),
        staleTime: 30_000,
    });
}

export function useFeature(id: string) {
    return useQuery({
        queryKey: featureKeys.detail(id),
        queryFn: () => getFeature(id),
        enabled: !!id,
    });
}

export function useCreateFeature() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: createFeature,
        onSuccess: (data) => {
            toast.success("Feature created");
            queryClient.invalidateQueries({ queryKey: ["feature", "list"] });
        },
        onError: (err) => toast.error(getErrorMessage(err)),
    });
}

// Similar for update/delete with optimistic updates
```

---

## 6. Form Patterns

### Compound Form (shadcn/ui)
**Location:** `components/feature-form.tsx`
```tsx
"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Form, FormField, FormItem, FormLabel, FormControl, FormMessage } from "@/components/ui/form";
import { useCreateFeature } from "../hooks/use-features";

const schema = z.object({
    name: z.string().min(1, "Name is required"),
    description: z.string().optional(),
});

type FormValues = z.infer<typeof schema>;

interface FeatureFormProps {
    onSuccess: () => void;
    defaultValues?: Partial<FormValues>;
}

export function FeatureForm({ onSuccess, defaultValues }: FeatureFormProps) {
    const { mutate: createFeature, isPending } = useCreateFeature();
    const form = useForm<FormValues>({
        resolver: zodResolver(schema),
        defaultValues: { name: "", description: "", ...defaultValues },
    });

    const onSubmit = form.handleSubmit((values) => {
        createFeature(values, { onSuccess });
    });

    return (
        <Form {...form}>
            <form onSubmit={onSubmit} className="space-y-4">
                <FormField
                    control={form.control}
                    name="name"
                    render={({ field }) => (
                        <FormItem>
                            <FormLabel>Name</FormLabel>
                            <FormControl>
                                <Input {...field} placeholder="Enter name" />
                            </FormControl>
                            <FormMessage />
                        </FormItem>
                    )}
                />
                <FormField
                    control={form.control}
                    name="description"
                    render={({ field }) => (
                        <FormItem>
                            <FormLabel>Description</FormLabel>
                            <FormControl>
                                <Input {...field} placeholder="Optional description" />
                            </FormControl>
                            <FormMessage />
                        </FormItem>
                    )}
                />
                <Button type="submit" disabled={isPending} className="w-full">
                    {isPending ? "Creating..." : "Create"}
                </Button>
            </form>
        </Form>
    );
}
```

### Form in Side Sheet (Edit)
- Same form component, pass `defaultValues` from fetched data
- `onSuccess` calls `router.push("/feature")` or invalidates queries

---

## 7. Type Definitions

**Location:** `src/features/feature/types/feature.ts`
```ts
export interface Feature {
    id: string;
    name: string;
    description: string;
    createdAt: string;
    updatedAt: string;
}

export interface ListFeaturesParams {
    page?: number;
    limit?: number;
    search?: string;
    filters?: Record<string, string | string[]>;
}

export interface ListFeaturesResponse {
    items: Feature[];
    total: number;
    page: number;
    limit: number;
}

export interface CreateFeatureRequest {
    name: string;
    description?: string;
}

export interface UpdateFeatureRequest {
    name?: string;
    description?: string;
}
```

---

## 8. Feature Index Export

**Location:** `src/features/feature/index.ts`
```ts
export * from "./hooks/use-features";
export { FeatureTable } from "./components/feature-table";
export { FeatureForm } from "./components/feature-form";
export { FeatureDetail } from "./components/feature-detail";
```

---

## 9. Navigation Integration

**File:** `src/components/layout/nav-main.tsx`
```tsx
{
    title: "Feature",
    url: "/feature",
    icon: <Icon className="size-4" />,
    items: [
        { title: "List", url: "/feature" },
        { title: "Settings", url: "/settings/feature" },
    ],
},
```

---

## 10. Backend Contract (Reference)

### Handler → Service → SQLC
```
Handler (internal/api/*_handler.go)
  → validates request, extracts locals
  → calls Service method
  → returns JSON response

Service (internal/services/*_service.go)
  → business logic
  → calls *sqlc.Queries methods
  → returns domain types

SQLC (db/queries/*.sql)
  → raw SQL with -- name: annotations
  → generated type-safe Go methods
```

### Error Response Format
```json
{
  "code": "snake_case_error_code",
  "message": "human readable message",
  "errors": { "field_name": ["validation message"] }
}
```

### Standard Pagination
- Default page: 1, default limit: 50, max limit: 200
- Response: `{ items: [], total: 0, page: 1, limit: 50 }`

---

## 11. Decision Matrix: Which Pattern?

| Scenario | Route Pattern | UI Component |
|----------|---------------|--------------|
| View record from list | `@modal/(.)feature/[id]/page.tsx` | Sheet (right) |
| Edit record from list | `@modal/(.)feature/[id]/page.tsx` | Sheet with Form |
| Complex detail (tabs, sub-pages) | `feature/[id]/page.tsx` | Full page |
| Create new record | `@modal/(.)feature/add/page.tsx` | Dialog |
| Import/bulk create | `@modal/(.)feature/add/page.tsx` | Dialog (multi-step) |
| Simple list only | `feature/page.tsx` | DataTable |
| Settings/configuration | `settings/feature/page.tsx` | Full page |

---

## 12. Component File Checklist for New Feature

```
src/features/feature/
├── components/
│   ├── feature-table.tsx      # DataTable with columns, filters
│   ├── feature-form.tsx       # Compound form (create/edit)
│   └── feature-detail.tsx     # Read-only detail view
├── hooks/
│   └── use-features.ts        # React Query hooks
├── services/
│   └── api.ts                 # API client calls
├── types/
│   └── feature.ts             # TypeScript interfaces
└── index.ts                   # Public exports
```

---

## 13. Common Gotchas

### ❌ Don't
- Use `asChild` on shadcn components (use `render={<Component />}` instead)
- Call `setState` in `useEffect` (derive with `useMemo` or combine in fetch)
- Hardcode `/backend` in API calls (use `api.get('/api/...')`)
- Add back buttons or headers in pages (app shell owns navigation)
- Use custom colors (use semantic CSS variables)

### ✅ Do
- Use `toEqual` for object comparisons in tests
- Provide context types for `useMutation` optimistic updates
- Wrap `deleteFn` to return `void` for DataTable
- Use `router.back()` in Sheet/Dialog `onOpenChange`
- Return `null` from `@modal/default.tsx`

---

## 14. Testing Patterns

### Unit Test (Vitest + React Testing Library)
```tsx
import { render, screen } from "@testing-library/react";
import { FeatureTable } from "@/features/feature";

vi.mock("@/features/feature/hooks/use-features", () => ({
    useFeatures: () => ({
        data: { items: [], total: 0, page: 1, limit: 50 },
        isLoading: false,
        isError: false,
    }),
}));

test("renders feature table", () => {
    render(<FeatureTable />);
    expect(screen.getByText("Feature Name")).toBeInTheDocument();
});
```

### Integration Test (Backend)
```go
//go:build integration

func TestFeature_CRUD(t *testing.T) {
    // 1. Setup test DB
    // 2. Create via service
    // 3. List and verify
    // 4. Update and verify
    // 5. Delete and verify
}
```

---

## 15. Adding a New Feature — Step by Step

1. **Backend first:**
   - Create migration → run `make sqlc-gen`
   - Add SQLC queries in `db/queries/`
   - Implement Service interface + implementation
   - Implement Handler with swaggo annotations
   - Register routes in `router.go`
   - Run `make generate-swagger` → `make generate-api-types`

2. **Frontend types:**
   - Create `src/features/feature/types/feature.ts`

3. **Frontend API:**
   - Create `src/features/feature/services/api.ts`

4. **Frontend hooks:**
   - Create `src/features/feature/hooks/use-features.ts`

5. **Frontend components:**
   - `feature-table.tsx` (DataTable)
   - `feature-form.tsx` (Form)
   - `feature-detail.tsx` (Detail)

6. **Frontend pages:**
   - `src/app/(dashboard)/feature/page.tsx` (list)
   - `src/app/(dashboard)/@modal/(.)feature/[id]/page.tsx` (detail sheet)
   - `src/app/(dashboard)/@modal/(.)feature/add/page.tsx` (create dialog)

7. **Navigation:**
   - Add to `nav-main.tsx`

8. **Verify:**
   - `pnpm build` (frontend)
   - `make build-backend` (backend)
   - `pnpm lint` / `golangci-lint run`