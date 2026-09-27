# Top Five European Football Sim — verification and run tasks.
# The check targets mirror .github/workflows/ci.yml exactly.

GO := go
BUN := bun
BACKEND := backend_go
FRONTEND := frontend

.PHONY: help test-backend vet-backend race-backend test-frontend build-frontend check run dev docker-build clean

help:
	@echo "Targets:"
	@echo "  make check          run every verification gate (backend + frontend)"
	@echo "  make test-backend   go test ./..."
	@echo "  make vet-backend    go vet ./..."
	@echo "  make race-backend   go test -race -p=1 ./..."
	@echo "  make test-frontend  bun test"
	@echo "  make build-frontend bun run build (tsc + vite)"
	@echo "  make run            build the client and start the Go server"
	@echo "  make dev            start Vite dev server (Go server must run separately)"
	@echo "  make docker-build   build the multi-stage container image"

test-backend:
	cd $(BACKEND) && $(GO) test ./...

vet-backend:
	cd $(BACKEND) && $(GO) vet ./...

race-backend:
	cd $(BACKEND) && $(GO) test -race -p=1 ./...

test-frontend:
	cd $(FRONTEND) && $(BUN) test

build-frontend:
	cd $(FRONTEND) && $(BUN) run build

check: test-backend vet-backend race-backend test-frontend build-frontend

run:
	cd $(FRONTEND) && $(BUN) install && $(BUN) run build
	cd $(BACKEND) && $(GO) run ./cmd/server -dataset ../dataset.json -static ../frontend/dist -save ../saves/career.json

dev:
	cd $(FRONTEND) && $(BUN) run dev

docker-build:
	docker build -t football-sim .

clean:
	cd $(FRONTEND) && rm -rf dist
