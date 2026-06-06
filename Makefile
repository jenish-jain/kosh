.PHONY: dev dev-backend dev-frontend build build-backend build-frontend clean install

# ── Dev (run both servers) ────────────────────────────────────
dev:
	@trap 'kill 0' INT; \
	$(MAKE) dev-backend & \
	$(MAKE) dev-frontend & \
	wait

dev-backend:
	cd backend && go run .

dev-frontend:
	cd frontend && npm run dev

# ── Build ─────────────────────────────────────────────────────
build: build-frontend build-backend

build-frontend:
	cd frontend && npm run build

build-backend:
	cd backend && go build -o kosh .

# ── Install dependencies ──────────────────────────────────────
install:
	cd backend && go mod download
	cd frontend && npm install

# ── Clean ─────────────────────────────────────────────────────
clean:
	rm -rf frontend/dist backend/kosh
