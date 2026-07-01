.PHONY: setup db-up db-down migrate ingest backend frontend test lint

# -- Database --
db-up:
	docker compose -f infra/docker-compose.yml up -d

db-down:
	docker compose -f infra/docker-compose.yml down

# -- Backend --
setup-backend:
	cd backend && go mod download && go mod tidy

migrate:
	cd backend && set -a && . ./.env && set +a && $(HOME)/go/bin/goose -dir migrations postgres "$$DATABASE_URL" up

ingest:
	cd backend && set -a && . ./.env && set +a && go run ./cmd/ingest

backend:
	cd backend && set -a && . ./.env && set +a && go run ./cmd/api

# -- Frontend --
setup-frontend:
	cd frontend && npm install

frontend:
	cd frontend && npm run dev

# -- Tests --
test-backend:
	cd backend && go test -race -count=1 ./...

test-frontend:
	cd frontend && npm run test

test: test-backend test-frontend

# -- Lint --
lint-backend:
	cd backend && golangci-lint run ./...

lint-frontend:
	cd frontend && npm run lint

lint: lint-backend lint-frontend

# -- All setup --
setup: db-up setup-backend setup-frontend migrate ingest
	@echo "Setup completo. Inicie backend (make backend) e frontend (make frontend) em terminais separados."
