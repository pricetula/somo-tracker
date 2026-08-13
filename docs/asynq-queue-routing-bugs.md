# Asynq Queue-Routing Bugs

| Field       | Value                                                        |
| ----------- | ------------------------------------------------------------ |
| Status      | **OPEN** — documented, not fixed (behavioral change needs decision) |
| Severity    | High                                                        |
| Discovered  | 2026-06 (while centralizing Asynq construction into `backend/internal/database/asyncq.go`) |
| Affected    | `backend/internal/attendance`, `backend/internal/assessments`, `backend/internal/imports`, `backend/internal/cohortpositions` |
| Depends on  | asynq `v0.26.0`                                             |

---

## Summary

Three queue-routing defects exist in the Asynq background-job layer. The
centralization refactor (single `database.NewAsynqClient` / `NewAsynqServer` /
`NewAsynqScheduler`) fixed the *construction* duplication but deliberately left
the *routing* behavior untouched. This document records the routing bugs so the
fix is a deliberate, reviewed change rather than incidental.

1. **BUG 1 — Two servers consume the `summaries` queue with disjoint muxes**
   (attendance + assessments). Cross-type tasks fail with `ErrHandlerNotFound`,
   are retried, then archived. **Intermittent.**
2. **BUG 2 — The imports retention-cleanup task is enqueued to the `default`
   queue, which no server consumes.** Cleanup never runs. **Deterministic.**
3. **BUG 3 — The cohortpositions refresh task is enqueued to the `default`
   queue, while its worker listens only on `cohortpositions`. The worker is
   idle forever and periodic refreshes never run. **Deterministic.**

---

## Asynq mechanics (verified in v0.26.0 source)

These three facts drive every bug below:

1. **A server processes only the queues in its `Config.Queues` map.** If the
   map is empty it defaults to `{"default": 1}`; any queue not listed is never
   consumed (`server.go:135, 409, 477`).
2. **A task enqueued without `asynq.Queue(...)` goes to the `default` queue**
   (`client.go:249`). This includes tasks registered via `asynq.Scheduler`.
3. **`ServeMux` with no matching pattern returns `ErrHandlerNotFound`**
   (`servemux.go`: "handler not found for task"). The task is treated as a
   failure, retried up to its retry limit (`MaxRetry(3)` in our enqueuers), then
   **archived** (`processor.go:343`).

When multiple servers consume the *same* queue, asynq distributes each task to
whichever server claims it — there is no per-task-type routing. Every server on
a queue must therefore be able to handle the **full** set of task types that can
land on that queue.

---

## Current topology (post-refactor)

All servers are built with `database.NewAsynqServer`; only the per-domain
`asynq.Config` differs.

### Servers

| Server (file)                                  | Concurrency | Queues                | Handlers registered |
| ---------------------------------------------- | ----------- | ---------------------- | ------------------- |
| `imports` (`internal/imports/module.go:109`)   | 3           | `{"imports": 10}`      | `imports:process_chunk`, `imports:cleanup_old_data` |
| `attendance` (`internal/attendance/worker.go:239`) | 1       | `{"summaries": 10}`    | `attendance:*` (6 task types) |
| `assessments` (`internal/assessments/worker.go:118`) | 2    | `{"summaries": 10}`    | `assessments:*` (4 task types) |
| `cohortpositions` (`internal/cohortpositions/service.go:143`) | 1 | `{"cohortpositions": 5}` | `cohortpositions:refresh` |

### Producers (enqueue targets)

| Producer                        | Queue      | Task type(s)                                        | Correct? |
| ------------------------------- | ---------- | --------------------------------------------------- | -------- |
| attendance `Enqueuer`           | `summaries`| `attendance:*` (6)                                  | ✔        |
| assessments `Enqueuer`          | `summaries`| `assessments:*` (4)                                 | ✔        |
| `cbctimetableslots.Enqueuer`    | `summaries`| `attendance:refresh_teacher_workload_summaries`     | ✔        |
| imports `Service` (chunks)      | `imports`  | `imports:process_chunk` (`service.go:410`)          | ✔        |
| imports `CleanupScheduler`      | **`default`** | `imports:cleanup_old_data` (`module.go:77`)       | ✘ **BUG 2** |
| cohortpositions `RefreshScheduler` | **`default`** | `cohortpositions:refresh` (`service.go:221`)    | ✘ **BUG 3** |

---

## BUG 1 — `summaries` queue, two servers, disjoint muxes

**Where:** `attendance` and `assessments` workers both set
`Queues: {"summaries": 10}` but register only their own task types.

**What happens:**

1. A refresh handler enqueues `assessments:refresh_overall_summaries` to
   `summaries`.
2. Either server may claim it. If the **attendance** server claims it, its mux
   has no `assessments:*` pattern → `ErrHandlerNotFound`.
3. asynq retries (enqueuers set `MaxRetry(3)`), then **archives** the task.
4. No error is logged by our code — the failure is internal to the asynq
   processor, so the API sees a successful enqueue and stale summary data.

**Impact:** Attendance and assessment summary refreshes fail **intermittently**
(roughly proportional to how many servers share the queue — here the attendance
server will fail `assessments:*` tasks and vice-versa). Chained rollups
(class → term → learning-area) break silently, leaving stale report data.
Archived tasks accumulate in Redis.

**Detection:** `asynq dash` → archived tasks with message
`handler not found for task`; or Redis `LLEN` on `asynq:{summaries}:archive` /
error logs emitted by the shared zap→asynq adapter when tasks are archived.

---

## BUG 2 — imports cleanup task stranded on `default` queue

**Where:** `imports.CleanupScheduler.Start` (`module.go:77`) registers
`imports:cleanup_old_data` with `asynq.NewTask(..., nil)` and **no**
`asynq.Queue(...)` option. The imports server listens only on `{"imports": 10}`.

**What happens:** Every 24h the scheduler enqueues the task to the `default`
queue. No server consumes `default`, so the task sits there forever. The
retention cleanup (`Service.CleanupExpiredData`) never executes.

**Impact:** `imports` staging/error rows are never purged — unbounded table
growth (retention job is dead code in practice).

**Detection:** `asynq dash` → `default` queue depth grows by 1/day; the
`imports:cleanup_old_data` task never leaves the queue.

---

## BUG 3 — cohortpositions refresh task stranded on `default` queue

**Where:** `RefreshScheduler.Start` (`service.go:221`) registers
`cohortpositions:refresh` with no `asynq.Queue(...)` option, while its worker
listens only on `{"cohortpositions": 5}`.

**What happens:** Same as BUG 2 — the 30-minute refresh task goes to `default`
and is never consumed. Additionally, **nothing ever enqueues to the
`cohortpositions` queue**, so that server is permanently idle.

**Impact:** Cohort position summaries are never refreshed on the schedule.
On-demand refreshes via the API service still work (they call the batch
function directly), so the degradation is silent.

**Detection:** `asynq dash` → `cohortpositions` queue always empty, `default`
queue accumulating `cohortpositions:refresh`; no periodic
`cohortpositions: batch refresh starting` log lines.

---

## Fix options

### Bug 1 (choose one)

- **A. One shared server + one shared mux.** Provide a single `*asynq.Server`
  with a merged queue map (e.g. `{"summaries": 10, "imports": 5,
  "cohortpositions": 3}`) and a single `ServeMux`; every domain registers its
  handlers onto the shared mux (a `RegisterHandlers(mux *asynq.ServeMux)` per
  module). Single concurrency budget; requires deciding one `Concurrency` value.
- **B. Keep two servers, make both handle the full `summaries` set.** Both the
  attendance and assessments servers register the complete union of
  `attendance:*` + `assessments:*` handlers (cross-domain handler registration
  is legal — handlers call the same SQL functions). This is the idiomatic asynq
  horizontal-scaling pattern (every consumer of a queue must handle all of its
  task types). Preserves per-domain concurrency tuning.
- **C. Split the queue** (e.g. attendance → `summaries`, assessments →
  `assessments`). Avoids the collision but fragments queue topology and changes
  concurrency semantics; least recommended.

### Bugs 2 & 3 (small, deterministic fixes)

Add the queue option at registration — `Register` accepts `opts ...asynq.Option`:

```go
// imports/module.go
entryID, err := scheduler.Register("@daily", task, asynq.Queue("imports"))

// cohortpositions/service.go
entryID, err := scheduler.Register("*/30 * * * *", task, asynq.Queue("cohortpositions"))
```

Alternatively include `"default"` in the relevant server's `Queues` map, but
explicit queue options at the producer are clearer.

---

## Recommended follow-up

1. Confirm the choice for Bug 1 (A vs B) with the platform team — it changes
   concurrency/queue-priority semantics.
2. Apply the two one-line scheduler fixes (Bugs 2 & 3) regardless.
3. Consider centralizing queue names as exported constants next to the task
   type constants in each domain, so producer/consumer agreement is
   compile-checkable rather than string-agreed.
4. After fixing, verify with `asynq dash` that the `default` queue drains to
   zero and no `handler not found for task` archives accumulate.

---

## Related

- Construction centralization (already merged): `backend/internal/database/asyncq.go`
  — `NewAsynqClient`, `NewAsynqServer`, `NewAsynqScheduler` + shared
  zap→asynq logger adapter. Removed 3 duplicate `NewAsynqClient`s, 3 inline
  server constructions, and 4 duplicate `asynqLogger` types.
- asynq v0.26.0: `servemux.go` (`ErrHandlerNotFound`), `server.go`
  (`defaultQueueConfig`), `client.go` (default queue), `processor.go` (retry/archive).
