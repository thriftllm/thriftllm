.PHONY: dev dev-backend dev-frontend build up down logs migrate

# Development
dev-backend:
	cd backend && go run ./cmd/server

dev-frontend:
	cd frontend && npm run dev

# Docker
build:
	docker compose build

up:
	docker compose up -d

down:
	docker compose down

logs:
	docker compose logs -f

restart:
	docker compose restart backend frontend

# Database
psql:
	docker compose exec postgres psql -U thrift -d thriftllm

# Redis
redis-cli:
	docker compose exec redis redis-cli

# Clean
clean:
	docker compose down -v
	rm -rf backend/bin frontend/.next
