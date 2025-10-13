
FRONTEND_DIR=frontend
VUE_DASHBOARD_DIR=frontend
APP_NAME=allstar-nexus
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-X 'main.buildVersion=$(VERSION)' -X 'main.buildTime=$(BUILD_TIME)'

.PHONY: frontend backend build frontend-install backend-install build-dashboard build run test test-e2e clean lint

# Build the legacy Next.js exported frontend (if used)
frontend:
	cd $(FRONTEND_DIR) && npm install && npm run build

# Build the Vue dashboard (now located in $(FRONTEND_DIR))
build-dashboard:
	cd $(VUE_DASHBOARD_DIR) && npm install && npm run build

# Install deps for frontend only (useful in CI)
frontend-install:
	cd $(FRONTEND_DIR) && npm ci

backend-install:
	go mod download

# Build backend binary. Depends on dashboard build so the embedded assets are up to date.
backend: build-dashboard backend-install
	go build -ldflags "$(LDFLAGS)" -o $(APP_NAME) .

build: backend

# Run the app (builds frontends first for consistency)
run: build-dashboard
	go run -ldflags "$(LDFLAGS)" .

test:
	go test ./backend/... -count=1
	cd $(FRONTEND_DIR) && CI=TRUE npm test

# Run end-to-end Playwright tests separately (Chromium by default)
test-e2e:
	cd $(FRONTEND_DIR) && npx playwright install --with-deps chromium && npm run test:e2e

lint:
	@echo "(placeholder) add golangci-lint or staticcheck here"

clean:
	rm -f $(APP_NAME)
	rm -rf $(FRONTEND_DIR)/out
	rm -rf $(VUE_DASHBOARD_DIR)/dist
