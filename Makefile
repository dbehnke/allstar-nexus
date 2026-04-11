
FRONTEND_DIR=frontend
VUE_DASHBOARD_DIR=frontend
APP_NAME=allstar-nexus
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-X 'main.buildVersion=$(VERSION)' -X 'main.buildTime=$(BUILD_TIME)'

# Installation paths
PREFIX=/usr/local
BINDIR=$(PREFIX)/bin
SYSCONFDIR=/etc
SYSTEMDDIR=/etc/systemd/system
STATEDIR=/var/lib/$(APP_NAME)

.PHONY: frontend backend build frontend-install backend-install build-dashboard build run test test-e2e clean lint tools cgnat-whitelist install uninstall

.PHONY: validate-config

# Validate the configuration file (uses the app's validator)
validate-config:
	./allstar-nexus config validate --config ./config.yaml

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
	cd $(FRONTEND_DIR) && npm run build && npx playwright install --with-deps chromium && npm run test:e2e

lint:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v2.11.4
	golangci-lint run ./...

clean:
	rm -f $(APP_NAME)
	rm -rf $(FRONTEND_DIR)/out
	rm -rf $(VUE_DASHBOARD_DIR)/dist

# Build standalone tools
tools: cgnat-whitelist

cgnat-whitelist:
	cd tools/cgnat-whitelist && go build -o cgnat-whitelist .

# Install the application (requires root/sudo)
install: build
	@echo "Installing $(APP_NAME) to $(BINDIR)..."
	install -d $(BINDIR)
	install -m 755 $(APP_NAME) $(BINDIR)/$(APP_NAME)
	@echo "Creating configuration directory at $(SYSCONFDIR)/$(APP_NAME)..."
	install -d $(SYSCONFDIR)/$(APP_NAME)
	@if [ ! -f $(SYSCONFDIR)/$(APP_NAME)/config.yaml ]; then \
		echo "Installing example config to $(SYSCONFDIR)/$(APP_NAME)/config.yaml..."; \
		install -m 644 config.yaml.example $(SYSCONFDIR)/$(APP_NAME)/config.yaml; \
		echo "*** Please edit $(SYSCONFDIR)/$(APP_NAME)/config.yaml before enabling the service ***"; \
	else \
		echo "Config file already exists at $(SYSCONFDIR)/$(APP_NAME)/config.yaml (not overwriting)"; \
	fi
	@echo "Creating state directory at $(STATEDIR)..."
	install -d -m 755 $(STATEDIR)
	@if id allstar-nexus >/dev/null 2>&1; then \
		echo "User allstar-nexus already exists"; \
	else \
		echo "Creating system user allstar-nexus..."; \
		useradd --system --no-create-home --shell /sbin/nologin allstar-nexus 2>/dev/null || echo "Warning: Could not create user allstar-nexus (insufficient privileges or user creation failed)"; \
	fi
	@if [ -d $(STATEDIR) ]; then \
		chown allstar-nexus:allstar-nexus $(STATEDIR) 2>/dev/null || echo "Warning: Could not set ownership on $(STATEDIR) (insufficient privileges or user does not exist)"; \
	fi
	@echo "Installing systemd service to $(SYSTEMDDIR)/$(APP_NAME).service..."
	install -d $(SYSTEMDDIR)
	install -m 644 $(APP_NAME).service $(SYSTEMDDIR)/$(APP_NAME).service
	@echo ""
	@echo "Installation complete!"
	@echo ""
	@echo "Next steps:"
	@echo "  1. Edit the configuration file: $(SYSCONFDIR)/$(APP_NAME)/config.yaml"
	@echo "  2. Reload systemd: sudo systemctl daemon-reload"
	@echo "  3. Enable the service (optional): sudo systemctl enable $(APP_NAME)"
	@echo "  4. Start the service: sudo systemctl start $(APP_NAME)"
	@echo "  5. Check status: sudo systemctl status $(APP_NAME)"
	@echo ""

# Uninstall the application (requires root/sudo)
uninstall:
	@echo "Stopping and disabling service (if running)..."
	-systemctl stop $(APP_NAME).service 2>/dev/null
	-systemctl disable $(APP_NAME).service 2>/dev/null
	@echo "Removing systemd service file..."
	rm -f $(SYSTEMDDIR)/$(APP_NAME).service
	systemctl daemon-reload 2>/dev/null || true
	@echo "Removing binary from $(BINDIR)..."
	rm -f $(BINDIR)/$(APP_NAME)
	@echo ""
	@echo "Uninstallation complete!"
	@echo ""
	@echo "Note: Configuration files in $(SYSCONFDIR)/$(APP_NAME) and"
	@echo "      data in $(STATEDIR) were preserved."
	@echo "      Remove them manually if desired:"
	@echo "        sudo rm -rf $(SYSCONFDIR)/$(APP_NAME)"
	@echo "        sudo rm -rf $(STATEDIR)"
	@echo ""
