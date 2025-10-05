FRONTEND_DIR=frontend
APP_NAME=allstar-nexus

.PHONY: build-frontend build-backend build run test clean lint build-dashboard

build-frontend:
	cd $(FRONTEND_DIR) && npm install && npm run build

build-dashboard:
	cd vue-dashboard && npm install && npm run build

build-backend: build-frontend build-dashboard
	go build -o $(APP_NAME) .

build: build-backend

run: build-frontend build-dashboard
	go run .

test:
	go test ./backend/... -count=1

lint:
	@echo "(placeholder) add golangci-lint or staticcheck here"

clean:
	rm -f $(APP_NAME)
	rm -rf $(FRONTEND_DIR)/out
