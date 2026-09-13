# Admin Invitation Worker — Explained for Everyone

## What is this?
When a school admin uploads a big list of people to invite, the system can't send thousands of emails instantly. This worker is the "mailroom" behind the scenes. It picks up chunks of invitations from a waiting list, sends them one by one through Stytch (the email/auth service), and reports progress back.

---

## Simple Text Flowchart

```
Admin uploads CSV file
         |
         v
Frontend checks fields (email, full name)
         |
         v
Backend saves the big list as a JOB in the database
         |
         v
List is chopped into chunks of 100 people
         |
         v
Each chunk is put in a waiting queue (Redis / asynq)
         |
         v
WORKER (always running, never sleeps)
         |
         +--> Picks up first chunk (100 items)
         |               |
         |               v
         |       For each person in chunk:
         |       +--> Load details from DB
         |       +--> Ask Stytch: "Send invitation email"
         |       +--> Wait for result
         |               |
         |          SUCCESS?  --> Mark done, save IDs
         |          FAIL?     --> Mark failed / try later
         |          DUPLICATE? --> Treat as done (already exists)
         |               |
         v       (Only 10 people processed at once)
         |
         +--> Repeat for all chunks
         |
         v
After every chunk:
         +--> Update JOB status (Processing / Completed / Errors)
         +--> Send update to frontend via Redis
         |
         v
Frontend shows progress bar / success / errors
```

---

## Why batches of 100?
- Stytch (the email service) has limits. Sending 10,000 at once could get blocked.
- If one chunk breaks, only that 100 need retry — not the whole list.
- The database stays stable because it updates 100 items at a time, not 10,000 at once.

---

## What is the "concurrency semaphore"?
Think of it like a ticket system with only 10 tickets.
Before the worker sends a Stytch call, it takes a ticket.
If 10 people are already being processed, the 11th waits until a ticket frees up.
This protects Stytch from being flooded.

---

## Why does the worker stay active forever?
Instead of starting and stopping for each file, it runs continuously like a background service. It listens to Redis and says: "Any new chunks? I'll grab them." Only stops when the server is shut down.

---

## What happens if Stytch fails?
The worker classifies the failure:
- **Temporary (slow / busy)**: Mark as deferred, try again automatically.
- **Permanent (bad email)**: Mark as failed. Admin can retry later via a button.
- **Already exists**: Treat as success so the list isn't blocked.

---

## Key terms in plain English
- **Asynq**: The queue/line system that holds chunks until the worker is ready.
- **Batch / Chunk**: A group of 100 invitation items processed together.
- **Semaphore**: The "only 10 at once" limit.
- **Stytch**: The external service that actually sends the invitation email.
- **Redis Pub/Sub**: The messaging pipe that tells the frontend "3 done, 97 left".

---

## Who touches what?
- **Admin** uploads file → frontend sends to backend
- **Backend handler** validates, creates job, splits into chunks
- **Worker** consumes chunks, talks to Stytch, updates DB
- **Frontend** reads progress from Redis and shows it
