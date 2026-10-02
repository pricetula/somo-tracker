# 01 — Rooms CRUD

## Overview
Implement full CRUD for rooms to enable timetable room assignment. Rooms are referenced by `class_timetable_slots` but currently have no management UI or API.

---

## Backend Changes

### 1. SQLC Queries
**File:** `backend/db/queries/rooms.sql` (new)

```sql
-- name: CreateRoom :one
INSERT INTO rooms (school_id, name, room_type, capacity, metadata)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetRoom :one
SELECT * FROM rooms WHERE id = $1 AND school_id = $2;

-- name: ListRooms :many
SELECT * FROM rooms
WHERE school_id = $1
ORDER BY name
LIMIT $2 OFFSET $3;

-- name: CountRooms :one
SELECT COUNT(*) FROM rooms WHERE school_id = $1;

-- name: UpdateRoom :one
UPDATE rooms
SET name = $2, room_type = $3, capacity = $4, metadata = $5, updated_at = NOW()
WHERE id = $1 AND school_id = $6
RETURNING *;

-- name: DeleteRoom :exec
DELETE FROM rooms WHERE id = $1 AND school_id = $2;
```

Run `make sqlc-gen` after creating.

### 2. Service
**File:** `backend/internal/services/rooms_service.go` (new)

```go
type Room struct {
    ID        uuid.UUID
    SchoolID  uuid.UUID
    Name      string
    RoomType  string
    Capacity  int
    Metadata  json.RawMessage
    CreatedAt time.Time
    UpdatedAt time.Time
}

type ListRoomsParams struct {
    SchoolID uuid.UUID
    Page     int
    Limit    int
}

type RoomService interface {
    CreateRoom(ctx context.Context, schoolID uuid.UUID, req CreateRoomRequest) (*Room, error)
    GetRoom(ctx context.Context, id uuid.UUID) (*Room, error)
    ListRooms(ctx context.Context, params ListRoomsParams) ([]*Room, int, error)
    UpdateRoom(ctx context.Context, id uuid.UUID, req UpdateRoomRequest) (*Room, error)
    DeleteRoom(ctx context.Context, id uuid.UUID) error
}
```

### 3. Handler
**File:** `backend/internal/api/rooms_handler.go` (new)

```go
// @Summary List rooms
// @Tags Rooms
// @Produce json
// @Param page query int false "Page number"
// @Param limit query int false "Page size"
// @Success 200 {object} map[string]interface{}
// @Router /api/school/rooms [get]
func (h *RoomsHandler) ListRooms(c fiber.Ctx) error { ... }

// @Summary Create room
// @Tags Rooms
// @Accept json
// @Produce json
// @Param body body CreateRoomRequest true "Room payload"
// @Success 201 {object} map[string]interface{}
// @Router /api/school/rooms [post]
func (h *RoomsHandler) CreateRoom(c fiber.Ctx) error { ... }

// @Summary Get room
// @Tags Rooms
// @Produce json
// @Param id path string true "Room ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/school/rooms/{id} [get]
func (h *RoomsHandler) GetRoom(c fiber.Ctx) error { ... }

// @Summary Update room
// @Tags Rooms
// @Accept json
// @Produce json
// @Param id path string true "Room ID"
// @Param body body UpdateRoomRequest true "Room payload"
// @Success 200 {object} map[string]interface{}
// @Router /api/school/rooms/{id} [patch]
func (h *RoomsHandler) UpdateRoom(c fiber.Ctx) error { ... }

// @Summary Delete room
// @Tags Rooms
// @Produce json
// @Param id path string true "Room ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/school/rooms/{id} [delete]
func (h *RoomsHandler) DeleteRoom(c fiber.Ctx) error { ... }
```

### 4. Router Registration
**File:** `backend/internal/api/router.go`

```go
// In RegisterRoutes, add to protected group:
protected.Get("/school/rooms", r.Rooms.ListRooms)
protected.Post("/school/rooms", r.Rooms.CreateRoom)
protected.Get("/school/rooms/:id", r.Rooms.GetRoom)
protected.Patch("/school/rooms/:id", r.Rooms.UpdateRoom)
protected.Delete("/school/rooms/:id", r.Rooms.DeleteRoom)
```

### 5. Migration Test
**File:** `backend/internal/database/migrator_integration_test.go`

Add `TestMigrator_RoomsTable` verifying:
- `rooms` table exists with correct columns
- `room_type` enum constraint
- Index on `school_id`

---

## Frontend Changes

### 1. Feature Module
```
src/features/rooms/
├── components/
│   ├── rooms-table.tsx
│   └── room-form.tsx
├── hooks/
│   └── use-rooms.ts
├── services/
│   └── api.ts
├── types/
│   └── room.ts
└── index.ts
```

### 2. Types (`types/room.ts`)
```ts
export type RoomType = "STANDARD" | "SCIENCE_LAB" | "COMPUTER_LAB" | "GYM";

export interface Room {
    id: string;
    schoolId: string;
    name: string;
    roomType: RoomType;
    capacity: number;
    metadata: Record<string, unknown>;
    createdAt: string;
    updatedAt: string;
}

export interface ListRoomsParams {
    page?: number;
    limit?: number;
}

export interface ListRoomsResponse {
    items: Room[];
    total: number;
    page: number;
    limit: number;
}

export interface CreateRoomRequest {
    name: string;
    roomType: RoomType;
    capacity: number;
    metadata?: Record<string, unknown>;
}

export interface UpdateRoomRequest {
    name?: string;
    roomType?: RoomType;
    capacity?: number;
    metadata?: Record<string, unknown>;
}
```

### 3. Page
**File:** `src/app/(dashboard)/settings/rooms/page.tsx` (new)

```tsx
import { RoomsTable } from "@/features/rooms";

export default function RoomsPage() {
    return (
        <div className="space-y-6">
            <h1 className="text-2xl font-semibold">Rooms</h1>
            <RoomsTable />
        </div>
    );
}
```

### 4. Navigation
Update `src/components/layout/nav-main.tsx`:
- Add "Rooms" under Settings submenu: `{ title: "Rooms", url: "/settings/rooms" }`

---

## Database Migration
**File:** `backend/db/migrations/XXXXXX_create_rooms.up.sql`

```sql
CREATE TYPE room_type AS ENUM ('STANDARD', 'SCIENCE_LAB', 'COMPUTER_LAB', 'GYM');

CREATE TABLE rooms (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    school_id UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    room_type room_type NOT NULL DEFAULT 'STANDARD',
    capacity INT NOT NULL DEFAULT 30,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (school_id, name)
);

CREATE INDEX idx_rooms_school_id ON rooms(school_id);
```

---

## Acceptance Criteria
- [ ] Can create room with name, type, capacity
- [ ] Can list rooms with pagination
- [ ] Can view single room
- [ ] Can update room details
- [ ] Can delete room (soft check: not referenced by timetable slots)
- [ ] Room types: STANDARD, SCIENCE_LAB, COMPUTER_LAB, GYM
- [ ] RLS: tenant isolation via `app.current_tenant_id`
- [ ] Frontend: DataTable with search, filter by room_type
- [ ] Form validation: name required, capacity > 0

---

## Dependencies
- None (foundational)

---

## Estimated Effort
- Backend: ~4 hours
- Frontend: ~3 hours
- **Total: ~7 hours**