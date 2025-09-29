# 🏗️ Go Build Tags in AgentHub

**Understanding-oriented**: Deep dive into how AgentHub uses Go build tags to create flexible, conditional compilation for different deployment scenarios and feature sets.

## The Problem: Conditional Feature Compilation

AgentHub needs to support multiple deployment scenarios:

- **Development**: Fast builds without observability overhead
- **Testing**: Minimal dependencies for unit tests
- **Production**: Full observability with tracing, metrics, and monitoring
- **Edge deployments**: Lightweight versions with reduced features

Traditional approaches like runtime flags have drawbacks:
- **Binary bloat**: All code included even when unused
- **Runtime overhead**: Conditional checks in hot paths
- **Dependency complexity**: All dependencies must be available
- **Security concerns**: Observable code included in minimal deployments

## The Solution: Go Build Tags

Build tags (also called build constraints) allow Go to conditionally include or exclude files during compilation based on specified tags.

### How Build Tags Work

Build tags are special comments at the top of Go files:

```go
//go:build observability
// +build observability

package main

// This file is only compiled when 'observability' tag is specified
```

### Build Tag Syntax

**Modern syntax** (Go 1.17+):
```go
//go:build observability
```

**Legacy syntax** (backwards compatibility):
```go
// +build observability
```

**Complex conditions**:
```go
//go:build observability && !minimal
//go:build (observability || debug) && linux
//go:build observability,debug
```

### Boolean Logic in Build Tags

| **Syntax** | **Meaning** | **Example** |
|------------|-------------|-------------|
| `tag1 && tag2` | Both tags required | `//go:build observability && debug` |
| `tag1 \|\| tag2` | Either tag required | `//go:build observability \|\| debug` |
| `!tag1` | Tag must NOT be present | `//go:build !minimal` |
| `tag1,tag2` | Either tag (legacy OR) | `// +build observability,debug` |

## AgentHub Build Tag Architecture

### File Organization Strategy

```
agents/publisher/
├── main.go                 # Unified abstraction with built-in observability
├── shared.go              # Common code (no build tags)
└── config.go              # Configuration (no build tags)

broker/
├── main.go                 # Unified abstraction with built-in observability
└── server.go              # Core server logic (shared)

internal/
├── grpc/                   # Generated code (always included)
└── observability/          # Observability package
    ├── config.go           # Build tag: observability
    ├── metrics.go          # Build tag: observability
    ├── tracing.go          # Build tag: observability
    └── healthcheck.go      # Build tag: observability
```

### Build Tag Usage in AgentHub

#### 1. Observable Components

Files that should only be compiled with observability:

```go
//go:build observability
// +build observability

package main

import (
    "context"
    "log/slog"
    "github.com/owulveryck/agenthub/internal/observability"
)

type ObservableAgent struct {
    obs            *observability.Observability
    traceManager   *observability.TraceManager
    metricsManager *observability.MetricsManager
    logger         *slog.Logger
}
```

#### 2. Basic Components

Files that should be excluded when observability is enabled:

```go
//go:build !observability
// +build !observability

package main

import (
    "log"
)

type BasicAgent struct {
    // Simple struct without observability
}

func main() {
    log.Println("Starting basic agent...")
    // Basic implementation
}
```

#### 3. Shared Components

Files without build tags are always included:

```go
package main

// No build tags - always included

import (
    "context"
    pb "github.com/owulveryck/agenthub/internal/grpc"
)

// Common functionality used by both versions
func processTask(ctx context.Context, task *pb.TaskMessage) error {
    // Shared business logic
    return nil
}
```

## Build Commands and Results

### Development Builds

**Basic agent** (default):
```bash
go build -o bin/publisher agents/publisher/
# Result: Small binary, no observability dependencies
# Files included: main.go, shared.go, config.go
# Files excluded: (none - unified abstraction includes observability)
```

**Observable agent**:
```bash
go build -tags observability -o bin/publisher-obs agents/publisher/
# Result: Full-featured binary with observability
# Files included: main.go with unified abstraction, shared.go, config.go
# Files excluded: main.go
```

### Production Builds

**Minimal production**:
```bash
go build -tags "!observability,!debug" -ldflags="-s -w" -o bin/publisher-minimal
# Ultra-small binary for resource-constrained environments
```

**Full production**:
```bash
go build -tags "observability,production" -ldflags="-s -w" -o bin/publisher-prod
# Full observability with production optimizations
```

## Advanced Build Tag Patterns

### 1. Environment-Specific Builds

```go
//go:build development
// +build development

// Development-only features
func enableDebugEndpoints() {
    // Debug HTTP endpoints, verbose logging
}
```

```go
//go:build production
// +build production

// Production-only optimizations
func enableProductionOptimizations() {
    // Performance tuning, resource limits
}
```

### 2. Platform-Specific Builds

```go
//go:build observability && linux
// +build observability,linux

// Linux-specific observability features
import "syscall"

func getLinuxMetrics() {
    // Linux-specific system metrics
}
```

### 3. Feature Flags

```go
//go:build experimental
// +build experimental

// Experimental features behind build tags
func experimentalEventProcessing() {
    // New algorithms not ready for production
}
```

## Build Tag Testing

### Testing Observable Code

```go
//go:build observability
// +build observability

package main

import "testing"

func TestObservableAgent(t *testing.T) {
    // Tests that only run with observability enabled
    agent := NewObservableAgent()
    // Test observability features
}
```

**Run observable tests**:
```bash
go test -tags observability ./...
```

### Testing Basic Code

```go
//go:build !observability
// +build !observability

package main

import "testing"

func TestBasicAgent(t *testing.T) {
    // Tests for basic functionality
    agent := NewBasicAgent()
    // Test core features without observability
}
```

**Run basic tests**:
```bash
go test ./...  # Default excludes observability tests
```

## Performance Impact Analysis

### Binary Size Comparison

| **Build Type** | **Binary Size** | **Dependencies** | **Startup Time** |
|----------------|-----------------|------------------|------------------|
| Basic | ~8MB | Core gRPC only | 50ms |
| Observable | ~15MB | + OpenTelemetry, Prometheus | 150ms |
| Full Production | ~12MB | + Optimizations | 100ms |

### Memory Usage Patterns

**Basic Agent**:
```
Heap: 10MB baseline
Goroutines: 5-10
Dependencies: minimal
```

**Observable Agent**:
```
Heap: 10MB + 5MB observability overhead
Goroutines: 5-10 + 3-5 observability routines
Dependencies: OpenTelemetry SDK, Prometheus client
```

## Build Tag Best Practices

### 1. Consistent Naming Conventions

```bash
# Files
main.go                    # Basic version
main.go                    # Unified version with built-in observability
main_debug.go             # Debug version

# Build tags
observability             # Observability features
debug                     # Debug features
experimental              # Experimental features
production                # Production optimizations
minimal                   # Minimal builds
```

### 2. Clear Documentation

```go
//go:build observability
// +build observability

// This file contains the observable version of the publisher agent.
// It includes distributed tracing, metrics collection, and structured logging.
//
// Build with: go build -tags observability
//
// Features included:
// - OpenTelemetry tracing
// - Prometheus metrics
// - Health checks
// - Graceful shutdown

package main
```

### 3. Makefile Integration

```makefile
# Basic builds
build-basic:
	go build -o bin/broker broker/
	go build -o bin/publisher agents/publisher/
	go build -o bin/subscriber agents/subscriber/

# Observable builds
build-observable:
	go build -tags observability -o bin/broker-obs broker/
	go build -tags observability -o bin/publisher-obs agents/publisher/
	go build -tags observability -o bin/subscriber-obs agents/subscriber/

# Production builds
build-production:
	go build -tags "observability,production" -ldflags="-s -w" -o bin/broker-prod broker/

# Development builds
build-dev:
	go build -tags "observability,debug" -o bin/broker-dev broker/
```

### 4. CI/CD Integration

```yaml
# .github/workflows/build.yml
strategy:
  matrix:
    build-type: [basic, observable, production]

steps:
- name: Build Basic
  if: matrix.build-type == 'basic'
  run: make build-basic

- name: Build Observable
  if: matrix.build-type == 'observable'
  run: make build-observable

- name: Build Production
  if: matrix.build-type == 'production'
  run: make build-production
```

## Troubleshooting Build Tags

### Common Issues

1. **File not included in build**
   ```bash
   # Check which files are included
   go list -f '{{.GoFiles}}' ./agents/publisher/

   # Verify build tags
   go build -tags observability -v ./agents/publisher/
   ```

2. **Conflicting build tags**
   ```go
   // Problem: Both files might be included
   //go:build observability     // File A
   //go:build !observability    // File B

   // Solution: Use more specific tags
   //go:build observability && !minimal    // File A
   //go:build !observability || minimal    // File B
   ```

3. **Missing dependencies**
   ```bash
   # Observability build fails due to missing imports
   go build -tags observability ./
   # Fix: Ensure all observability dependencies are available
   go mod tidy
   ```

### Debug Build Tag Issues

```bash
# Show which files would be compiled
go list -f '{{.GoFiles}}' -tags observability ./agents/publisher/

# Show build constraints
go list -f '{{.Imports}}' -tags observability ./agents/publisher/

# Verbose build output
go build -tags observability -v -x ./agents/publisher/
```

## Design Decisions and Trade-offs

### Why Build Tags vs Runtime Flags?

**Build Tags Advantages**:
- ✅ Zero runtime overhead for excluded features
- ✅ Smaller binary size for minimal deployments
- ✅ Clear separation of concerns
- ✅ Compile-time safety

**Build Tags Disadvantages**:
- ❌ More complex build process
- ❌ Multiple binaries to maintain
- ❌ Cannot change features at runtime

**Runtime Flags Advantages**:
- ✅ Single binary for all deployments
- ✅ Runtime feature toggling
- ✅ Simpler deployment process

**Runtime Flags Disadvantages**:
- ❌ All code included in binary
- ❌ Runtime performance overhead
- ❌ Security concerns (observable code in minimal builds)

### AgentHub's Choice

AgentHub chose build tags because:

1. **Performance Critical**: Event processing requires minimal overhead
2. **Security Conscious**: Minimal deployments shouldn't include observability code
3. **Resource Constrained**: Edge deployments need smallest possible binaries
4. **Clear Boundaries**: Observability is a distinct architectural concern

## Integration with AgentHub Architecture

### Event Flow with Build Tags

**Basic Flow**:
```
Publisher → Broker → Subscriber
(No tracing, minimal logging)
```

**Observable Flow**:
```
Publisher (+ tracing) → Broker (+ metrics) → Subscriber (+ logging)
     ↓                       ↓                      ↓
  Jaeger                 Prometheus              Structured Logs
```

### Observability Package Architecture

```go
//go:build observability
package observability

type Observability struct {
    // Only compiled when observability tag is used
    Tracer     trace.Tracer
    Meter      metric.Meter
    Logger     *slog.Logger
}

func NewObservability(config Config) (*Observability, error) {
    // OpenTelemetry initialization
    // Only available in observable builds
}
```

## Future Considerations

### Planned Build Tag Extensions

1. **Cloud Provider Tags**:
   ```go
   //go:build observability && aws
   // AWS-specific observability features

   //go:build observability && gcp
   // Google Cloud specific features
   ```

2. **Protocol Tags**:
   ```go
   //go:build grpc
   // gRPC transport only

   //go:build http
   // HTTP transport only
   ```

3. **Feature Flags**:
   ```go
   //go:build eventstore
   // Event sourcing capabilities

   //go:build encryption
   // End-to-end encryption
   ```

---

Build tags are a powerful tool that enables AgentHub to maintain flexibility while optimizing for different deployment scenarios. They provide compile-time feature selection that ensures optimal performance and minimal resource usage across diverse environments.