.PHONY: build run clean test install help

# Default target
all: build

# Build the application
build:
	go build -o newsreadr cmd/newsreadr/main.go

# Run the application
run: build
	./newsreadr

# Clean build artifacts
clean:
	rm -f newsreadr
	go clean

# Test the application
test:
	chmod +x test_app.sh
	./test_app.sh

# Install dependencies
deps:
	go mod download
	go mod tidy

# Install to system PATH
install: build
	sudo mv newsreadr /usr/local/bin/

# Install locally to ~/bin
install-local: build
	mkdir -p ~/bin
	mv newsreadr ~/bin/

# Clean configuration and database (reset app)
reset:
	rm -rf ~/.config/newsreader/

# Check if Ollama is available
check-ollama:
	@if curl -s http://localhost:11434/api/tags > /dev/null 2>&1; then \
		echo "✓ Ollama is running and accessible"; \
	else \
		echo "⚠ Ollama is not running or not accessible"; \
		echo "  Install with: curl https://ollama.ai/install.sh | sh"; \
		echo "  Then run: ollama pull llama2"; \
	fi

# Development build with race detection
dev:
	go build -race -o newsreadr cmd/newsreadr/main.go

# Format code
fmt:
	go fmt ./...

# Vet code for issues
vet:
	go vet ./...

# Show help
help:
	@echo "NewsReadr - Available Make targets:"
	@echo ""
	@echo "  build          Build the application"
	@echo "  run            Build and run the application"
	@echo "  test           Run the test script"
	@echo "  clean          Clean build artifacts"
	@echo "  deps           Install Go dependencies"
	@echo "  install        Install to /usr/local/bin (requires sudo)"
	@echo "  install-local  Install to ~/bin"
	@echo "  reset          Remove config and database files"
	@echo "  check-ollama   Check if Ollama is running"
	@echo "  dev            Build with race detection"
	@echo "  fmt            Format Go code"
	@echo "  vet            Run go vet"
	@echo "  help           Show this help message"
	@echo ""
	@echo "First time setup:"
	@echo "  make build     # Build the app"
	@echo "  make run       # Create config file"
	@echo "  make reset     # Reset if needed"
	@echo "  make test      # Verify everything works"
