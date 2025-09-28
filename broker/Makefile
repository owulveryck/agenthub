# Makefile for broker tests

.PHONY: test test-unit test-integration test-concurrency test-performance test-all coverage benchmark clean help

# Default target
all: test

# Run all tests
test: test-unit test-integration test-concurrency

# Run unit tests only
test-unit:
	@echo "Running unit tests..."
	go test -v -run="^Test.*" -short ./

# Run integration tests
test-integration:
	@echo "Running integration tests..."
	go test -v -run="^TestIntegration.*" ./

# Run concurrency tests
test-concurrency:
	@echo "Running concurrency tests..."
	go test -v -run="^TestConcurrent.*|^TestRace.*|^TestMemoryLeak.*|^TestDeadlock.*" ./

# Run performance tests
test-performance:
	@echo "Running performance tests..."
	go test -v -run="^TestThroughput.*|^TestLatency.*|^TestMemoryUsage.*|^TestResource.*" ./

# Run all tests including performance
test-all: test test-performance

# Run tests with race detection
test-race:
	@echo "Running tests with race detection..."
	go test -v -race ./

# Generate test coverage
coverage:
	@echo "Generating test coverage..."
	./coverage.sh

# Run benchmarks
benchmark:
	@echo "Running benchmarks..."
	go test -v -bench=. -benchmem -run=^$$ ./

# Run benchmarks with CPU profiling
benchmark-cpu:
	@echo "Running benchmarks with CPU profiling..."
	go test -v -bench=. -benchmem -cpuprofile=cpu.prof -run=^$$ ./

# Run benchmarks with memory profiling
benchmark-mem:
	@echo "Running benchmarks with memory profiling..."
	go test -v -bench=. -benchmem -memprofile=mem.prof -run=^$$ ./

# Run stress tests
stress:
	@echo "Running stress tests..."
	go test -v -count=10 -run="^TestConcurrent.*" ./

# Lint the code
lint:
	@echo "Running linter..."
	golangci-lint run

# Format the code
fmt:
	@echo "Formatting code..."
	go fmt ./
	goimports -w .

# Tidy modules
tidy:
	@echo "Tidying modules..."
	go mod tidy

# Verify modules
verify:
	@echo "Verifying modules..."
	go mod verify

# Clean test artifacts
clean:
	@echo "Cleaning test artifacts..."
	rm -rf coverage/
	rm -f *.prof
	rm -f *.test

# Install test dependencies
deps:
	@echo "Installing test dependencies..."
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run quick tests (short mode)
quick:
	@echo "Running quick tests..."
	go test -v -short ./

# Run tests in verbose mode
verbose:
	@echo "Running tests in verbose mode..."
	go test -v -race -cover ./

# Generate test report
report:
	@echo "Generating test report..."
	go test -v -json -cover ./ > test_report.json

# Run mutation tests (if gotestfmt is available)
mutation:
	@echo "Running mutation tests..."
	go test -v -cover ./ | gotestfmt

# Display help
help:
	@echo "Available targets:"
	@echo "  test            - Run unit, integration, and concurrency tests"
	@echo "  test-unit       - Run only unit tests"
	@echo "  test-integration- Run only integration tests"
	@echo "  test-concurrency- Run only concurrency tests"
	@echo "  test-performance- Run only performance tests"
	@echo "  test-all        - Run all tests including performance"
	@echo "  test-race       - Run tests with race detection"
	@echo "  coverage        - Generate test coverage report"
	@echo "  benchmark       - Run performance benchmarks"
	@echo "  benchmark-cpu   - Run benchmarks with CPU profiling"
	@echo "  benchmark-mem   - Run benchmarks with memory profiling"
	@echo "  stress          - Run stress tests"
	@echo "  lint            - Run linter"
	@echo "  fmt             - Format code"
	@echo "  tidy            - Tidy modules"
	@echo "  verify          - Verify modules"
	@echo "  clean           - Clean test artifacts"
	@echo "  deps            - Install test dependencies"
	@echo "  quick           - Run quick tests (short mode)"
	@echo "  verbose         - Run tests in verbose mode"
	@echo "  report          - Generate JSON test report"
	@echo "  help            - Show this help message"