# Performance and Scaling Considerations

This document explores the performance characteristics of AgentHub, scaling patterns, and optimization strategies for different deployment scenarios.

## Performance Characteristics

### Baseline Performance Metrics

**Test Environment:**
- 4-core Intel i7 processor
- 16GB RAM
- Local network (localhost)
- Go 1.24

**Measured Performance:**
- **Task throughput**: 8,000-12,000 tasks/second
- **Task routing latency**: 0.1-0.5ms average
- **End-to-end latency**: 2-10ms (including processing)
- **Memory per agent**: ~1KB active subscription state
- **Concurrent agents**: 1,000+ agents per broker instance

### Performance Factors

#### 1. Task Routing Performance

Task routing is the core performance bottleneck in AgentHub:

```go
// Fast path: Direct agent routing
if responderID := req.GetTask().GetResponderAgentId(); responderID != "" {
    if subs, ok := s.taskSubscribers[responderID]; ok {
        targetChannels = subs  // O(1) lookup
    }
}
```

**Optimization factors:**
- **Direct routing**: O(1) lookup time for targeted tasks
- **Broadcast routing**: O(n) where n = number of subscribed agents
- **Channel delivery**: Concurrent delivery via goroutines
- **Lock contention**: Read locks allow concurrent routing

#### 2. Message Serialization

Protocol Buffers provide efficient serialization:

- **Binary encoding**: ~60% smaller than JSON
- **Zero-copy operations**: Direct memory mapping where possible
- **Schema evolution**: Backward/forward compatibility
- **Type safety**: Compile-time validation

#### 3. Memory Usage Patterns

```go
// Memory usage breakdown per agent:
type agentMemoryFootprint struct {
    SubscriptionState    int // ~200 bytes (map entry + channel)
    ChannelBuffer       int // ~800 bytes (10 message buffer * 80 bytes avg)
    ConnectionOverhead  int // ~2KB (gRPC stream state)
    // Total: ~3KB per active agent
}
```

**Memory optimization strategies:**
- **Bounded channels**: Prevent unbounded growth
- **Connection pooling**: Reuse gRPC connections
- **Garbage collection**: Go's GC handles cleanup automatically

## Scaling Patterns

### Vertical Scaling (Scale Up)

Increasing resources on a single broker instance:

#### CPU Scaling
- **Multi-core utilization**: Go's runtime leverages multiple cores
- **Goroutine efficiency**: Lightweight concurrency (2KB stack)
- **CPU-bound operations**: Message serialization, routing logic

```go
// Configure for CPU optimization
export GOMAXPROCS=8  // Match available CPU cores
```

#### Memory Scaling
- **Linear growth**: Memory usage scales with number of agents
- **Buffer tuning**: Adjust channel buffer sizes based on throughput

```go
// Memory-optimized configuration
subChan := make(chan *pb.TaskMessage, 5)  // Smaller buffers for memory-constrained environments
// vs
subChan := make(chan *pb.TaskMessage, 50) // Larger buffers for high-throughput environments
```

#### Network Scaling
- **Connection limits**: OS file descriptor limits (ulimit -n)
- **Bandwidth utilization**: Protocol Buffers minimize bandwidth usage
- **Connection keepalive**: Efficient connection reuse

### Horizontal Scaling (Scale Out)

Distributing load across multiple broker instances:

#### 1. Agent Partitioning

**Static Partitioning:**
```
Agent Groups:
├── Broker 1: agents_1-1000
├── Broker 2: agents_1001-2000
└── Broker 3: agents_2001-3000
```

**Hash-based Partitioning:**
```go
func selectBroker(agentID string) string {
    hash := fnv.New32a()
    hash.Write([]byte(agentID))
    brokerIndex := hash.Sum32() % uint32(len(brokers))
    return brokers[brokerIndex]
}
```

#### 2. Task Type Partitioning

**Specialized Brokers:**
```
Task Routing:
├── Broker 1: data_processing, analytics
├── Broker 2: image_processing, ml_inference
└── Broker 3: notifications, logging
```

#### 3. Geographic Partitioning

**Regional Distribution:**
```
Geographic Deployment:
├── US-East: Broker cluster for East Coast agents
├── US-West: Broker cluster for West Coast agents
└── EU: Broker cluster for European agents
```

### Load Balancing Strategies

#### 1. Round-Robin Agent Distribution

```go
type LoadBalancer struct {
    brokers []string
    current int
    mu      sync.Mutex
}

func (lb *LoadBalancer) NextBroker() string {
    lb.mu.Lock()
    defer lb.mu.Unlock()

    broker := lb.brokers[lb.current]
    lb.current = (lb.current + 1) % len(lb.brokers)
    return broker
}
```

#### 2. Capacity-Based Routing

```go
type BrokerMetrics struct {
    ActiveAgents int
    TasksPerSec  float64
    CPUUsage     float64
    MemoryUsage  float64
}

func selectBestBroker(brokers []BrokerMetrics) int {
    // Select broker with lowest load score
    bestIndex := 0
    bestScore := calculateLoadScore(brokers[0])

    for i, broker := range brokers[1:] {
        score := calculateLoadScore(broker)
        if score < bestScore {
            bestScore = score
            bestIndex = i + 1
        }
    }
    return bestIndex
}
```

## Performance Optimization Strategies

### 1. Message Batching

For high-throughput scenarios, implement message batching:

```go
type BatchProcessor struct {
    tasks     []*pb.TaskMessage
    batchSize int
    timeout   time.Duration
    ticker    *time.Ticker
}

func (bp *BatchProcessor) processBatch() {
    batch := make([]*pb.TaskMessage, len(bp.tasks))
    copy(batch, bp.tasks)
    bp.tasks = bp.tasks[:0] // Clear slice

    // Process entire batch
    go bp.routeBatch(batch)
}
```

### 2. Connection Pooling

Optimize gRPC connections for better resource utilization:

```go
type ConnectionPool struct {
    connections map[string]*grpc.ClientConn
    maxConns    int
    mu          sync.RWMutex
}

func (cp *ConnectionPool) GetConnection(addr string) (*grpc.ClientConn, error) {
    cp.mu.RLock()
    if conn, exists := cp.connections[addr]; exists {
        cp.mu.RUnlock()
        return conn, nil
    }
    cp.mu.RUnlock()

    // Create new connection
    return cp.createConnection(addr)
}
```

### 3. Adaptive Channel Sizing

Dynamically adjust channel buffer sizes based on load:

```go
func calculateOptimalBufferSize(avgTaskRate float64, processingTime time.Duration) int {
    // Buffer size = rate * processing time + safety margin
    bufferSize := int(avgTaskRate * processingTime.Seconds()) + 10

    // Clamp to reasonable bounds
    if bufferSize < 5 {
        return 5
    }
    if bufferSize > 100 {
        return 100
    }
    return bufferSize
}
```

### 4. Memory Optimization

Reduce memory allocations in hot paths:

```go
// Use sync.Pool for frequent allocations
var taskPool = sync.Pool{
    New: func() interface{} {
        return &pb.TaskMessage{}
    },
}

func processTaskOptimized(task *pb.TaskMessage) {
    // Reuse task objects
    pooledTask := taskPool.Get().(*pb.TaskMessage)
    defer taskPool.Put(pooledTask)

    // Copy and process
    *pooledTask = *task
    // ... processing logic
}
```

## Monitoring and Metrics

### Key Performance Indicators (KPIs)

#### Throughput Metrics
```go
type ThroughputMetrics struct {
    TasksPerSecond     float64
    ResultsPerSecond   float64
    ProgressPerSecond  float64
    MessagesPerSecond  float64
}
```

#### Latency Metrics
```go
type LatencyMetrics struct {
    RoutingLatency     time.Duration // Broker routing time
    ProcessingLatency  time.Duration // Agent processing time
    EndToEndLatency    time.Duration // Total task completion time
    P50, P95, P99      time.Duration // Percentile latencies
}
```

#### Resource Metrics
```go
type ResourceMetrics struct {
    ActiveAgents       int
    ActiveTasks        int
    MemoryUsage        int64
    CPUUsage           float64
    GoroutineCount     int
    OpenConnections    int
}
```

### Monitoring Implementation

```go
import "github.com/prometheus/client_golang/prometheus"

var (
    taskCounter = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "agenthub_tasks_total",
            Help: "Total number of tasks processed",
        },
        []string{"task_type", "status"},
    )

    latencyHistogram = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "agenthub_task_duration_seconds",
            Help:    "Task processing duration",
            Buckets: prometheus.DefBuckets,
        },
        []string{"task_type"},
    )
)
```

## Scaling Recommendations

### Small Deployments (1-100 agents)
- **Single broker instance**: Sufficient for most small deployments
- **Vertical scaling**: Add CPU/memory as needed
- **Simple monitoring**: Basic logging and health checks

### Medium Deployments (100-1,000 agents)
- **Load balancing**: Implement agent distribution
- **Resource monitoring**: Track CPU, memory, and throughput
- **Optimization**: Tune channel buffer sizes and timeouts

### Large Deployments (1,000+ agents)
- **Horizontal scaling**: Multiple broker instances
- **Partitioning strategy**: Implement agent or task type partitioning
- **Advanced monitoring**: Full metrics and alerting
- **Performance testing**: Regular load testing and optimization

### High-Throughput Scenarios (10,000+ tasks/second)
- **Message batching**: Implement batch processing
- **Connection optimization**: Use connection pooling
- **Hardware optimization**: SSD storage, high-speed networking
- **Profiling**: Regular performance profiling and optimization

## Troubleshooting Performance Issues

### Common Performance Problems

#### 1. High Latency
**Symptoms:** Slow task processing times
**Causes:** Network latency, overloaded agents, inefficient routing
**Solutions:** Optimize routing, add caching, scale horizontally

#### 2. Memory Leaks
**Symptoms:** Increasing memory usage over time
**Causes:** Unclosed channels, goroutine leaks, connection leaks
**Solutions:** Proper cleanup, monitoring, garbage collection tuning

#### 3. Connection Limits
**Symptoms:** New agents can't connect
**Causes:** OS file descriptor limits, broker resource limits
**Solutions:** Increase limits, implement connection pooling

#### 4. Message Loss
**Symptoms:** Tasks not reaching agents or results not returned
**Causes:** Timeout issues, network problems, buffer overflows
**Solutions:** Increase timeouts, improve error handling, adjust buffer sizes

### Performance Testing

#### Load Testing Script
```go
func loadTest() {
    // Create multiple publishers
    publishers := make([]Publisher, 10)
    for i := range publishers {
        publishers[i] = NewPublisher(fmt.Sprintf("publisher_%d", i))
    }

    // Send tasks concurrently
    taskRate := 1000 // tasks per second
    duration := 60 * time.Second

    ticker := time.NewTicker(time.Duration(1e9 / taskRate))
    timeout := time.After(duration)

    for {
        select {
        case <-ticker.C:
            publisher := publishers[rand.Intn(len(publishers))]
            go publisher.PublishTask(generateRandomTask())
        case <-timeout:
            return
        }
    }
}
```

The AgentHub architecture provides solid performance for most use cases and clear scaling paths for growing deployments. Regular monitoring and optimization ensure continued performance as your agent ecosystem evolves.