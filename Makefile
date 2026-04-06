.PHONY: build build-web dev test release clean

# Default target
all: build

# Build the web frontend
build-web:
	cd web && npm install && npx nuxi generate
	rm -rf cmd/juli/web
	cp -r web/.output/public cmd/juli/web

# Build the Go binary (includes embedded web)
build: build-web
	go build -o juli ./cmd/juli

# Development mode: run Go server + Nuxt dev server concurrently
dev:
	@echo "Start in two terminals:"
	@echo "  Terminal 1: go run ./cmd/juli --token SECRET --port 8080"
	@echo "  Terminal 2: cd web && npm run dev"

# Run tests
test:
	go test ./...
	cd web && npm install --silent && npm test

# Cross-compile releases
release: build-web
	GOOS=linux GOARCH=amd64 go build -o release/juli-linux-amd64 ./cmd/juli
	GOOS=linux GOARCH=arm64 go build -o release/juli-linux-arm64 ./cmd/juli
	GOOS=darwin GOARCH=amd64 go build -o release/juli-darwin-amd64 ./cmd/juli
	GOOS=darwin GOARCH=arm64 go build -o release/juli-darwin-arm64 ./cmd/juli

# Clean build artifacts
clean:
	rm -f juli
	rm -rf release/
	rm -rf cmd/juli/web
	rm -rf web/.output web/.nuxt web/node_modules
