# GET /health

**Group:** Global

**Auth:** None

**Description:** Health check endpoint that returns the status of the application and its dependencies (PostgreSQL and Redis). Used by load balancers, orchestrators, and monitoring systems.

**Response `200`:**

```json
{
  "status": "ok",
  "postgres": "healthy",
  "redis": "healthy",
  "env": "development|staging|production"
}
```
