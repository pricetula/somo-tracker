.PHONY: up up-d down ps logs-down logs-api logs-postgres logs-redis logs-frontend restart-api rebuild-api build-api

# ─── Docker Compose shortcuts ────────────────────────────────────────────────

up:    ## Start all services (detached)
	docker compose up -d

up-d:  ## Start all services (foreground, live logs)
	docker compose up

down:  ## Stop and remove all containers
	docker compose down

ps:    ## Show running service status
	docker compose ps

# ─── Logs ────────────────────────────────────────────────────────────────────

logs-down:  ## Tail logs for all services
	docker compose logs -f

logs-api:   ## Tail API (Go/Fiber) logs
	docker compose logs -f somotracker_api

logs-postgres: ## Tail Postgres logs
	docker compose logs -f somotracker_postgres

logs-redis: ## Tail Redis logs
	docker compose logs -f somotracker_redis

logs-frontend: ## Tail Next.js frontend logs
	docker compose logs -f somotracker_frontend

# ─── Service lifecycle ────────────────────────────────────────────────────────

restart-api:  ## Restart the API service (triggers Air hot-reload)
	docker compose restart somotracker_api

rebuild-api:  ## Rebuild and recreate the API container
	docker compose up -d --force-recreate --no-deps somotracker_api

build-api:    ## Build the API service (no start)
	docker compose build somotracker_api

# ─── Help ─────────────────────────────────────────────────────────────────────

help:  ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| sort \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
