SHELL := /bin/sh
DEV_DB := postgresql://root@localhost:26257/dashboard?sslmode=disable
export PATH := $(HOME)/go/bin:$(PATH)

.PHONY: up down db backend frontend test smoke proto fixture

up: ## build and start the whole stack
	docker compose up -d --build

down:
	docker compose down

db: ## CockroachDB only, published on 127.0.0.1:26257
	docker compose -f docker-compose.yml -f docker-compose.dev.yml up -d cockroach

backend: ## run the backend on the host against the dev database
	set -a; . ./.env; set +a; cd backend && DATABASE_URL='$(DEV_DB)' go run ./cmd/server

frontend: ## next dev on :3300 against the local backend
	cd frontend && BACKEND_URL=http://localhost:8000 npx next dev -p 3300

test:
	cd backend && go vet ./... && go test ./...
	cd frontend && npx tsc --noEmit && npx eslint app components lib hooks

smoke:
	sh scripts/smoke.sh

proto:
	cd backend && buf lint && buf generate

fixture:
	cd backend && DATABASE_URL='$(DEV_DB)' go run ./cmd/fixture > internal/report/testdata/fixture.json
